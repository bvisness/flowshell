package trace

import "github.com/go-stack/stack"

type CallStack []StackFrame
type StackFrame struct {
	File     string
	Line     int
	Function string
}

func Trace() CallStack {
	trace := stack.Trace().TrimRuntime()[1:]
	frames := make(CallStack, len(trace))
	for i, call := range trace {
		callFrame := call.Frame()
		frames[i] = StackFrame{
			File:     callFrame.File,
			Line:     callFrame.Line,
			Function: callFrame.Function,
		}
	}

	return frames
}
