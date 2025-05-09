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
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/PaesslerAG/gval"
	"github.com/expr-lang/expr"

	"github.com/seata/seata-go/pkg/saga/statemachine/engine/sequence"
)

const DefaultExpressionType string = "Default"

type ExpressionResolver interface {
	Expression(expressionStr string) Expression
	ExpressionFactoryManager() ExpressionFactoryManager
	SetExpressionFactoryManager(expressionFactoryManager ExpressionFactoryManager)
}

type Expression interface {
	Value(elContext any) any
	SetValue(value any, elContext any)
	ExpressionString() string
}

type ExpressionFactory interface {
	CreateExpression(expression string) Expression
}

type ExpressionFactoryManagerInterface interface {
	GetExpressionFactory(expressionType string) ExpressionFactory
	SetExpressionFactoryMap(expressionFactoryMap map[string]ExpressionFactory)
	PutExpressionFactory(expressionType string, factory ExpressionFactory)
	Register(typeName string, factory ExpressionFactory)
}
type ExpressionFactoryManager struct {
	expressionFactoryMap map[string]ExpressionFactory
}

func (m *ExpressionFactoryManager) Register(typeName string, factory ExpressionFactory) {
	m.expressionFactoryMap[typeName] = factory
}

type ELExpression struct {
	expression string
}

func (e *ELExpression) Value(elContext any) any {
	program, err := expr.Compile(e.expression)
	if err != nil {
		log.Printf("Failed to compile expression %s: %v", e.expression, err)
		return nil
	}

	output, err := expr.Run(program, elContext.(map[string]any))
	if err != nil {
		log.Printf("Failed to run expression %s: %v", e.expression, err)
		return nil
	}
	return output
}

func (e *ELExpression) SetValue(value any, elContext any) {
	ctxMap, ok := elContext.(map[string]any)
	if !ok {
		log.Printf("Invalid context type. Expected map[string]any, got %T", elContext)
		return
	}
	assignmentExpr := fmt.Sprintf("%s = %v", e.expression, value)
	evaluable, err := gval.Full().NewEvaluable(assignmentExpr)
	if err != nil {
		log.Printf("Failed to compile assignment expression %s: %v", assignmentExpr, err)
		return
	}
	ctx := context.Background()
	_, err = evaluable(ctx, ctxMap)
	if err != nil {
		log.Printf("Failed to evaluate assignment expression %s: %v", assignmentExpr, err)
	}
}

func (e *ELExpression) ExpressionString() string {
	return e.expression
}

type ELExpressionFactory struct{}

func (f *ELExpressionFactory) CreateExpression(expression string) Expression {
	return &ELExpression{expression: expression}
}

func NewExpressionFactoryManager() ExpressionFactoryManagerInterface {
	return &ExpressionFactoryManager{
		expressionFactoryMap: make(map[string]ExpressionFactory),
	}
}

func NewELExpressionFactory() *ELExpressionFactory {
	return &ELExpressionFactory{}
}

func (e *ExpressionFactoryManager) GetExpressionFactory(expressionType string) ExpressionFactory {
	if strings.TrimSpace(expressionType) == "" {
		expressionType = DefaultExpressionType
	}
	return e.expressionFactoryMap[expressionType]
}

var _ ExpressionFactoryManagerInterface = (*ExpressionFactoryManager)(nil)

func (e *ExpressionFactoryManager) SetExpressionFactoryMap(expressionFactoryMap map[string]ExpressionFactory) {
	for k, v := range expressionFactoryMap {
		e.expressionFactoryMap[k] = v
	}
}

func (e *ExpressionFactoryManager) PutExpressionFactory(expressionType string, factory ExpressionFactory) {
	e.expressionFactoryMap[expressionType] = factory
}

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

func (s SequenceExpression) Value(elContext any) any {
	return s.seqGenerator.GenerateId(s.entity, s.rule)
}

func (s SequenceExpression) SetValue(value any, elContext any) {

}

func (s SequenceExpression) ExpressionString() string {
	return s.entity + "|" + s.rule
}
