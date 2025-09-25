package app

import (
	"context"
	"os/exec"
	"strings"
	"sync"

	"github.com/bvisness/flowshell/clay"
	rl "github.com/gen2brain/raylib-go/raylib"
)

type V2 = rl.Vector2

type Node struct {
	ID   int
	Pos  V2
	Name string

	InputPorts  []NodePort
	OutputPorts []NodePort

	Action NodeAction

	Running bool
}

type NodePort struct {
	Name string
	Type FlowType
}

func (n *Node) Run() {
	if n.Running {
		return
	}
	n.Running = true

	done := n.Action.Run(n)
	go func() {
		<-done
		n.Running = false
	}()
}

type NodeAction interface {
	UI(n *Node)
	Run(n *Node) (done <-chan struct{})
	Result() NodeActionResult
}

type NodeActionResult struct {
	Outputs []FlowValue
	Err     error
}

type CmdAction struct {
	CmdString string

	state             CmdActionRuntimeState
	outputStreamMutex sync.Mutex
}

// The state that gets reset every time you run a command
type CmdActionRuntimeState struct {
	cmd    *exec.Cmd
	cancel context.CancelFunc

	stdout   []byte
	stderr   []byte
	combined []byte

	err      error
	exitCode int
}

func (c *CmdAction) UI(n *Node) {
	UITextBox(clay.ID("Cmd"), &c.CmdString, clay.EL{Layout: clay.LAY{Sizing: GROWH}})
}

func (c *CmdAction) Run(n *Node) <-chan struct{} {
	pieces := strings.Split(c.CmdString, " ")
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, pieces[0], pieces[1:]...)

	done := make(chan struct{})

	c.state = CmdActionRuntimeState{
		cmd:    cmd,
		cancel: cancel,
	}

	cmd.Stdout = &multiSliceWriter{
		mu: &c.outputStreamMutex,
		a:  &c.state.stdout,
		b:  &c.state.combined,
	}
	cmd.Stderr = &multiSliceWriter{
		mu: &c.outputStreamMutex,
		a:  &c.state.stderr,
		b:  &c.state.combined,
	}

	go func() {
		c.state.err = c.state.cmd.Run()
		if c.state.err != nil {
			// TODO: Extract exit code
		}
		done <- struct{}{}
	}()

	return done
}

func (c *CmdAction) Result() NodeActionResult {
	return NodeActionResult{
		Err: c.state.err,
		Outputs: []FlowValue{
			{
				Type:       &FlowType{Kind: FSKindBytes},
				BytesValue: c.state.stdout,
			},
			{
				Type:       &FlowType{Kind: FSKindBytes},
				BytesValue: c.state.stderr,
			},
			{
				Type:       &FlowType{Kind: FSKindBytes},
				BytesValue: c.state.combined,
			},
		},
	}
}
