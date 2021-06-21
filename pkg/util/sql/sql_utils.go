package sql

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

func AppendInParam(size int) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "(")
	for i := 0; i < size; i++ {
		fmt.Fprintf(&sb, "?")
		if i < size-1 {
			fmt.Fprint(&sb, ",")
		}
	}
	fmt.Fprintf(&sb, ")")
	return sb.String()
}

func AppendInParamPostgres(size int) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "(")
	for i := 0; i < size; i++ {
		str := "$" + strconv.Itoa(i+1)
		fmt.Fprintf(&sb, str)
		if i < size-1 {
			fmt.Fprint(&sb, ",")
		}
	}
	fmt.Fprintf(&sb, ")")
	return sb.String()
}

//postgresql解析条件时，需要转换$为顺序的
func ConvertParamOrder(condition string) string {
	//匹配条件的时候，将$符依次重新排序
	reg1 := regexp.MustCompile(`\$\d+`)
	result := reg1.FindAllStringSubmatch(condition, -1)
	var i = 0
	for _, value := range result {
		i++
		condition = strings.Replace(condition, value[0], "$"+strconv.Itoa(i), -1)
	}
	return condition
}
