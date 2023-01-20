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

package main

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/seata/seata-go/pkg/client"
	ginmiddleware "github.com/seata/seata-go/pkg/integration/gin"
	"github.com/seata/seata-go/pkg/util/log"
)

func main() {
	client.InitPath("./sample/conf/seatago.yml")
	initService()

	r := gin.Default()

	// NOTE: when use gin，must set ContextWithFallback true when gin version >= 1.8.1
	// r.ContextWithFallback = true

	r.Use(ginmiddleware.TransactionMiddleware())

	r.POST("/updateDataSuccess", func(c *gin.Context) {
		log.Infof("get tm updateData")
		if err := updateDataSuccess(c); err != nil {
			c.JSON(http.StatusBadRequest, "updateData failure")
			return
		}
		c.JSON(http.StatusOK, "updateData ok")
	})

	if err := r.Run(":8080"); err != nil {
		log.Fatalf("start tcc server fatal: %v", err)
	}
}
