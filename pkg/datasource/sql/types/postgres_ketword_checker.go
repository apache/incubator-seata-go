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

package types

var PostgresKeyWord map[string]string

func GetPostgresKeyWord() map[string]string {
	if PostgresKeyWord == nil {
		PostgresKeyWord = map[string]string{
			"SELECT":    "SELECT",
			"INSERT":    "INSERT",
			"UPDATE":    "UPDATE",
			"DELETE":    "DELETE",
			"FROM":      "FROM",
			"WHERE":     "WHERE",
			"AND":       "AND",
			"OR":        "OR",
			"NOT":       "NOT",
			"IN":        "IN",
			"NOT IN":    "NOT IN",
			"BETWEEN":   "BETWEEN",
			"LIKE":      "LIKE",
			"AS":        "AS",
			"JOIN":      "JOIN",
			"INNER":     "INNER",
			"LEFT":      "LEFT",
			"RIGHT":     "RIGHT",
			"FULL":      "FULL",
			"ON":        "ON",
			"GROUP BY":  "GROUP BY",
			"HAVING":    "HAVING",
			"ORDER":     "ORDER",
			"ORDER BY":  "ORDER BY",
			"LIMIT":     "LIMIT",
			"OFFSET":    "OFFSET",
			"DISTINCT":  "DISTINCT",
			"ALL":       "ALL",
			"ANY":       "ANY",
			"SOME":      "SOME",
			"EXISTS":    "EXISTS",
			"IS":        "IS",
			"NULL":      "NULL",
			"TRUE":      "TRUE",
			"FALSE":     "FALSE",
			"DEFAULT":   "DEFAULT",
			"VALUES":    "VALUES",
			"RETURNING": "RETURNING",

			"INT":       "INT",
			"INTEGER":   "INTEGER",
			"BIGINT":    "BIGINT",
			"SMALLINT":  "SMALLINT",
			"TINYINT":   "TINYINT",
			"BOOLEAN":   "BOOLEAN",
			"VARCHAR":   "VARCHAR",
			"CHAR":      "CHAR",
			"TEXT":      "TEXT",
			"BLOB":      "BLOB",
			"DATE":      "DATE",
			"TIME":      "TIME",
			"TIMESTAMP": "TIMESTAMP",
			"NUMERIC":   "NUMERIC",
			"DECIMAL":   "DECIMAL",
			"FLOAT":     "FLOAT",
			"DOUBLE":    "DOUBLE",

			"SERIAL":      "SERIAL",
			"BIGSERIAL":   "BIGSERIAL",
			"UUID":        "UUID",
			"JSON":        "JSON",
			"JSONB":       "JSONB",
			"ARRAY":       "ARRAY",
			"HSTORE":      "HSTORE",
			"ENUM":        "ENUM",
			"RANGE":       "RANGE",
			"INTERVAL":    "INTERVAL",
			"CASCADE":     "CASCADE",
			"RESTRICT":    "RESTRICT",
			"SET NULL":    "SET NULL",
			"SET DEFAULT": "SET DEFAULT",
			"TABLE":       "TABLE",
			"VIEW":        "VIEW",
			"INDEX":       "INDEX",
			"CONSTRAINT":  "CONSTRAINT",
			"PRIMARY":     "PRIMARY",
			"FOREIGN":     "FOREIGN",
			"KEY":         "KEY",
			"UNIQUE":      "UNIQUE",
			"CHECK":       "CHECK",
			"REFERENCES":  "REFERENCES",
			"CREATE":      "CREATE",
			"ALTER":       "ALTER",
			"DROP":        "DROP",
			"TRUNCATE":    "TRUNCATE",
			"COMMIT":      "COMMIT",
			"ROLLBACK":    "ROLLBACK",
			"SAVEPOINT":   "SAVEPOINT",
			"LOCK":        "LOCK",
			"GRANT":       "GRANT",
			"REVOKE":      "REVOKE",
			"USER":        "USER",
			"ROLE":        "ROLE",
			"SCHEMA":      "SCHEMA",
			"EXTENSION":   "EXTENSION",
			"FUNCTION":    "FUNCTION",
			"PROCEDURE":   "PROCEDURE",
			"TRIGGER":     "TRIGGER",
			"EVENT":       "EVENT",
		}
	}
	return PostgresKeyWord
}
