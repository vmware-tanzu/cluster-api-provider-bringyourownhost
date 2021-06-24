package utils

import (
	"encoding/json"
	"fmt"
)

//you can added this temporary unified prefix string when you debug your code, for the convenience of searching log
const DebugPrefix = ""

func PrintObj(prefix string, v interface{}) {
	if v == nil {
		fmt.Printf("\n%s%s nil\n", DebugPrefix, prefix)
		return
	}
	b, err := json.Marshal(v)
	if err != nil {
		fmt.Printf("\n%s%s error= %v\n", DebugPrefix, prefix, err)
		return
	}
	fmt.Printf("\n%s%s %s\n", DebugPrefix, prefix, string(b))
}
