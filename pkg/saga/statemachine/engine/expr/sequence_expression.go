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
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/sequence"
)

type SequenceExpression struct {
	seqGenerator sequence.SeqGenerator
	entity       string
	rule         string
}

func (s *SequenceExpression) SeqGenerator() sequence.SeqGenerator {
	return s.seqGenerator
}

func (s *SequenceExpression) SetSeqGenerator(seqGenerator sequence.SeqGenerator) {
	s.seqGenerator = seqGenerator
}

func (s *SequenceExpression) Entity() string {
	return s.entity
}

func (s *SequenceExpression) SetEntity(entity string) {
	s.entity = entity
}

func (s *SequenceExpression) Rule() string {
	return s.rule
}

func (s *SequenceExpression) SetRule(rule string) {
	s.rule = rule
}

func (s SequenceExpression) Value(vars map[string]any) any {
	return s.seqGenerator.GenerateId(s.entity, s.rule)
}

func (s SequenceExpression) SetValue(value any, elContext any) {

}

func (s SequenceExpression) ExpressionString() string {
	return s.entity + "|" + s.rule
}
