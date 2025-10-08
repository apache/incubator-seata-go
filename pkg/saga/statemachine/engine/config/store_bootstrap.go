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

package config

import (
	"database/sql"
	"fmt"

	"github.com/seata/seata-go/pkg/saga/statemachine/engine/repo/repository"
	dbstore "github.com/seata/seata-go/pkg/saga/statemachine/store/db"
	sagaTm "github.com/seata/seata-go/pkg/saga/tm"
)

// SetupStoresFromConfig wires DB-backed StateLogStore/StateLangStore and Saga TM template
// according to runtime options loaded into DefaultStateMachineConfig.
// It keeps Go style (no DI/SPI), and is safe to call multiple times.
func (c *DefaultStateMachineConfig) SetupStoresFromConfig() error {
	if !c.storeEnabled {
		// keep Noop stores for in-memory usage
		return nil
	}
	if c.storeType == "" || c.storeDSN == "" {
		return fmt.Errorf("store_enabled=true but store_type/dsn not provided")
	}

	driver := c.storeType
	if driver == "sqlite" || driver == "sqlite3" {
		driver = "sqlite3"
	}

	db, err := sql.Open(driver, c.storeDSN)
	if err != nil {
		return err
	}
	if err := db.Ping(); err != nil {
		return fmt.Errorf("db ping failed: %w", err)
	}

	// build stores with default table prefix `seata_`
	lang := dbstore.NewStateLangStore(db, "seata_")
	logStore := dbstore.NewStateLogStore(db, "seata_")

	// inject transactional template only when tc is enabled
	if c.tcEnabled {
		// keep minimal default template; client init is handled externally by user
		var tmpl sagaTm.SagaTransactionalTemplate = &sagaTm.DefaultSagaTransactionalTemplate{}
		logStore.SetSagaTransactionalTemplate(tmpl)
	}

	// set into config & repository
	c.SetStateLangStore(lang)
	c.SetStateLogStore(logStore)
	repository.GetStateMachineRepositoryImpl().SetStateLangStore(lang)

	return nil
}
