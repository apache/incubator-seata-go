package filter

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestJsonMarshal(t *testing.T) {

	user := struct {
		Name  string `json:"name,select(lang)"`
		Age   int    `json:"age"`
		Hobby string `json:"hobby"`
		Lang  string `json:"lang,select(lang)"`
	}{
		Name:  "boyan",
		Age:   18,
		Hobby: "code",
		Lang:  "Go",
	}

	f := SelectMarshal("lang", user)
	marshal, err := json.Marshal(f)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(marshal))
	fmt.Println(f)
	//{"lang":"Go","name":"boyan"}
	//{"lang":"Go","name":"boyan"}
}
