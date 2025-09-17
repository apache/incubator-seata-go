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

// pkg/datasource/sql/mock_rows.go
// pkg/datasource/sql/mock_rows.go
package mock

import (
	"database/sql"
	"database/sql/driver"
	"errors"
)

type GenericMockRows struct {
	ColumnNames []string
	Data        [][]driver.Value
	idx         int
	err         error
}

func NewGenericMockRows(columns []string, data [][]driver.Value) *GenericMockRows {
	return &GenericMockRows{
		ColumnNames: columns,
		Data:        data,
		idx:         0,
		err:         nil,
	}
}

func (m *GenericMockRows) Columns() []string {
	return m.ColumnNames
}

func (m *GenericMockRows) Close() error {
	m.idx = 0
	m.err = nil
	return nil
}

func (m *GenericMockRows) Next(dest []driver.Value) error {
	if m.err != nil {
		return m.err
	}

	if m.idx >= len(m.Data) {
		return sql.ErrNoRows
	}

	currentRow := m.Data[m.idx]
	if len(dest) < len(currentRow) {
		m.err = errors.New("dest length is less than row data length")
		return m.err
	}
	copy(dest, currentRow)

	m.idx++
	return nil
}

func (m *GenericMockRows) Err() error {
	return m.err
}

type mysqlMockRows struct {
	idx  int
	data [][]interface{}
}

func (m *mysqlMockRows) Columns() []string {
	return []string{"VERSION()"}
}

func (m *mysqlMockRows) Close() error {
	return nil
}

func (m *mysqlMockRows) Next(dest []driver.Value) error {
	if m.idx >= len(m.data) {
		return sql.ErrNoRows
	}

	currentRow := m.data[m.idx]
	copyLength := len(currentRow)
	if len(dest) < copyLength {
		copyLength = len(dest)
	}

	for i := 0; i < copyLength; i++ {
		dest[i] = currentRow[i]
	}

	m.idx++
	return nil
}

func (m *mysqlMockRows) Err() error {
	return nil
}

type pgMockRows struct {
	idx  int
	data [][]interface{}
}

func (p *pgMockRows) Columns() []string {
	return []string{"version"}
}

func (p *pgMockRows) Close() error {
	return nil
}

func (p *pgMockRows) Next(dest []driver.Value) error {
	if p.idx >= len(p.data) {
		return sql.ErrNoRows
	}

	currentRow := p.data[p.idx]
	copyLength := len(currentRow)
	if len(dest) < copyLength {
		copyLength = len(dest)
	}

	for i := 0; i < copyLength; i++ {
		dest[i] = currentRow[i]
	}

	p.idx++
	return nil
}

func (p *pgMockRows) Err() error {
	return nil
}
