/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package sequence

import (
	"fmt"
	"sync"
	"time"

	"github.com/seata/seata-go/pkg/util/log"
)

// SnowflakeSeqGenerator snowflake gen ids
// ref: https://en.wikipedia.org/wiki/Snowflake_ID

var (
	// set the beginning time
	epoch = time.Date(2024, time.January, 01, 00, 00, 00, 00, time.UTC).UnixMilli()
)

const (
	// timestamp occupancy bits
	timestampBits = 41
	// dataCenterId occupancy bits
	dataCenterIdBits = 5
	// workerId occupancy bits
	workerIdBits = 5
	// sequence occupancy bits
	seqBits = 12

	// timestamp max value, just like 2^41-1 = 2199023255551
	timestampMaxValue = -1 ^ (-1 << timestampBits)
	// dataCenterId max value, just like 2^5-1 = 31
	dataCenterIdMaxValue = -1 ^ (-1 << dataCenterIdBits)
	// workId max value, just like 2^5-1 = 31
	workerIdMaxValue = -1 ^ (-1 << workerIdBits)
	// sequence max value, just like 2^12-1 = 4095
	seqMaxValue = -1 ^ (-1 << seqBits)

	// number of workId offsets (seqBits)
	workIdShift = 12
	// number of dataCenterId offsets (seqBits + workerIdBits)
	dataCenterIdShift = 17
	// number of timestamp offsets (seqBits + workerIdBits + dataCenterIdBits)
	timestampShift = 22

	defaultInitValue = 0
)

type SnowflakeSeqGenerator struct {
	mu           *sync.Mutex
	timestamp    int64
	dataCenterId int64
	workerId     int64
	sequence     int64
}

// NewSnowflakeSeqGenerator initiates the snowflake generator
func NewSnowflakeSeqGenerator(dataCenterId, workId int64) (r *SnowflakeSeqGenerator, err error) {
	if dataCenterId < 0 || dataCenterId > dataCenterIdMaxValue {
		err = fmt.Errorf("dataCenterId should between 0 and %d", dataCenterIdMaxValue-1)
		return
	}

	if workId < 0 || workId > workerIdMaxValue {
		err = fmt.Errorf("workId should between 0 and %d", dataCenterIdMaxValue-1)
		return
	}

	return &SnowflakeSeqGenerator{
		mu:           new(sync.Mutex),
		timestamp:    defaultInitValue - 1,
		dataCenterId: dataCenterId,
		workerId:     workId,
		sequence:     defaultInitValue,
	}, nil
}

// GenerateId timestamp + dataCenterId + workId + sequence
func (S *SnowflakeSeqGenerator) GenerateId(entity string, ruleName string) string {
	S.mu.Lock()
	defer S.mu.Unlock()

	now := time.Now().UnixMilli()

	if S.timestamp > now { // Clock callback
		log.Errorf("Clock moved backwards. Refusing to generate ID, last timestamp is %d, now is %d", S.timestamp, now)
		return ""
	}

	if S.timestamp == now {
		// generate multiple IDs in the same millisecond, incrementing the sequence number to prevent conflicts
		S.sequence = (S.sequence + 1) & seqMaxValue
		if S.sequence == 0 {
			// sequence overflow, waiting for next millisecond
			for now <= S.timestamp {
				now = time.Now().UnixMilli()
			}
		}
	} else {
		// initialized sequences are used directly at different millisecond timestamps
		S.sequence = defaultInitValue
	}
	tmp := now - epoch
	if tmp > timestampMaxValue {
		log.Errorf("epoch should between 0 and %d", timestampMaxValue-1)
		return ""
	}
	S.timestamp = now

	// combine the parts to generate the final ID and convert the 64-bit binary to decimal digits.
	r := (tmp)<<timestampShift |
		(S.dataCenterId << dataCenterIdShift) |
		(S.workerId << workIdShift) |
		(S.sequence)

	return fmt.Sprintf("%d", r)
}
