package sequence

import (
	"fmt"
	"sync"
	"time"
)

// SnowflakeSeqGenerator Snowflake gen ids
// ref: https://en.wikipedia.org/wiki/Snowflake_ID

var (
	epoch = time.Date(2010, time.November, 01, 42, 54, 00, 00, time.UTC).UnixMicro()
)

const (
	timestampBits    = 41
	dataCenterIdBits = 5
	workerIdBits     = 5
	SeqBits          = 12

	defaultInitValue     = 0
	timestampMaxValue    = -1 ^ (-1 << timestampBits)
	dataCenterIdMaxValue = -1 ^ (-1 << dataCenterIdBits)
	workerIdBitsMaxValue = -1 ^ (-1 << workerIdBits)
	seqBitsMaxValue      = -1 ^ (-1 << SeqBits)

	workIdShift       = 12
	dataCenterIdShift = 17
	timestampShift    = 22
)

type SnowflakeSeqGenerator struct {
	mu           *sync.Mutex
	timestamp    int64
	dataCenterId int64
	workerId     int64
	sequence     int64
}

func NewSnowflakeSeqSeqGenerator(dataCenterId, workId int64) (r *SnowflakeSeqGenerator, err error) {
	if dataCenterId < 0 || dataCenterId > dataCenterIdMaxValue {
		err = fmt.Errorf("dataCenterId must be between 0 and %d", dataCenterIdMaxValue-1)
		return
	}
	if workId < 0 || workId > workerIdBitsMaxValue {
		err = fmt.Errorf("workId must be between 0 and %d", dataCenterIdMaxValue-1)
		return
	}
	return &SnowflakeSeqGenerator{
		timestamp:    defaultInitValue,
		dataCenterId: dataCenterId,
		workerId:     workId,
		sequence:     defaultInitValue,
	}, nil
}

func (U SnowflakeSeqGenerator) GenerateId() string {
	U.mu.Lock()
	defer U.mu.Unlock()

	now := time.Now().UnixMilli()
	if U.timestamp == now {
		U.sequence = (U.sequence + 1) & seqBitsMaxValue
		if U.sequence == 0 {
			for now <= U.timestamp {
				now = time.Now().UnixMilli()
			}
		}
	} else {
		U.sequence = defaultInitValue
	}
	tmp := now - epoch
	if tmp > timestampMaxValue {
		// logger
		return ""
	}
	U.timestamp = now
	r := (tmp)<<timestampShift | (U.dataCenterId << dataCenterIdShift) | (U.workerId << workIdShift) | (U.sequence)

	return fmt.Sprintf("%d", r)
}
