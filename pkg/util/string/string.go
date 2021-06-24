package string

import "strings"

func Escape(tableName, cutset string) string {
	var tn = tableName
	if strings.Contains(tableName, ".") {
		idx := strings.LastIndex(tableName, ".")
		tName := tableName[idx+1:]
		tn = strings.Trim(tName, cutset)
	} else {
		tn = strings.Trim(tableName, cutset)
	}
	return tn
}
