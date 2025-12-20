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

package expr

import (
	"fmt"
	"strings"

	"github.com/seata/seata-go/pkg/saga/statemachine/engine/sequence"
)

// SequenceExpressionFactory implements the ExpressionFactory interface,
// designed to parse strings in the format "entity|rule" and create SequenceExpression instances.
// If the format is invalid, it returns an *ErrorExpression containing the parsing error.
type SequenceExpressionFactory struct {
	seqGenerator sequence.SeqGenerator
}

func NewSequenceExpressionFactory(seqGenerator sequence.SeqGenerator) *SequenceExpressionFactory {
	return &SequenceExpressionFactory{seqGenerator: seqGenerator}
}

// CreateExpression parses the input string into a SequenceExpression.
// The input must be in the format "entity|rule". If the format is invalid,
// it returns an ErrorExpression with a descriptive error message.
func (f *SequenceExpressionFactory) CreateExpression(expression string) Expression {
	parts := strings.Split(expression, "|")
	if len(parts) != 2 {
		return &ErrorExpression{
			err:           fmt.Errorf("invalid sequence expression format: %s, expected 'entity|rule'", expression),
			expressionStr: expression,
		}
	}

	seqExpr := &SequenceExpression{
		seqGenerator: f.seqGenerator,
		entity:       strings.TrimSpace(parts[0]),
		rule:         strings.TrimSpace(parts[1]),
	}
	return seqExpr
}
