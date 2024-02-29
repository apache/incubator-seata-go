package sequence

import "sync"

// SnowflakeSeqGenerator Snowflake gen ids
// ref: https://en.wikipedia.org/wiki/Snowflake_ID
type SnowflakeSeqGenerator struct {
	Mutex        *sync.Mutex
	Timestamp    int64
	DataCenterId int64
	WorkerId     int64
	Sequence     int64
}

func NewSnowflakeSeqSeqGenerator(dataCenterId, workId int64) *SnowflakeSeqGenerator {
	return &SnowflakeSeqGenerator{
		DataCenterId: dataCenterId,
		WorkerId:     workId,
	}
}

func (U SnowflakeSeqGenerator) GenerateId() string {
	return ""
}
