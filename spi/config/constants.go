package config

const (
	PropertyPostgresqlConnection              = "postgresql.connection"
	PropertyPostgresqlPassword                = "postgresql.password"
	PropertyPostgresqlPublicationName         = "postgresql.publication.name"
	PropertyPostgresqlPublicationCreate       = "postgresql.publication.create"
	PropertyPostgresqlPublicationAutoDrop     = "postgresql.publication.autodrop"
	PropertyPostgresqlSnapshotBatchsize       = "postgresql.snapshot.batchsize"
	PropertyPostgresqlReplicationSlotName     = "postgresql.replicationslot.name"
	PropertyPostgresqlReplicationSlotCreate   = "postgresql.replicationslot.create"
	PropertyPostgresqlReplicationSlotAutoDrop = "postgresql.replicationslot.autodrop"
	PropertyPostgresqlTxwindowEnabled         = "postgresql.transaction.window.enabled"
	PropertyPostgresqlTxwindowTimeout         = "postgresql.transaction.window.timeout"
	PropertyPostgresqlTxwindowMaxsize         = "postgresql.transaction.window.maxsize"

	PropertySink          = "sink.type"
	PropertySinkTombstone = "sink.tombstone"

	PropertyStateStorageType     = "statestorage.type"
	PropertyFileStateStoragePath = "statestorage.file.path"

	PropertyEventsRead          = "timescaledb.events.read"
	PropertyEventsInsert        = "timescaledb.events.insert"
	PropertyEventsUpdate        = "timescaledb.events.update"
	PropertyEventsDelete        = "timescaledb.events.delete"
	PropertyEventsTruncate      = "timescaledb.events.truncate"
	PropertyEventsMessage       = "timescaledb.events.message"
	PropertyEventsCompression   = "timescaledb.events.compression"
	PropertyEventsDecompression = "timescaledb.events.decompression"

	PropertyNamingStrategy = "topic.namingstrategy.type"

	PropertyKafkaBrokers       = "sink.kafka.brokers"
	PropertyKafkaSaslEnabled   = "sink.kafka.sasl.enabled"
	PropertyKafkaSaslUser      = "sink.kafka.sasl.user"
	PropertyKafkaSaslPassword  = "sink.kafka.sasl.password"
	PropertyKafkaSaslMechanism = "sink.kafka.sasl.mechanism"
	PropertyKafkaTlsEnabled    = "sink.kafka.tls.enabled"
	PropertyKafkaTlsSkipVerify = "sink.kafka.tls.skipverify"
	PropertyKafkaTlsClientAuth = "sink.kafka.tls.clientauth"

	PropertyNatsAddress                = "sink.nats.address"
	PropertyNatsAuthorization          = "sink.nats.authorization"
	PropertyNatsUserinfoUsername       = "sink.nats.userinfo.username"
	PropertyNatsUserinfoPassword       = "sink.nats.userinfo.password"
	PropertyNatsCredentialsCertificate = "sink.nats.credentials.certificate"
	PropertyNatsCredentialsSeeds       = "sink.nats.credentials.seeds"
	PropertyNatsJwt                    = "sink.nats.jwt.jwt"
	PropertyNatsJwtSeed                = "sink.nats.jwt.seed"

	PropertyRedisNetwork           = "sink.redis.network"
	PropertyRedisAddress           = "sink.redis.address"
	PropertyRedisPassword          = "sink.redis.password"
	PropertyRedisDatabase          = "sink.redis.database"
	PropertyRedisPoolsize          = "sink.redis.poolsize"
	PropertyRedisRetriesMax        = "sink.redis.retries.maxattempts"
	PropertyRedisRetriesBackoffMin = "sink.redis.retries.backoff.min"
	PropertyRedisRetriesBackoffMax = "sink.redis.retries.backoff.max"
	PropertyRedisTimeoutDial       = "sink.redis.timeouts.dial"
	PropertyRedisTimeoutRead       = "sink.redis.timeouts.read"
	PropertyRedisTimeoutWrite      = "sink.redis.timeouts.write"
	PropertyRedisTimeoutPool       = "sink.redis.timeouts.pool"
	PropertyRedisTimeoutIdle       = "sink.redis.timeouts.idle"
	PropertyRedisTlsSkipVerify     = "sink.redis.tls.skipverify"
	PropertyRedisTlsClientAuth     = "sink.redis.tls.clientauth"

	PropertyKinesisStreamName         = "sink.kinesis.stream.name"
	PropertyKinesisStreamCreate       = "sink.kinesis.stream.create"
	PropertyKinesisStreamShardCount   = "sink.kinesis.stream.shardcount"
	PropertyKinesisStreamMode         = "sink.kinesis.stream.mode"
	PropertyKinesisRegion             = "sink.kinesis.region"
	PropertyKinesisAwsEndpoint        = "sink.kinesis.aws.endpoint"
	PropertyKinesisAwsAccessKeyId     = "sink.kinesis.aws.accesskeyid"
	PropertyKinesisAwsSecretAccessKey = "sink.kinesis.aws.secretaccesskey"
	PropertyKinesisAwsSessionToken    = "sink.kinesis.aws.sessiontoken"
)
