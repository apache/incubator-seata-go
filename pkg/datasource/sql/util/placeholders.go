/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package util

import (
	"database/sql/driver"
	"fmt"
	"strconv"
	"strings"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
)

// RewritePlaceholders rewrites internal "?" placeholders for dialects that do
// not use MySQL-style positional placeholders.
func RewritePlaceholders(query string, dbType types.DBType) string {
	if dbType != types.DBTypePostgreSQL || !strings.Contains(query, "?") {
		return query
	}

	var builder strings.Builder
	builder.Grow(len(query) + 8)

	ordinal := 1
	for _, ch := range query {
		if ch != '?' {
			builder.WriteRune(ch)
			continue
		}

		builder.WriteByte('$')
		builder.WriteString(strconv.Itoa(ordinal))
		ordinal++
	}

	return builder.String()
}

// CompactPostgreSQLPlaceholders rewrites sparse PostgreSQL placeholders like
// "$3 AND $5" into dense placeholders like "$1 AND $2" and returns the
// corresponding compacted argument slice.
func CompactPostgreSQLPlaceholders(query string, args []driver.NamedValue) (string, []driver.NamedValue, error) {
	if !strings.Contains(query, "$") {
		return query, args, nil
	}

	var builder strings.Builder
	builder.Grow(len(query))

	placeholderMap := make(map[int]int, len(args))
	compactedArgs := make([]driver.NamedValue, 0, len(args))

	for i := 0; i < len(query); {
		if query[i] != '$' {
			builder.WriteByte(query[i])
			i++
			continue
		}

		j := i + 1
		for j < len(query) && query[j] >= '0' && query[j] <= '9' {
			j++
		}
		if j == i+1 {
			builder.WriteByte(query[i])
			i++
			continue
		}

		oldOrdinal, err := strconv.Atoi(query[i+1 : j])
		if err != nil {
			return "", nil, err
		}
		if oldOrdinal <= 0 || oldOrdinal > len(args) {
			return "", nil, fmt.Errorf("postgres placeholder index %d out of range", oldOrdinal)
		}

		newOrdinal, ok := placeholderMap[oldOrdinal]
		if !ok {
			newOrdinal = len(compactedArgs) + 1
			placeholderMap[oldOrdinal] = newOrdinal

			arg := args[oldOrdinal-1]
			arg.Ordinal = newOrdinal
			arg.Name = ""
			compactedArgs = append(compactedArgs, arg)
		}

		builder.WriteByte('$')
		builder.WriteString(strconv.Itoa(newOrdinal))
		i = j
	}

	return builder.String(), compactedArgs, nil
}
