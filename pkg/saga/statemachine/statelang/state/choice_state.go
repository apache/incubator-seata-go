package state

import "seata.apache.org/seata-go/pkg/saga/statemachine/statelang"

type ChoiceState interface {
	statelang.State

	Choices() []Choice

	Default() string
}

type Choice interface {
	Expression() string

	SetExpression(expression string)

	Next() string

	SetNext(next string)
}

type ChoiceStateImpl struct {
	*statelang.BaseState
	defaultChoice string   `alias:"Default"`
	choices       []Choice `alias:"Choices"`
}

func NewChoiceStateImpl() *ChoiceStateImpl {
	return &ChoiceStateImpl{
		BaseState: statelang.NewBaseState(),
		choices:   make([]Choice, 0),
	}
}

func (choiceState *ChoiceStateImpl) Default() string {
	return choiceState.defaultChoice
}

func (choiceState *ChoiceStateImpl) Choices() []Choice {
	return choiceState.choices
}

func (choiceState *ChoiceStateImpl) SetDefault(defaultChoice string) {
	choiceState.defaultChoice = defaultChoice
}

func (choiceState *ChoiceStateImpl) SetChoices(choices []Choice) {
	choiceState.choices = choices
}

type ChoiceImpl struct {
	expression string
	next       string
}

func NewChoiceImpl() *ChoiceImpl {
	return &ChoiceImpl{}
}

func (c *ChoiceImpl) Expression() string {
	return c.expression
}

func (c *ChoiceImpl) SetExpression(expression string) {
	c.expression = expression
}

func (c *ChoiceImpl) Next() string {
	return c.next
}

func (c *ChoiceImpl) SetNext(next string) {
	c.next = next
}
