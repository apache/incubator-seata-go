package state

import "github.com/seata/seata-go/pkg/saga/statemachine/statelang"

type ChoiceState interface {
	statelang.State

	GetChoices() []Choice

	GetDefault() string
}

type Choice interface {
	GetExpression() string

	SetExpression(expression string)

	GetNext() string

	SetNext(next string)
}

type ChoiceStateImpl struct {
	statelang.BaseState
	defaultChoice string   `alias:"Default"`
	choices       []Choice `alias:"Choices"`
}

func NewChoiceStateImpl() *ChoiceStateImpl {
	return &ChoiceStateImpl{
		choices: make([]Choice, 0),
	}
}

func (choiceState *ChoiceStateImpl) GetDefault() string {
	return choiceState.defaultChoice
}

func (choiceState *ChoiceStateImpl) GetChoices() []Choice {
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

func (c *ChoiceImpl) GetExpression() string {
	return c.expression
}

func (c *ChoiceImpl) SetExpression(expression string) {
	c.expression = expression
}

func (c *ChoiceImpl) GetNext() string {
	return c.next
}

func (c *ChoiceImpl) SetNext(next string) {
	c.next = next
}
