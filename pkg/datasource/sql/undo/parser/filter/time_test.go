package filter

import (
	"fmt"
	"testing"
	"time"
)

type UserTime struct {
	NilBirthTime Time      `json:"nil_birth_time,select(list)"`
	BirthTime2   *Time     `json:"birth_time2,select(list)"`
	BirthTime    Time      `json:"birth_time,select(list)"`
	Timer        time.Time `json:"timer,select($any)"`
}

func TestTime(t *testing.T) {
	now := time.Now()
	user := UserTime{
		BirthTime:  Time(now),
		BirthTime2: (*Time)(&now),
		Timer:      time.Now(),
	}

	fmt.Println(SelectMarshal("list", user).MustJSON())
	fmt.Println(SelectMarshal("list", user))
	//{"birth_time":"2022-06-25 14:22:24","birth_time2":"2022-06-25 14:22:24","nil_birth_time":"0001-01-01 00:00:00","timer":"2022-06-25T14:22:24.35714+08:00"}
}
