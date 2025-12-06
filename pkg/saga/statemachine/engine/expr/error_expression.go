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

// ErrorExpression is a placeholder implementation that always reports an error.
// When parsing/constructing an expression fails, this type is returned directly.
// The Value() method returns the error as-is, and SetValue() is a no-op.
type ErrorExpression struct {
	err           error
	expressionStr string
}

func (e *ErrorExpression) Value(elContext any) any {
	return e.err
}

func (e *ErrorExpression) SetValue(value any, elContext any) {
	// No write operation for error expressions
}

func (e *ErrorExpression) ExpressionString() string {
	return e.expressionStr
}
