package filter

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func GetSelectTag(tag string) []string {
	tags := strings.Split(tag, ",")
	selectTags := make([]string, 0, 5)
	for _, s := range tags {
		if strings.HasPrefix(s, "select(") {
			selectStr := s[7 : len(s)-1]
			scene := strings.Split(selectStr, "|")
			for _, v := range scene {
				selectTags = append(selectTags, v)
			}
		}
	}
	return selectTags
}
func TestTagSelect(t *testing.T) {
	tag := "name,omitempty,select(req|res),omit(chat|profile|article)"
	want := []string{
		"req", "res",
	}
	got := GetSelectTag(tag)

	if len(got) != len(want) {
		t.Errorf("tag 解析不符合预期want:%v got:%v", want, got)
		return
	}

	for i, v := range got {
		if !(v == want[i]) {
			t.Errorf("tag 解析不符合预期want:%v got:%v", want, got)
		}
	}
}
func TestNewSelectTag(t *testing.T) {
	selector := "req"
	name := "name"
	tag := "name,omitempty,select(req|res),omit(chat|profile|article)"
	got := newSelectTag(tag, "req", "name")
	if got.IsOmitField {
		t.Errorf("IsOmitField 应该为true")
	}
	if !got.IsSelect {
		t.Errorf("IsSelect 应该为true")
	}
	if !got.Omitempty {
		t.Errorf("Omitempty 应该为true")
	}

	if got.SelectScene != selector {
		t.Errorf("SelectScene 应为%v 实际%v", selector, got.SelectScene)
	}
	if got.UseFieldName != name {
		t.Errorf("UseFieldName 应为%v 实际%v", name, got.UseFieldName)
	}

	//=== RUN   TestNewSelectTag
	//--- PASS: TestNewSelectTag (0.00s)
	//PASS
}

func TestNewSelectTag1(t *testing.T) {
	//selector := "req"
	//name := "name"
	tag := ",select(req|res),omit(chat|profile|article)"
	got := newSelectTag(tag, "req", "name")
	fmt.Println(fmt.Sprintf("%+v", got))

	type Book struct {
		BookName string
	}

	type User struct {
		Name string
		Book
	}

	fmt.Println(reflect.TypeOf(User{}).Field(0).Name)
	fmt.Println(reflect.TypeOf(User{}).Field(1).Name)

	//=== RUN   TestNewSelectTag
	//--- PASS: TestNewSelectTag (0.00s)
	//PASS
}

func OmitTest() {
	_ = newOmitTag("name,omitempty,select(req|res),omit(chat|profile|article)", "article", "IsOmitField:true")
}

func BenchmarkTags(b *testing.B) {

	b.Run("select", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			newSelectTag("name,omitempty,select(req|res|user),omit(chat|profile|article)", "user", "Name")
		}
	})
	b.Run("select-f", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			newSelectTag("name,omitempty,select(req|res|user),omit(chat|profile|article)", "req", "Name")
		}
	})
	b.Run("omit", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			newOmitTag("name,omitempty,select(req|res|user),omit(chat|profile|article)", "article", "Name")
		}
	})
	b.Run("omit-f", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			newOmitTag("name,omitempty,select(req|res|user),omit(chat|profile|article)", "chat", "Name")
		}
	})

	//goos: darwin
	//goarch: amd64
	//pkg: filter/filter
	//cpu: Intel(R) Core(TM) i9-9880H CPU @ 2.30GHz
	//BenchmarkTags
	//BenchmarkTags/select
	//BenchmarkTags/select-16         	 5682181	       205.6 ns/op
	//BenchmarkTags/select-f
	//BenchmarkTags/select-f-16       	 5831988	       197.4 ns/op
	//BenchmarkTags/omit
	//BenchmarkTags/omit-16           	 5868252	       203.6 ns/op
	//BenchmarkTags/omit-f
	//BenchmarkTags/omit-f-16         	 5985828	       204.8 ns/op
	//PASS

}

func TestOmitTag1(t *testing.T) {
	tag1 := "name,omit(lang)"
	tag2 := "name"
	gotTag1 := newOmitTag(tag1, "lang", "Name")
	gotTag2 := newOmitTag(tag2, "lang", "Name")
	fmt.Println(fmt.Sprintf("%+v", gotTag1))
	//{SelectScene:lang IsOmitField:true IsSelect:false UseFieldName:name IsAnonymous:false Omitempty:false}

	fmt.Println(fmt.Sprintf("%+v", gotTag2))
	//{SelectScene:lang IsOmitField:false IsSelect:true UseFieldName:name IsAnonymous:false Omitempty:false}

	//ok
}
