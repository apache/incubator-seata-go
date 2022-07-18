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

package exec

import (
	"context"

	"github.com/seata/seata-go-datasource/sql/exec"
	"github.com/seata/seata-go-datasource/sql/types"
	"github.com/seata/seata-go/pkg/common/log"
	"go.uber.org/zap"
)

func init() {
	exec.RegisCommonHook(&loggerSQLHook{})
}

type loggerSQLHook struct {
}

func (h *loggerSQLHook) Type() types.SQLType {
	return types.SQLType_Unknown
}

// Before
func (h *loggerSQLHook) Before(ctx context.Context, execCtx *exec.ExecContext) {
	fields := []zap.Field{
		zap.String("tx-id", execCtx.TxCtx.LocalTransID),
		zap.String("sql", execCtx.Query),
	}

	if len(execCtx.NamedValues) != 0 {
		fields = append(fields, zap.Any("namedValues", execCtx.NamedValues))
	}

	if len(execCtx.Values) != 0 {
		fields = append(fields, zap.Any("values", execCtx.Values))
	}

	log.Debug("sql exec log", fields)
}

// After
func (h *loggerSQLHook) After(ctx context.Context, execCtx *exec.ExecContext) {

}
