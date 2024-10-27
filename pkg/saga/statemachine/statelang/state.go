package statelang

type State interface {
	Name() string

	SetName(name string)

	Comment() string

	SetComment(comment string)

	Type() string

	SetType(typeName string)

	Next() string

	SetNext(next string)

	StateMachine() StateMachine

	SetStateMachine(machine StateMachine)
}

type BaseState struct {
	name         string `alias:"Name"`
	comment      string `alias:"Comment"`
	typeName     string `alias:"Type"`
	next         string `alias:"Next"`
	stateMachine StateMachine
}

func NewBaseState() *BaseState {
	return &BaseState{}
}

func (b *BaseState) Name() string {
	return b.name
}

func (b *BaseState) SetName(name string) {
	b.name = name
}

func (b *BaseState) Comment() string {
	return b.comment
}

func (b *BaseState) SetComment(comment string) {
	b.comment = comment
}

func (b *BaseState) Type() string {
	return b.typeName
}

func (b *BaseState) SetType(typeName string) {
	b.typeName = typeName
}

func (b *BaseState) Next() string {
	return b.next
}

func (b *BaseState) SetNext(next string) {
	b.next = next
}

func (b *BaseState) StateMachine() StateMachine {
	return b.stateMachine
}

func (b *BaseState) SetStateMachine(machine StateMachine) {
	b.stateMachine = machine
}
