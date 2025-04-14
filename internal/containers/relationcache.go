/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements. See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License. You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package containers

type RelationCache[V any] struct {
	cache map[uint32]V
}

func NewRelationCache[V any]() *RelationCache[V] {
	return &RelationCache[V]{
		cache: make(map[uint32]V),
	}
}

func (rc *RelationCache[V]) Get(
	oid uint32,
) (value V, present bool) {
	v, ok := rc.cache[oid]
	return v, ok
}

func (rc *RelationCache[V]) Set(
	oid uint32, value V,
) {
	rc.cache[oid] = value
}
