package sequence

import (
	"strconv"
	"testing"
)

func TestSnowflakeSeqGenerator_GenerateId(t *testing.T) {
	var dataCenterId, workId int64 = 1, 1
	generator, err := NewSnowflakeSeqGenerator(dataCenterId, workId)
	if err != nil {
		t.Error(err)
		return
	}
	var x, y string
	for i := 0; i < 100; i++ {
		y = generator.GenerateId("", "")
		if x == y {
			t.Errorf("x(%s) & y(%s) are the same", x, y)
		}
		x = y
	}
}

func TestEpoch(t *testing.T) {
	t.Log(epoch)
	t.Log(len(strconv.FormatInt(epoch, 10)))
}
