package channels

import (
	"context"
	"fmt"
	"github.com/go-errors/errors"
	"github.com/jackc/pglogrepl"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/noctarius/timescaledb-event-streamer/internal/pg/decoding"
	"github.com/noctarius/timescaledb-event-streamer/internal/supporting"
	"github.com/noctarius/timescaledb-event-streamer/spi/pgtypes"
	"github.com/noctarius/timescaledb-event-streamer/spi/systemcatalog"
	"github.com/noctarius/timescaledb-event-streamer/spi/version"
	"time"
)

const addTableToPublicationQuery = "ALTER PUBLICATION %s ADD TABLE %s"

const dropTableFromPublicationQuery = "ALTER PUBLICATION %s DROP TABLE %s"

const initialHypertableQuery = `
SELECT h1.id, h1.schema_name, h1.table_name, h1.associated_schema_name, h1.associated_table_prefix,
	 h1.compression_state, h1.compressed_hypertable_id, coalesce(h2.is_distributed, false),
	 ca.user_view_schema, ca.user_view_name
FROM _timescaledb_catalog.hypertable h1
LEFT JOIN timescaledb_information.hypertables h2
	 ON h2.hypertable_schema = h1.schema_name
	AND h2.hypertable_name = h1.table_name
LEFT JOIN _timescaledb_catalog.continuous_agg ca
    ON h1.id = ca.mat_hypertable_id`

const initialChunkQuery = `
SELECT c1.id, c1.hypertable_id, c1.schema_name, c1.table_name, c1.compressed_chunk_id, c1.dropped, c1.status
FROM _timescaledb_catalog.chunk c1
LEFT JOIN timescaledb_information.chunks c2
       ON c2.chunk_schema = c1.schema_name
      AND c2.chunk_name = c1.table_name
LEFT JOIN _timescaledb_catalog.chunk c3 ON c3.compressed_chunk_id = c1.id
LEFT JOIN timescaledb_information.chunks c4
       ON c4.chunk_schema = c3.schema_name
      AND c4.chunk_name = c3.table_name
ORDER BY c1.hypertable_id, coalesce(c2.range_start, c4.range_start)`

const initialTableSchemaQuery = `
SELECT
   c.column_name,
   t.oid::int,
   CASE WHEN c.is_nullable = 'YES' THEN true ELSE false END,
   CASE WHEN p.attname IS NOT NULL THEN true ELSE false END,
   c.column_default
FROM information_schema.columns c
LEFT JOIN pg_catalog.pg_namespace n ON n.nspname = c.udt_schema
LEFT JOIN pg_catalog.pg_type t ON t.typnamespace = n.oid AND t.typname = c.udt_name
LEFT JOIN LATERAL (
    SELECT a.attname
    FROM pg_index i, pg_attribute a, pg_class cl, pg_namespace n
    WHERE cl.relname = c.table_name
      AND n.nspname = c.table_schema
      AND cl.relnamespace = n.oid
      AND a.attrelid = cl.oid
      AND i.indrelid = cl.oid
      AND a.attname = c.column_name
      AND a.attnum = any(i.indkey)
      AND i.indisprimary
) p ON TRUE
WHERE c.table_schema = $1
  AND c.table_name = $2
ORDER BY c.ordinal_position`

const replicaIdentityQuery = `
SELECT c.relreplident
FROM pg_catalog.pg_class c
LEFT JOIN pg_catalog.pg_namespace n ON c.relnamespace = n.oid
WHERE n.nspname=$1 and c.relname=$2
`

const hypertableContinuousAggregateQuery = `
SELECT ca.user_view_schema, ca.user_view_name
FROM _timescaledb_catalog.continuous_agg ca 
WHERE ca.mat_hypertable_id = $1`

const existingPublicationPublishedTablesQuery = `
SELECT pt.schemaname, pt.tablename
FROM pg_catalog.pg_publication_tables pt
WHERE pt.pubname = $1`

