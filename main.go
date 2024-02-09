package main

import (
	"fmt"
	"os"
	"taskmanager/pkg/parser"
	"taskmanager/pkg/taskmanager"
	"unsafe"
)

// Local test
func main() {
	argsData, err := os.ReadFile("args.buf")
	if err != nil {
		fmt.Printf("Could not open arguments file: %v\n", err)
		return
	}
	argParser, err := parser.NewParser((uintptr)(unsafe.Pointer(&argsData[0])), uintptr(len(argsData)))
	if err != nil {
		fmt.Printf("Could not create argument parser: %v\n", err)
		return
	}

	cmdString, err := argParser.GetString()
	if err != nil {
		fmt.Printf("Could not get command string: %v\n", err)
	}

	result, err := taskmanager.ExecuteCommand(cmdString)
	if err != nil {
		fmt.Printf("Error running main function: %v\n", err)
		return
	}
	fmt.Println(result)
}
