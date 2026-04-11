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

package xa

import (
	"context"
	"database/sql/driver"
	"fmt"
	"time"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
)

const (
	// TMEndRScan ends a recovery scan.
	TMEndRScan = 0x00800000
	// TMFail disassociates the caller and marks the transaction branch
	// rollback-only.
	TMFail = 0x20000000

	// TMJoin joining existing transaction branch.
	TMJoin = 0x00200000

	// TMNoFlags indicate no flags value is selected.
	TMNoFlags = 0x00000000

	// TMOnePhase using one-phase optimization.
	TMOnePhase = 0x40000000

	// TMResume is resuming association with a suspended transaction branch.
	TMResume = 0x08000000

	// TMStartRScan starts a recovery scan.
	TMStartRScan = 0x01000000

	// TMSuccess disassociates caller from a transaction branch.
	TMSuccess = 0x04000000

	// TMSuspend is suspending (not ending) its association with a transaction branch.
	TMSuspend = 0x02000000

	// XAReadOnly the transaction branch has been read-only and has been committed.
	XAReadOnly = 0x00000003

	// XAOk The transaction work has been prepared normally.
	XAOk = 0
)

// XAResource defines the contract for XA transaction operations across different databases.
// Each database (MySQL, PostgreSQL, Oracle, etc.) provides its own implementation.
type XAResource interface {
	Commit(ctx context.Context, xid string, onePhase bool) error
	End(ctx context.Context, xid string, flags int) error
	Forget(ctx context.Context, xid string) error
	GetTransactionTimeout() time.Duration
	IsSameRM(ctx context.Context, resource XAResource) bool
	XAPrepare(ctx context.Context, xid string) error
	Recover(ctx context.Context, flag int) ([]string, error)
	Rollback(ctx context.Context, xid string) error
	SetTransactionTimeout(duration time.Duration) bool
	Start(ctx context.Context, xid string, flags int) error
}

// XAErrorClassifier abstracts database-specific XA error classification.
// This allows the upper layer (conn_xa.go) to handle XA errors without
// importing database-specific driver packages.
type XAErrorClassifier interface {
	// IsAlreadyEnded checks if the error indicates the XA branch is already ended.
	// For MySQL: XAER_RMFAIL with IDLE state (error 1399).
	// For PostgreSQL: transaction already committed/rolled back.
	// For Oracle: ORA-24756 (transaction does not exist).
	IsAlreadyEnded(err error) bool
}

// defaultErrorClassifier is a no-op classifier that never matches any error.
// Used as a fallback when no database-specific classifier is registered.
type defaultErrorClassifier struct{}

func (c *defaultErrorClassifier) IsAlreadyEnded(err error) bool { return false }

// XAResourceFactory creates database-specific XA resources and error classifiers.
type XAResourceFactory interface {
	// CreateXAResource creates a new XAResource for the given driver connection.
	CreateXAResource(conn driver.Conn) XAResource
	// CreateErrorClassifier creates a database-specific error classifier.
	CreateErrorClassifier() XAErrorClassifier
}

// registry holds registered XAResourceFactory instances per DBType.
var registry = map[types.DBType]XAResourceFactory{}

// RegisterXAResourceFactory registers a factory for the given database type.
// Each database driver package should call this in its init() function.
func RegisterXAResourceFactory(dbType types.DBType, factory XAResourceFactory) {
	registry[dbType] = factory
}

// GetXAResourceFactory returns the registered factory for the given database type.
func GetXAResourceFactory(dbType types.DBType) (XAResourceFactory, bool) {
	f, ok := registry[dbType]
	return f, ok
}

// CreateXAResource creates an XAResource for the given database type and connection.
// It uses the registered factory for the database type.
func CreateXAResource(conn driver.Conn, dbType types.DBType) (XAResource, error) {
	factory, ok := GetXAResourceFactory(dbType)
	if !ok {
		return nil, fmt.Errorf("no XA resource factory registered for db type: %s", dbType.String())
	}
	return factory.CreateXAResource(conn), nil
}

// CreateErrorClassifier creates an XAErrorClassifier for the given database type.
// Returns a default no-op classifier if no factory is registered.
func CreateErrorClassifier(dbType types.DBType) XAErrorClassifier {
	factory, ok := GetXAResourceFactory(dbType)
	if !ok {
		return &defaultErrorClassifier{}
	}
	return factory.CreateErrorClassifier()
}
