package sequence

import "github.com/google/uuid"

type UUIDSeqGenerator struct {
}

func NewUUIDSeqGenerator() *UUIDSeqGenerator {
	return &UUIDSeqGenerator{}
}

func (U UUIDSeqGenerator) GenerateId(entity string, ruleName string) string {
	return uuid.New().String()
}
