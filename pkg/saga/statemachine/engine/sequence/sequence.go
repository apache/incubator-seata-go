package sequence

type SeqGenerator interface {
	GenerateId(entity string, ruleName string) string
}
