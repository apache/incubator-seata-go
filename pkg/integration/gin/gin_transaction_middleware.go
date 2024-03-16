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

package gin

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"seata.apache.org/seata-go/pkg/constant"
	"seata.apache.org/seata-go/pkg/tm"
	"seata.apache.org/seata-go/pkg/util/log"
)

// TransactionMiddleware filter gin invocation
// NOTE: when use gin，must set gin.ContextWithFallback true when gin version >= 1.8.1
func TransactionMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		xid := ctx.GetHeader(constant.XidKey)
		if xid == "" {
			xid = ctx.GetHeader(constant.XidKeyLowercase)
		}

		if len(xid) == 0 {
			log.Errorf("Gin: header not contain header: %s, global transaction xid", constant.XidKey)
			ctx.AbortWithStatus(http.StatusBadRequest)
			return
		}

		newCtx := ctx.Request.Context()
		newCtx = tm.InitSeataContext(newCtx)
		tm.SetXID(newCtx, xid)
		ctx.Request = ctx.Request.WithContext(newCtx)

		log.Infof("global transaction xid is :%s", xid)
	}
}