const timescaledbVersionQuery = `
SELECT extversion
FROM pg_catalog.pg_extension
WHERE extname = 'timescaledb'`

const postgresqlVersionQuery = `SHOW SERVER_VERSION`

const walLevelQuery = `SHOW WAL_LEVEL`

type sideChannel struct {
	connConfig        *pgx.ConnConfig
	publicationName   string
	snapshotBatchSize int
}

func NewSideChannel(connConfig *pgx.ConnConfig, publicationName string, snapshotBatchSize int) SideChannel {
	sc := &sideChannel{
		connConfig:        connConfig,
		publicationName:   publicationName,
		snapshotBatchSize: snapshotBatchSize,
	}
	return sc
}

func (sc *sideChannel) GetWalLevel() (walLevel string, err error) {
	walLevel = "unknown"
	err = sc.newSession(time.Second*10, func(session *session) error {
		return session.queryRow(walLevelQuery).Scan(&walLevel)
	})
	return
}

func (sc *sideChannel) GetPostgresVersion() (pgVersion version.PostgresVersion, err error) {
	if err = sc.newSession(time.Second*10, func(session *session) error {
		var v string
		if err := session.queryRow(postgresqlVersionQuery).Scan(&v); err != nil {
			return err
		}
		pgVersion, err = version.ParsePostgresVersion(v)
		return nil
	}); err != nil {
		return
	}
	return
}

func (sc *sideChannel) GetTimescaleDBVersion() (tsdbVersion version.TimescaleVersion, err error) {
	if err = sc.newSession(time.Second*10, func(session *session) error {
		var v string
		if err := session.queryRow(timescaledbVersionQuery).Scan(&v); err != nil {
			return err
		}
		tsdbVersion, err = version.ParseTimescaleVersion(v)
		return nil
	}); err != nil {
		return
	}
	return
}

func (sc *sideChannel) ReadHypertables(cb func(hypertable *systemcatalog.Hypertable) error) error {
	return sc.newSession(time.Second*20, func(session *session) error {
		return session.queryFunc(func(row pgx.Row) error {
			var id int32
			var schemaName, hypertableName, associatedSchemaName, associatedTablePrefix string
			var compressionState int16
			var compressedHypertableId *int32
			var distributed bool
			var viewSchema, viewName *string

			if err := row.Scan(&id, &schemaName, &hypertableName, &associatedSchemaName,
				&associatedTablePrefix, &compressionState, &compressedHypertableId,
				&distributed, &viewSchema, &viewName); err != nil {

				return errors.Wrap(err, 0)
			}

			hypertable := systemcatalog.NewHypertable(id, sc.connConfig.Database, schemaName, hypertableName,
				associatedSchemaName, associatedTablePrefix, compressedHypertableId, compressionState,
				distributed, viewSchema, viewName)

			return cb(hypertable)
		}, initialHypertableQuery)
	})
}

func (sc *sideChannel) ReadChunks(cb func(chunk *systemcatalog.Chunk) error) error {
	return sc.newSession(time.Second*20, func(session *session) error {
		return session.queryFunc(func(row pgx.Row) error {
			var id, hypertableId int32
			var schemaName, tableName string
			var compressedChunkId *int32
			var dropped bool
			var status int32

			if err := row.Scan(&id, &hypertableId, &schemaName, &tableName,
				&compressedChunkId, &dropped, &status); err != nil {
				return errors.Wrap(err, 0)
			}

			return cb(
				systemcatalog.NewChunk(id, hypertableId, schemaName, tableName, dropped, status, compressedChunkId),
			)
		}, initialChunkQuery)
	})
}

func (sc *sideChannel) ReadHypertableSchema(
	cb func(hypertable *systemcatalog.Hypertable, columns []systemcatalog.Column) bool,
	hypertables ...*systemcatalog.Hypertable) error {

	return sc.newSession(time.Second*10, func(session *session) error {
		for _, hypertable := range hypertables {
			if (!hypertable.IsContinuousAggregate() && hypertable.SchemaName() == "_timescaledb_internal") ||
				hypertable.SchemaName() == "_timescaledb_catalog" {

				continue
			}
			if err := sc.readHypertableSchema(session, hypertable, cb); err != nil {
				return err
			}
		}
		return nil
	})
}

