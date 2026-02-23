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

package sql

import (
	"database/sql/driver"
	"errors"
	"io"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/mock"
)

/*
 * ------------------------------------------------------------------------
 * Close()
 * ------------------------------------------------------------------------
 */

func TestRowsCommitOnClose_Close(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(ctrl *gomock.Controller) *RowsCommitOnClose
		wantErr   error
		wantState int32
	}{
		{
			name: "rows and tx success",
			setup: func(ctrl *gomock.Controller) *RowsCommitOnClose {
				rows := mock.NewMockTestDriverRows(ctrl)
				tx := mock.NewMockTestDriverTx(ctrl)
				rows.EXPECT().Close().Return(nil)
				tx.EXPECT().Commit().Return(nil)

				return &RowsCommitOnClose{
					rows:  rows,
					tx:    tx,
					state: int32(stateOpen),
				}
			},
			wantErr:   nil,
			wantState: int32(stateClosed),
		},
		{
			name: "rows close error only",
			setup: func(ctrl *gomock.Controller) *RowsCommitOnClose {
				rows := mock.NewMockTestDriverRows(ctrl)
				tx := mock.NewMockTestDriverTx(ctrl)
				rowErr := errors.New("close failed")

				rows.EXPECT().Close().Return(rowErr)
				tx.EXPECT().Commit().Return(nil)

				return &RowsCommitOnClose{
					rows:  rows,
					tx:    tx,
					state: int32(stateOpen),
				}
			},
			wantErr:   errors.New("close failed"),
			wantState: int32(stateClosed),
		},
		{
			name: "tx commit error only",
			setup: func(ctrl *gomock.Controller) *RowsCommitOnClose {
				rows := mock.NewMockTestDriverRows(ctrl)
				tx := mock.NewMockTestDriverTx(ctrl)
				txErr := errors.New("commit failed")

				rows.EXPECT().Close().Return(nil)
				tx.EXPECT().Commit().Return(txErr)

				return &RowsCommitOnClose{
					rows:  rows,
					tx:    tx,
					state: int32(stateOpen),
				}
			},
			wantErr:   errors.New("commit failed"),
			wantState: int32(stateClosed),
		},
		{
			name: "rows and tx both error",
			setup: func(ctrl *gomock.Controller) *RowsCommitOnClose {
				rows := mock.NewMockTestDriverRows(ctrl)
				tx := mock.NewMockTestDriverTx(ctrl)
				rowErr := errors.New("close failed")
				txErr := errors.New("commit failed")

				rows.EXPECT().Close().Return(rowErr)
				tx.EXPECT().Commit().Return(txErr)

				return &RowsCommitOnClose{
					rows:  rows,
					tx:    tx,
					state: int32(stateOpen),
				}
			},
			wantErr:   errors.New("close failed\ncommit failed"),
			wantState: int32(stateClosed),
		},
		{
			name: "already closed",
			setup: func(ctrl *gomock.Controller) *RowsCommitOnClose {
				return &RowsCommitOnClose{
					state: int32(stateClosed),
				}
			},
			wantErr:   nil,
			wantState: int32(stateClosed),
		},
		{
			name: "closing in progress",
			setup: func(ctrl *gomock.Controller) *RowsCommitOnClose {
				return &RowsCommitOnClose{
					state: int32(stateClosing),
				}
			},
			wantErr:   nil,
			wantState: int32(stateClosing),
		},
		{
			name: "nil rows",
			setup: func(ctrl *gomock.Controller) *RowsCommitOnClose {
				tx := mock.NewMockTestDriverTx(ctrl)
				tx.EXPECT().Commit().Return(nil)
				return &RowsCommitOnClose{
					tx:    tx,
					state: int32(stateOpen),
				}
			},
			wantErr:   nil,
			wantState: int32(stateClosed),
		},
		{
			name: "nil tx",
			setup: func(ctrl *gomock.Controller) *RowsCommitOnClose {
				rows := mock.NewMockTestDriverRows(ctrl)
				rows.EXPECT().Close().Return(nil)
				return &RowsCommitOnClose{
					rows:  rows,
					state: int32(stateOpen),
				}
			},
			wantErr:   nil,
			wantState: int32(stateClosed),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			r := tt.setup(ctrl)
			err := r.Close()

			if tt.wantErr == nil {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr.Error(), err.Error())
			}

			assert.Equal(t, tt.wantState, r.state)
			if r.state == int32(stateClosed) {
				assert.Nil(t, r.tx)
			}
		})
	}
}

/*
 * ------------------------------------------------------------------------
 * Columns()
 * ------------------------------------------------------------------------
 */

func TestRowsCommitOnClose_Columns_WithRows(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	rowsMock := mock.NewMockTestDriverRows(ctrl)
	rowsMock.EXPECT().Columns().Return([]string{"id", "name"})

	r := &RowsCommitOnClose{rows: rowsMock}
	cols := r.Columns()

	assert.Equal(t, []string{"id", "name"}, cols)
}

func TestRowsCommitOnClose_Columns_NilRows(t *testing.T) {
	r := &RowsCommitOnClose{}
	assert.Panics(t, func() {
		_ = r.Columns()
	})
}

/*
 * ------------------------------------------------------------------------
 * Next()
 * ------------------------------------------------------------------------
 */

func TestRowsCommitOnClose_Next_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	rowsMock := mock.NewMockTestDriverRows(ctrl)
	dest := make([]driver.Value, 1)
	rowsMock.EXPECT().Next(dest).Return(errors.New("next error"))

	r := &RowsCommitOnClose{rows: rowsMock}
	err := r.Next(dest)

	assert.EqualError(t, err, "next error")
}

func TestRowsCommitOnClose_Next_NilRows(t *testing.T) {
	r := &RowsCommitOnClose{}
	assert.Panics(t, func() {
		_ = r.Next(make([]driver.Value, 1))
	})
}

func TestRowsCommitOnClose_Next_EOF_Close(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	rowsMock := mock.NewMockTestDriverRows(ctrl)
	txMock := mock.NewMockTestDriverTx(ctrl)

	rowsMock.EXPECT().Next(gomock.Any()).Return(io.EOF)
	rowsMock.EXPECT().Close().Return(nil)
	txMock.EXPECT().Commit().Return(nil)

	r := &RowsCommitOnClose{
		rows:  rowsMock,
		tx:    txMock,
		state: int32(stateOpen),
	}

	err := r.Next(make([]driver.Value, 1))
	assert.Equal(t, io.EOF, err)
	assert.Equal(t, int32(stateClosed), r.state)
	assert.Nil(t, r.tx)
}
