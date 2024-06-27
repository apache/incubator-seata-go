package expr

import (
	"seata.apache.org/seata-go/pkg/saga/statemachine/engine/sequence"
	"strings"
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

type ExpressionFactoryManager struct {
	expressionFactoryMap map[string]ExpressionFactory
}

func NewExpressionFactoryManager() *ExpressionFactoryManager {
	return &ExpressionFactoryManager{
		expressionFactoryMap: make(map[string]ExpressionFactory),
	}
}

func (e *ExpressionFactoryManager) GetExpressionFactory(expressionType string) ExpressionFactory {
	if strings.TrimSpace(expressionType) == "" {
		expressionType = DefaultExpressionType
	}
	return e.expressionFactoryMap[expressionType]
}

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
