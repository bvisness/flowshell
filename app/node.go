package app

import (
	"context"
	"os/exec"
	"strings"
	"sync"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type V2 = rl.Vector2

type Node struct {
	ID  int
	Pos V2

	Cmd NodeCmd

	Running bool
}

type NodeCmd struct {
	CmdString string

	state             NodeCmdRuntimeState
	outputStreamMutex sync.Mutex
}

func (c *NodeCmd) Err() error {
	return c.state.err
}

func (c *NodeCmd) CombinedOutput() []byte {
	return c.state.combined
}

// The state that gets reset every time you run a command
type NodeCmdRuntimeState struct {
	cmd    *exec.Cmd
	cancel context.CancelFunc

	stdout   []byte
	stderr   []byte
	combined []byte

	err      error
	exitCode int
}

func (n *Node) Run() {
	if n.Running {
		return
	}
	n.Running = true

	// TODO: Handle other node types besides cmd
	pieces := strings.Split(n.Cmd.CmdString, " ")
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, pieces[0], pieces[1:]...)

	n.Cmd.state = NodeCmdRuntimeState{
		cmd:    cmd,
		cancel: cancel,
	}

	cmd.Stdout = &multiSliceWriter{
		mu: &n.Cmd.outputStreamMutex,
		a:  &n.Cmd.state.stdout,
		b:  &n.Cmd.state.combined,
	}
	cmd.Stdout = &multiSliceWriter{
		mu: &n.Cmd.outputStreamMutex,
		a:  &n.Cmd.state.stderr,
		b:  &n.Cmd.state.combined,
	}

	go func() {
		n.Cmd.state.err = n.Cmd.state.cmd.Run()
		if n.Cmd.state.err != nil {
			// TODO: Extract exit code
		}
		n.Running = false
	}()
}