func (sc *sideChannel) AttachChunkToPublication(chunk *systemcatalog.Chunk) error {
	canonicalChunkName := chunk.CanonicalName()
	attachingQuery := fmt.Sprintf(addTableToPublicationQuery, sc.publicationName, canonicalChunkName)
	return sc.newSession(time.Second*20, func(session *session) error {
		if _, err := session.exec(attachingQuery); err != nil {
			return errors.Wrap(err, 0)
		}
		logger.Infoln("Updated publication %s to add table %s", sc.publicationName, canonicalChunkName)
		return nil
	})
}

func (sc *sideChannel) DetachChunkFromPublication(chunk *systemcatalog.Chunk) error {
	canonicalChunkName := chunk.CanonicalName()
	detachingQuery := fmt.Sprintf(dropTableFromPublicationQuery, sc.publicationName, canonicalChunkName)
	return sc.newSession(time.Second*20, func(session *session) error {
		if _, err := session.exec(detachingQuery); err != nil {
			return errors.Wrap(err, 0)
		}
		logger.Infof("Updated publication %s to drop table %s", sc.publicationName, canonicalChunkName)
		return nil
	})
}

func (sc *sideChannel) SnapshotTable(canonicalName string, startingLSN *pglogrepl.LSN,
	cb func(lsn pglogrepl.LSN, values map[string]any) error) (pglogrepl.LSN, error) {

	var currentLSN pglogrepl.LSN

	err := sc.newSession(time.Minute*60, func(session *session) error {
		if _, err := session.exec("BEGIN TRANSACTION ISOLATION LEVEL REPEATABLE READ"); err != nil {
			return err
		}

		if err := session.queryRow("SELECT pg_current_wal_lsn()").Scan(&currentLSN); err != nil {
			return err
		}

		if startingLSN != nil {
			if _, err := session.exec(
				fmt.Sprintf("SET TRANSACTION SNAPSHOT '%s'", startingLSN.String()),
			); err != nil {
				return err
			}
		}

		cursorName := supporting.RandomTextString(15)
		if _, err := session.exec(
			fmt.Sprintf("DECLARE %s SCROLL CURSOR FOR SELECT * FROM %s", cursorName, canonicalName),
		); err != nil {
			return errors.Wrap(err, 0)
		}

		var rowDecoder *decoding.RowDecoder
		for {
			count := 0
			if err := session.queryFunc(func(row pgx.Row) error {
				rows := row.(pgx.Rows)

				if rowDecoder == nil {
					rd, err := decoding.NewRowDecoder(rows.FieldDescriptions())
					if err != nil {
						return err
					}
					rowDecoder = rd
				}

				return rowDecoder.DecodeMapAndSink(rows.RawValues(), func(values map[string]any) error {
					count++
					return cb(currentLSN, values)
				})
			}, fmt.Sprintf("FETCH FORWARD %d FROM %s", sc.snapshotBatchSize, cursorName)); err != nil {
				return errors.Wrap(err, 0)
			}
			if count == 0 || count < sc.snapshotBatchSize {
				break
			}
			session.ResetTimeout(time.Minute * 60)
		}

		_, err := session.exec(fmt.Sprintf("CLOSE %s", cursorName))
		if err != nil {
			return errors.Wrap(err, 0)
		}

		_, err = session.exec("ROLLBACK")
		if err != nil {
			return errors.Wrap(err, 0)
		}
		return nil
	})
	if err != nil {
		return 0, errors.Wrap(err, 0)
	}

	return currentLSN, nil
}

