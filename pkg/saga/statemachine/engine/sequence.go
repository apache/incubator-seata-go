package engine

type SeqGenerator interface {
	GenerateId(entity string, ruleName string)
}
