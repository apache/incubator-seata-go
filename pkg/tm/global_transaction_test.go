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

package tm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBegin(t *testing.T) {
	gts := []struct {
		globalTransaction GlobalTransaction
		wantHasError      bool
		wantErrString     string
	}{
		{
			globalTransaction: GlobalTransaction{
				Role: PARTICIPANT,
			},
			wantHasError: false,
		},
		{
			globalTransaction: GlobalTransaction{
				Role: LAUNCHER,
				Xid:  "123456",
			},
			wantHasError:  true,
			wantErrString: "Global transaction already exists,can't begin a new global transaction, currentXid = 123456 ",
		},
		// todo the next case depend on network service, it can't measured now.
	}
	for _, v := range gts {
		err := GetGlobalTransactionManager().Begin(context.Background(), &v.globalTransaction, 1, "GlobalTransaction")
		if v.wantHasError {
			assert.NotNil(t, err)
			assert.Regexp(t, v.wantErrString, err.Error())
		} else {
			assert.Nil(t, err)
		}
	}
}

func TestCommit(t *testing.T) {
	gts := []struct {
		globalTransaction GlobalTransaction
		wantHasError      bool
		wantErrString     string
	}{
		{
			globalTransaction: GlobalTransaction{
				Role: PARTICIPANT,
				Xid:  "123456",
			},
			wantHasError: false,
		},
		{
			globalTransaction: GlobalTransaction{
				Role: LAUNCHER,
			},
			wantHasError:  true,
			wantErrString: "Commit xid should not be empty",
		},
		// todo the next case depend on network service, it can't measured now.
	}
	for _, v := range gts {
		err := GetGlobalTransactionManager().Commit(context.Background(), &v.globalTransaction)
		if v.wantHasError {
			assert.NotNil(t, err)
			assert.Regexp(t, v.wantErrString, err.Error())
		} else {
			assert.Nil(t, err)
		}
	}
}

func TestRollback(t *testing.T) {
	gts := []struct {
		globalTransaction GlobalTransaction
		wantHasError      bool
		wantErrString     string
	}{
		{
			globalTransaction: GlobalTransaction{
				Role: PARTICIPANT,
				Xid:  "123456",
			},
			wantHasError: false,
		},
		{
			globalTransaction: GlobalTransaction{
				Role: LAUNCHER,
			},
			wantHasError:  true,
			wantErrString: "Rollback xid should not be empty",
		},
		// todo the next case dependent network service, it can't measured now.
	}
	for _, v := range gts {
		err := GetGlobalTransactionManager().Rollback(context.Background(), &v.globalTransaction)
		if v.wantHasError {
			assert.NotNil(t, err)
			assert.Regexp(t, v.wantErrString, err.Error())
		} else {
			assert.Nil(t, err)
		}
	}
}
