package sequence

import (
	"fmt"
	"sync"
	"time"
)

// SnowflakeSeqGenerator Snowflake gen ids
// ref: https://en.wikipedia.org/wiki/Snowflake_ID

var (
	epoch = time.Date(2010, time.November, 01, 42, 54, 00, 00, time.UTC).UnixMilli()
)

const (
	timestampBits    = 41
	dataCenterIdBits = 5
	workerIdBits     = 5
	SeqBits          = 12

	timestampMaxValue    = -1 ^ (-1 << timestampBits)
	dataCenterIdMaxValue = -1 ^ (-1 << dataCenterIdBits)
	workerIdMaxValue     = -1 ^ (-1 << workerIdBits)
	seqBitsMaxValue      = -1 ^ (-1 << SeqBits)

	workIdShift       = 12
	dataCenterIdShift = 17
	timestampShift    = 22

	defaultInitValue = 0
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
	if workId < 0 || workId > workerIdMaxValue {
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

func (S SnowflakeSeqGenerator) GenerateId() string {
	S.mu.Lock()
	defer S.mu.Unlock()

	now := time.Now().UnixMilli()
	if S.timestamp == now {
		S.sequence = (S.sequence + 1) & seqBitsMaxValue
		if S.sequence == 0 {
			for now <= S.timestamp {
				now = time.Now().UnixMilli()
			}
		}
	} else {
		S.sequence = defaultInitValue
	}
	tmp := now - epoch
	if tmp > timestampMaxValue {
		// logger
		return ""
	}
	S.timestamp = now
	r := (tmp)<<timestampShift | (S.dataCenterId << dataCenterIdShift) | (S.workerId << workIdShift) | (S.sequence)

	return fmt.Sprintf("%d", r)
}