func (sc *sideChannel) ReadReplicaIdentity(schemaName, tableName string) (pgtypes.ReplicaIdentity, error) {
	var replicaIdentity pgtypes.ReplicaIdentity
	if err := sc.newSession(time.Second*10, func(session *session) error {
		row := session.queryRow(replicaIdentityQuery, schemaName, tableName)

		var val string
		if err := row.Scan(&val); err != nil {
			return err
		}
		replicaIdentity = pgtypes.AsReplicaIdentity(val)
		return nil
	}); err != nil {
		return pgtypes.UNKNOWN, err
	}
	return replicaIdentity, nil
}

func (sc *sideChannel) ReadContinuousAggregate(materializedHypertableId int32) (string, string, bool, error) {
	var viewSchema, viewName string

	found := false
	if err := sc.newSession(time.Second*10, func(session *session) error {
		row := session.queryRow(hypertableContinuousAggregateQuery, materializedHypertableId)
		if err := row.Scan(&viewSchema, &viewName); err != nil {
			if err != pgx.ErrNoRows {
				return err
			}
		} else {
			found = true
		}
		return nil
	}); err != nil {
		return "", "", false, err
	}
	return viewSchema, viewName, found, nil
}

func (sc *sideChannel) ReadPublishedTables(publicationName string) ([]string, error) {
	tableNames := make([]string, 0)
	if err := sc.newSession(time.Second*20, func(session *session) error {
		return session.queryFunc(func(row pgx.Row) error {
			var schemaName, tableName string
			if err := row.Scan(&schemaName, &tableName); err != nil {
				return err
			}
			tableNames = append(tableNames, systemcatalog.MakeRelationKey(schemaName, tableName))
			return nil
		}, existingPublicationPublishedTablesQuery, publicationName)
	}); err != nil {
		return nil, err
	}
	return tableNames, nil
}

func (sc *sideChannel) readHypertableSchema(
	session *session, hypertable *systemcatalog.Hypertable,
	cb func(hypertable *systemcatalog.Hypertable, columns []systemcatalog.Column) bool) error {

	columns := make([]systemcatalog.Column, 0)
	if err := session.queryFunc(func(row pgx.Row) error {
		var name string
		var oid uint32
		var nullable, primaryKey bool
		var defaultValue *string

		if err := row.Scan(&name, &oid, &nullable, &primaryKey, &defaultValue); err != nil {
			return errors.Wrap(err, 0)
		}

		dataType, err := systemcatalog.DataTypeByOID(oid)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		column := systemcatalog.NewColumn(name, oid, string(dataType), nullable, primaryKey, defaultValue)
		columns = append(columns, column)
		return nil
	}, initialTableSchemaQuery, hypertable.SchemaName(), hypertable.HypertableName()); err != nil {
		return errors.Wrap(err, 0)
	}

	if !cb(hypertable, columns) {
		return errors.Errorf("hypertable schema callback failed")
	}

	return nil
}

func (sc *sideChannel) newSession(timeout time.Duration, fn func(session *session) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	connection, err := pgx.ConnectConfig(ctx, sc.connConfig)
	if err != nil {
		return fmt.Errorf("unable to connect to database: %v", err)
	}
	defer connection.Close(context.Background())

	s := &session{
		connection: connection,
		ctx:        ctx,
		cancel:     cancel,
	}

	defer func() {
		s.cancel()
	}()

	return fn(s)
}

type rowFunction = func(row pgx.Row) error

type session struct {
	connection *pgx.Conn
	ctx        context.Context
	cancel     func()
}

func (s *session) ResetTimeout(timeout time.Duration) {
	// Cancel old context
	s.cancel()

	// Initialize new timeout
	s.ctx, s.cancel = context.WithTimeout(context.Background(), timeout)
}

func (s *session) queryFunc(fn rowFunction, query string, args ...any) error {
	rows, err := s.connection.Query(s.ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		if err := fn(rows); err != nil {
			return err
		}
	}

	return rows.Err()
}

func (s *session) queryRow(query string, args ...any) pgx.Row {
	return s.connection.QueryRow(s.ctx, query, args...)
}

func (s *session) exec(query string, args ...any) (pgconn.CommandTag, error) {
	return s.connection.Exec(s.ctx, query, args...)
}
