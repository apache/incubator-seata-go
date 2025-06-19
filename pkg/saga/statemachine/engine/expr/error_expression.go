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
