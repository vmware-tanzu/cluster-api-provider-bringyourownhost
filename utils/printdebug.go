package utils

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
)

//you can added this temporary unified prefix string when you debug your code, for the convenience of searching log
const DebugPrefix = "huchen"
const StructType = 1
const OtherType = 2
const EmptyType = 3

func PrintObj(prefix string, v interface{}, objType int) {

	pc, _, line, _ := runtime.Caller(1)
	funcNam := runtime.FuncForPC(pc).Name()

	if v == nil && objType != EmptyType {
		fmt.Printf("\n%s: %s: %d: %s is nil\n", DebugPrefix, funcNam, line, prefix)
		return
	}

	if objType == StructType {
		b, err := json.Marshal(v)
		if err != nil {
			fmt.Printf("\n%s: %s: %d: %s error= %v\n", DebugPrefix, funcNam, line, prefix, err)
			return
		}
		fmt.Printf("\n%s: %s: %d: %s  %s\n", DebugPrefix, funcNam, line, prefix, string(b))
	} else if objType == OtherType {
		fmt.Printf("\n%s: %s: %d: %s  %v\n", DebugPrefix, funcNam, line, prefix, v)
	} else {
		fmt.Printf("\n%s: %s: %d: %s \n", DebugPrefix, funcNam, line, prefix)
	}

}

func PrintFileInfo(fileName string) {

	pc, _, line, _ := runtime.Caller(2)
	funcNam := runtime.FuncForPC(pc).Name()

	cmd := fmt.Sprintf("ls -l %s", fileName)

	command := exec.Command("/bin/sh", "-c", cmd)
	output, err := command.Output()
	if err != nil {
		fmt.Printf("\n%s: %s: %d: ls -l %s error= %v\n", DebugPrefix, funcNam, line, fileName, err)
		return
	}

	fmt.Printf("\n%s: %s: %d: ls -l %s\n%s", DebugPrefix, funcNam, line, fileName, output)

}
