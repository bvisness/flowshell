package app

import (
	"context"
	"os/exec"
	"strings"
	"sync"

	"github.com/bvisness/flowshell/clay"
)

// GEN:NodeAction
type RunProcessAction struct {
	CmdString string

	state             RunProcessActionRuntimeState
	outputStreamMutex sync.Mutex
}

func NewRunProcessNode(cmd string) *Node {
	return &Node{
		ID:   NewNodeID(),
		Name: "Run Process",

		InputPorts: nil,
		OutputPorts: []NodePort{
			{
				Name: "Stdout",
				Type: FlowType{Kind: FSKindBytes},
			},
			{
				Name: "Stderr",
				Type: FlowType{Kind: FSKindBytes},
			},
			{
				Name: "Combined Stdout/Stderr",
				Type: FlowType{Kind: FSKindBytes},
			},
		},

		Action: &RunProcessAction{
			CmdString: cmd,
		},
	}
}

var _ NodeAction = &RunProcessAction{}

// The state that gets reset every time you run a command
type RunProcessActionRuntimeState struct {
	cmd    *exec.Cmd
	cancel context.CancelFunc

	stdout   []byte
	stderr   []byte
	combined []byte

	err      error
	exitCode int
}

func (c *RunProcessAction) UpdateAndValidate(n *Node) {
	n.Valid = true
}

func (c *RunProcessAction) UI(n *Node) {
	clay.CLAY_AUTO_ID(clay.EL{
		Layout: clay.LAY{
			LayoutDirection: clay.TopToBottom,
			Sizing:          GROWH,
			ChildGap:        S2,
		},
	}, func() {
		UITextBox(clay.IDI("RunProcessCmd", n.ID), &c.CmdString, UITextBoxConfig{
			El: clay.EL{Layout: clay.LAY{Sizing: GROWH}},
		})

		clay.CLAY_AUTO_ID(clay.EL{
			Layout: clay.LAY{
				LayoutDirection: clay.TopToBottom,
				Sizing:          GROWH,
				ChildAlignment:  XRIGHT,
			},
		}, func() {
			UIOutputPort(n, 0)
			UIOutputPort(n, 1)
			UIOutputPort(n, 2)
		})
	})
}

func (c *RunProcessAction) Run(n *Node) <-chan NodeActionResult {
	pieces := strings.Split(c.CmdString, " ")
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, pieces[0], pieces[1:]...)

	done := make(chan NodeActionResult)

	c.state = RunProcessActionRuntimeState{
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
		var res NodeActionResult
		defer func() { done <- res }()

		c.state.err = c.state.cmd.Run()
		if c.state.err != nil {
			// TODO: Extract exit code
		}

		res = NodeActionResult{
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
	}()

	return done
}

func (n *RunProcessAction) Serialize(s *Serializer) bool {
	SStr(s, &n.CmdString)
	return s.Ok()
}
