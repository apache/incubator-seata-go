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
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/constant"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestTransactionMiddleware_WithXidKey(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(constant.XidKey, "test-xid-123")
	c.Request = req

	middleware := TransactionMiddleware()
	middleware(c)

	// Should not abort
	assert.False(t, c.IsAborted())
	// Context should be updated with XID
	assert.NotNil(t, c.Request.Context())
}

func TestTransactionMiddleware_WithXidKeyLowercase(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(constant.XidKeyLowercase, "test-xid-lowercase")
	c.Request = req

	middleware := TransactionMiddleware()
	middleware(c)

	assert.False(t, c.IsAborted())
	assert.NotNil(t, c.Request.Context())
}

func TestTransactionMiddleware_XidKeyPrecedence(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(constant.XidKey, "primary-xid")
	req.Header.Set(constant.XidKeyLowercase, "secondary-xid")
	c.Request = req

	middleware := TransactionMiddleware()
	middleware(c)

	assert.False(t, c.IsAborted())
}

func TestTransactionMiddleware_NoXid(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Request = req

	middleware := TransactionMiddleware()
	middleware(c)

	// Should abort with BadRequest
	assert.True(t, c.IsAborted())
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTransactionMiddleware_EmptyXid(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(constant.XidKey, "")
	c.Request = req

	middleware := TransactionMiddleware()
	middleware(c)

	assert.True(t, c.IsAborted())
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTransactionMiddleware_EmptyPrimaryXid(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(constant.XidKey, "")
	req.Header.Set(constant.XidKeyLowercase, "fallback-xid")
	c.Request = req

	middleware := TransactionMiddleware()
	middleware(c)

	assert.False(t, c.IsAborted())
}

func TestTransactionMiddleware_Integration(t *testing.T) {
	router := gin.New()
	router.Use(TransactionMiddleware())

	router.GET("/api/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	t.Run("with_valid_xid", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		req.Header.Set(constant.XidKey, "integration-xid")

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("without_xid", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestTransactionMiddleware_ContextPropagation(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	originalCtx := context.Background()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req = req.WithContext(originalCtx)
	req.Header.Set(constant.XidKey, "ctx-test-xid")
	c.Request = req

	middleware := TransactionMiddleware()
	middleware(c)

	// Context should be updated
	assert.NotEqual(t, originalCtx, c.Request.Context())
	assert.False(t, c.IsAborted())
}
