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

package parser

import (
	"encoding/json"

	"seata.apache.org/seata-go/pkg/datasource/sql/undo"
)

type JsonParser struct {
}

// Get the name of parser;
// return the name of parser
func (j *JsonParser) GetName() string {
	return "json"
}

// Get default context of this parser
// return the default content if undo log is empty
func (j *JsonParser) GetDefaultContent() []byte {
	return []byte("{}")
}

// Encode branch undo log to byte array.
// param branchUndoLog the branch undo log
// return the byte array
func (j *JsonParser) Encode(branchUndoLog *undo.BranchUndoLog) ([]byte, error) {
	bytes, err := json.Marshal(branchUndoLog)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

// Decode byte array to branch undo log.
// param bytes the byte array
// return the branch undo log
func (j *JsonParser) Decode(bytes []byte) (*undo.BranchUndoLog, error) {
	var branchUndoLog *undo.BranchUndoLog
	if err := json.Unmarshal(bytes, &branchUndoLog); err != nil {
		return nil, err
	}

	return branchUndoLog, nil
}
