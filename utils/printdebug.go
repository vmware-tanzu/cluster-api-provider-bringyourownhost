package utils

import (
	"encoding/json"
	"fmt"
)

//temporary Unified prefix, for the convenience of searching log
const DebugPrefix = "huchen:"

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
