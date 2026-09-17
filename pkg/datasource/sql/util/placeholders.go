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
	for i := 0; i < len(query); {
		if next, ok := copyPostgreSQLNonCode(query, i, &builder); ok {
			i = next
			continue
		}

		if query[i] == '?' {
			builder.WriteByte('$')
			builder.WriteString(strconv.Itoa(ordinal))
			ordinal++
		} else {
			builder.WriteByte(query[i])
		}
		i++
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
		if next, ok := copyPostgreSQLNonCode(query, i, &builder); ok {
			i = next
			continue
		}

		if query[i] != '$' || (i > 0 && isPostgreSQLIdentifierPart(query[i-1])) {
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

// StripPostgreSQLStringCharset removes MySQL charset introducers emitted by the
// parser for string literals. PostgreSQL does not accept _UTF8MB4'...', and the
// following placeholder compaction must still see the literal quotes.
func StripPostgreSQLStringCharset(query string) string {
	const utf8mb4 = "_UTF8MB4"
	found := false
	for i := 0; i+len(utf8mb4) < len(query); i++ {
		if hasPostgreSQLUTF8MB4StringIntroducer(query, i) {
			found = true
			break
		}
	}
	if !found {
		return query
	}

	var builder strings.Builder
	builder.Grow(len(query))

	for i := 0; i < len(query); {
		if next, ok := copyPostgreSQLNonCode(query, i, &builder); ok {
			i = next
			continue
		}

		if hasPostgreSQLUTF8MB4StringIntroducer(query, i) {
			i += len(utf8mb4)
			continue
		}

		builder.WriteByte(query[i])
		i++
	}

	return builder.String()
}

func hasPostgreSQLUTF8MB4StringIntroducer(query string, start int) bool {
	const utf8mb4 = "_UTF8MB4"
	end := start + len(utf8mb4)
	return end < len(query) &&
		query[end] == '\'' &&
		(start == 0 || !isPostgreSQLIdentifierPart(query[start-1])) &&
		strings.EqualFold(query[start:end], utf8mb4)
}

// copyPostgreSQLNonCode copies a quoted string, quoted identifier, comment,
// or dollar-quoted string starting at start. Placeholders inside these regions
// are SQL text and must not be rewritten.
func copyPostgreSQLNonCode(query string, start int, builder *strings.Builder) (int, bool) {
	if start >= len(query) {
		return start, false
	}

	var end int
	switch query[start] {
	case '\'':
		end = scanPostgreSQLQuoted(query, start, '\'')
	case '"':
		end = scanPostgreSQLQuoted(query, start, '"')
	case '-':
		if start+1 >= len(query) || query[start+1] != '-' {
			return start, false
		}
		end = start + 2
		for end < len(query) && query[end] != '\n' && query[end] != '\r' {
			end++
		}
	case '/':
		if start+1 >= len(query) || query[start+1] != '*' {
			return start, false
		}
		end = scanPostgreSQLBlockComment(query, start)
	case '$':
		delimiter, ok := postgreSQLDollarQuoteDelimiter(query, start)
		if !ok {
			return start, false
		}
		contentStart := start + len(delimiter)
		closing := strings.Index(query[contentStart:], delimiter)
		if closing < 0 {
			end = len(query)
		} else {
			end = contentStart + closing + len(delimiter)
		}
	default:
		return start, false
	}

	builder.WriteString(query[start:end])
	return end, true
}

func scanPostgreSQLQuoted(query string, start int, quote byte) int {
	backslashEscapes := hasPostgreSQLBackslashEscapePrefix(query, start)
	for i := start + 1; i < len(query); i++ {
		if backslashEscapes && query[i] == '\\' && i+1 < len(query) {
			i++
			continue
		}
		if query[i] != quote {
			continue
		}
		if i+1 < len(query) && query[i+1] == quote {
			i++
			continue
		}
		return i + 1
	}
	return len(query)
}

func hasPostgreSQLBackslashEscapePrefix(query string, quoteStart int) bool {
	if quoteStart > 0 && (query[quoteStart-1] == 'e' || query[quoteStart-1] == 'E') &&
		(quoteStart == 1 || !isPostgreSQLIdentifierPart(query[quoteStart-2])) {
		return true
	}
	return quoteStart > 1 && query[quoteStart-1] == '&' &&
		(query[quoteStart-2] == 'u' || query[quoteStart-2] == 'U') &&
		(quoteStart == 2 || !isPostgreSQLIdentifierPart(query[quoteStart-3]))
}

func scanPostgreSQLBlockComment(query string, start int) int {
	depth := 1
	for i := start + 2; i < len(query); {
		switch {
		case i+1 < len(query) && query[i] == '/' && query[i+1] == '*':
			depth++
			i += 2
		case i+1 < len(query) && query[i] == '*' && query[i+1] == '/':
			depth--
			i += 2
			if depth == 0 {
				return i
			}
		default:
			i++
		}
	}
	return len(query)
}

func postgreSQLDollarQuoteDelimiter(query string, start int) (string, bool) {
	if start+1 >= len(query) || query[start] != '$' {
		return "", false
	}
	if query[start+1] == '$' {
		return "$$", true
	}
	if !isPostgreSQLIdentifierStart(query[start+1]) {
		return "", false
	}

	i := start + 2
	for i < len(query) && isPostgreSQLDollarQuoteTagPart(query[i]) {
		i++
	}
	if i >= len(query) || query[i] != '$' {
		return "", false
	}
	return query[start : i+1], true
}

func isPostgreSQLIdentifierStart(ch byte) bool {
	return ch == '_' || ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= 0x80
}

func isPostgreSQLIdentifierPart(ch byte) bool {
	return isPostgreSQLIdentifierStart(ch) || ch >= '0' && ch <= '9' || ch == '$'
}

func isPostgreSQLDollarQuoteTagPart(ch byte) bool {
	return isPostgreSQLIdentifierStart(ch) || ch >= '0' && ch <= '9'
}
