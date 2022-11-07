package filter

import (
	"fmt"
	"time"
)

type Time time.Time

const timeFmt = "2006-01-02 15:04:05"

func (t Time) MarshalJSON() ([]byte, error) {
	fmtTime := time.Time(t)
	formatted := fmt.Sprintf("\"%s\"", fmtTime.Format(timeFmt))
	return []byte(formatted), nil
}

func (t *Time) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	var err error
	var timeTmp time.Time
	timeTmp, err = time.Parse(`"`+timeFmt+`"`, string(data))
	*t = Time(timeTmp)
	return err
}

func (t Time) String() string {
	return time.Time(t).Format(timeFmt)
}
