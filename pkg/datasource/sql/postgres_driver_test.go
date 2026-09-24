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
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/mock"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
)

func TestParseConnectorMetadataPostgres(t *testing.T) {
	meta, err := parseConnectorMetadata("postgres://postgres:password@127.0.0.1:5432/seata_demo?sslmode=disable", types.DBTypePostgreSQL)
	assert.NoError(t, err)
	assert.Equal(t, "seata_demo", meta.dbName)
}

func TestSeataConnectorConnectPostgres(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConn := mock.NewMockTestDriverConn(ctrl)
	mockConnector := mock.NewMockTestDriverConnector(ctrl)
	mockConnector.EXPECT().Connect(gomock.Any()).Return(mockConn, nil)

	connector := &seataConnector{
		target: mockConnector,
		res: &DBResource{
			dbType: types.DBTypePostgreSQL,
		},
		dbName: "seata_demo",
		dbType: types.DBTypePostgreSQL,
	}

	conn, err := connector.Connect(context.Background())
	assert.NoError(t, err)

	got, ok := conn.(*Conn)
	assert.True(t, ok)
	assert.Equal(t, "seata_demo", got.dbName)
	assert.Equal(t, types.DBTypePostgreSQL, got.dbType)
}

func TestSeataXAConnectorConnectPostgres(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConn := mock.NewMockTestDriverConn(ctrl)
	mockConnector := mock.NewMockTestDriverConnector(ctrl)
	mockConnector.EXPECT().Connect(gomock.Any()).Return(mockConn, nil)

	connector := &seataXAConnector{
		seataConnector: &seataConnector{
			target: mockConnector,
			res: &DBResource{
				dbType: types.DBTypePostgreSQL,
			},
			dbName: "seata_demo",
			dbType: types.DBTypePostgreSQL,
		},
	}

	conn, err := connector.Connect(context.Background())
	assert.NoError(t, err)

	xaConn, ok := conn.(*XAConn)
	assert.True(t, ok)
	assert.Equal(t, "seata_demo", xaConn.dbName)
	assert.Equal(t, types.DBTypePostgreSQL, xaConn.dbType)
	assert.True(t, xaConn.txCtx.TransactionMode == types.Local)
}

func TestSeataConnectorDriverPreservesTargetName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDriver := mock.NewMockTestDriver(ctrl)
	mockConnector := mock.NewMockTestDriverConnector(ctrl)

	connector := &seataConnector{
		target:       mockConnector,
		targetDriver: mockDriver,
		targetName:   "pgx",
		transType:    types.XAMode,
	}

	got, ok := connector.Driver().(*seataDriver)
	assert.True(t, ok)
	assert.Equal(t, "pgx", got.getTargetDriverName())
}
