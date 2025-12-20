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
	"errors"
	"strings"
)

type ExpressionResolver interface {
	Expression(expressionStr string) Expression
	ExpressionFactoryManager() ExpressionFactoryManager
	SetExpressionFactoryManager(expressionFactoryManager ExpressionFactoryManager)
}

type DefaultExpressionResolver struct {
	expressionFactoryManager ExpressionFactoryManager
}

func (resolver *DefaultExpressionResolver) Expression(expressionStr string) Expression {
	expressionStruct, err := parseExpressionStruct(expressionStr)
	if err != nil {
		return nil
	}
	expressionFactory := resolver.expressionFactoryManager.GetExpressionFactory(expressionStruct.typ)
	if expressionFactory == nil {
		return nil
	}
	return expressionFactory.CreateExpression(expressionStruct.content)
}

func (resolver *DefaultExpressionResolver) ExpressionFactoryManager() ExpressionFactoryManager {
	return resolver.expressionFactoryManager
}

func (resolver *DefaultExpressionResolver) SetExpressionFactoryManager(expressionFactoryManager ExpressionFactoryManager) {
	resolver.expressionFactoryManager = expressionFactoryManager
}

type ExpressionStruct struct {
	typeStart int
	typeEnd   int
	end       int
	typ       string
	content   string
}

// old style: $type{content}
// new style: $type.content
func parseExpressionStruct(expressionStr string) (*ExpressionStruct, error) {
	eStruct := &ExpressionStruct{}
	eStruct.typeStart = strings.Index(expressionStr, "$")
	if eStruct.typeStart == -1 {
		return nil, errors.New("invalid expression")
	}

	dot := strings.Index(expressionStr, ".")
	leftBracket := strings.Index(expressionStr, "{")

	isOldEvaluatorStyle := false
	if eStruct.typeStart == 0 {
		if leftBracket < 0 && dot < 0 {
			return nil, errors.New("invalid expression")
		}
		// Backward compatible for structure: $expressionType{expressionContent}
		if leftBracket > 0 && (leftBracket < dot || dot < 0) {
			eStruct.typeEnd = leftBracket
			isOldEvaluatorStyle = true
		}
		if dot > 0 && (dot < leftBracket || leftBracket < 0) {
			eStruct.typeEnd = dot
		}
	}

	if eStruct.typeStart == 0 && leftBracket != -1 && leftBracket < dot {
		// Backward compatible for structure: $expressionType{expressionContent}
		eStruct.typeEnd = strings.Index(expressionStr, "{")
		isOldEvaluatorStyle = true
	}

	eStruct.typ = expressionStr[eStruct.typeStart+1 : eStruct.typeEnd]

	if isOldEvaluatorStyle {
		eStruct.end = strings.Index(expressionStr, "}")
	} else {
		eStruct.end = len(expressionStr)
	}
	eStruct.content = expressionStr[eStruct.typeEnd+1 : eStruct.end]
	return eStruct, nil
}
