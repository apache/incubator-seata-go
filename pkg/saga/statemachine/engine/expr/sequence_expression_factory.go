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
