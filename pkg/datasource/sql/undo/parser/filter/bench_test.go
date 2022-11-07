package filter

import (
	"testing"
	"time"
)

type Users struct {
	UID    uint   `json:"uid,select(article)"`    //The scene represented in the selection (the scene used by this scene)
	Avatar string `json:"avatar,select(article)"` //Same as above, this field will only be parsed when the article interface is used

	Nickname string `json:"nickname,select(article|profile)"` //"|" indicates that there are multiple scenarios that require this field. The article interface requires the profile interface as well

	Sex        int       `json:"sex,select(profile)"`          //This field is only used by profile
	VipEndTime time.Time `json:"vip_end_time,select(profile)"` // same as above
	Price      string    `json:"price,select(profile)"`        // same as above

	Hobby string    `json:"hobby,omitempty,select($any)"` //Ignore if empty in any scenario
	Lang  []LangAge `json:"lang,omitempty,select($any)"`  //Ignore if empty in any scenario
}

type LangAge struct {
	Name string `json:"name,omitempty,select($any)"`
	Art  string `json:"art,omitempty,select($any)"`
}

func newUsers() Users {
	return Users{
		UID:        1,
		Nickname:   "boyan",
		Avatar:     "avatar",
		Sex:        1,
		VipEndTime: time.Now().Add(time.Hour * 24 * 365),
		Price:      "999.9",
		Lang: []LangAge{
			{
				Name: "1",
				Art:  "24",
			},
			{
				Name: "2",
				Art:  "35",
			},
		},
	}
}

var str string

func BenchmarkUserPointer(b *testing.B) {
	user := newUsers()
	for i := 0; i < b.N; i++ {
		str = SelectMarshal("article", &user).MustJSON()
	}
}

func BenchmarkUserVal(b *testing.B) {
	user := newUsers()
	for i := 0; i < b.N; i++ {
		str = SelectMarshal("article", user).MustJSON()
	}
}
