package nodes

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/bvisness/flowshell/app/core"
	"github.com/bvisness/flowshell/clay"
)

// GEN:NodeAction
type HTTPRequestAction struct{}

func NewHTTPRequestNode() *core.Node {
	return &core.Node{
		Name: "HTTP Request",
		InputPorts: []core.NodePort{
			{Name: "URL", Type: core.FlowType{Kind: core.FSKindBytes}},
			{Name: "Method", Type: core.FlowType{Kind: core.FSKindBytes}}, // Optional, default GET
			{Name: "Body", Type: core.FlowType{Kind: core.FSKindBytes}},   // Optional
		},
		OutputPorts: []core.NodePort{
			{Name: "Status", Type: core.FlowType{Kind: core.FSKindInt64}},
			{Name: "Body", Type: core.FlowType{Kind: core.FSKindBytes}},
		},
		Action: &HTTPRequestAction{},
	}
}

var _ core.NodeAction = &HTTPRequestAction{}

func (a *HTTPRequestAction) UpdateAndValidate(n *core.Node) {
	n.Valid = true
}

func (a *HTTPRequestAction) UI(n *core.Node) {
	clay.CLAY(clay.IDI("NodeContent", n.ID), clay.EL{
		Layout: clay.LAY{LayoutDirection: clay.TopToBottom, Sizing: core.GROWH, ChildGap: core.S2},
	}, func() {
		clay.CLAY(clay.IDI("Row1", n.ID), clay.EL{
			Layout: clay.LAY{Sizing: core.GROWH, ChildAlignment: core.YCENTER},
		}, func() {
			core.UIInputPort(n, 0)
			core.UISpacer(clay.IDI("Spacer1", n.ID), core.GROWH)
			core.UIOutputPort(n, 0)
		})
		clay.CLAY(clay.IDI("Row2", n.ID), clay.EL{
			Layout: clay.LAY{Sizing: core.GROWH, ChildAlignment: core.YCENTER},
		}, func() {
			core.UIInputPort(n, 1)
			core.UISpacer(clay.IDI("Spacer2", n.ID), core.GROWH)
			core.UIOutputPort(n, 1)
		})
		clay.CLAY(clay.IDI("Row3", n.ID), clay.EL{
			Layout: clay.LAY{Sizing: core.GROWH, ChildAlignment: core.YCENTER},
		}, func() {
			core.UIInputPort(n, 2)
		})
	})
}

func (a *HTTPRequestAction) Run(n *core.Node) <-chan core.NodeActionResult {
	done := make(chan core.NodeActionResult, 1)
	go func() {
		// Get Inputs
		// URL is required (index 0)
		valURL, _, errURL := n.GetInputValue(0)
		if errURL != nil {
			done <- core.NodeActionResult{Err: fmt.Errorf("bad URL input: %v", errURL)}
			return
		}
		urlStr := string(valURL.BytesValue)
		if urlStr == "" {
			done <- core.NodeActionResult{Err: fmt.Errorf("URL cannot be empty")}
			return
		}

		// Method (index 1) - Optional, default "GET"
		method := "GET"
		if valMethod, wired, err := n.GetInputValue(1); err == nil && wired {
			m := string(valMethod.BytesValue)
			if m != "" {
				method = strings.ToUpper(m)
			}
		} else if err != nil {
			done <- core.NodeActionResult{Err: fmt.Errorf("bad Method input: %v", err)}
			return
		}

		// Body (index 2) - Optional
		var bodyReader io.Reader
		if valBody, wired, err := n.GetInputValue(2); err == nil && wired {
			if len(valBody.BytesValue) > 0 {
				bodyReader = bytes.NewReader(valBody.BytesValue)
			}
		} else if err != nil {
			done <- core.NodeActionResult{Err: fmt.Errorf("bad Body input: %v", err)}
			return
		}

		// Create Request
		// We use context.Background() here because Run() doesn't pass ctx.
		// Use RunContext if available.
		req, err := http.NewRequest(method, urlStr, bodyReader)
		if err != nil {
			done <- core.NodeActionResult{Err: fmt.Errorf("failed to create request: %v", err)}
			return
		}

		// Execute
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			done <- core.NodeActionResult{Err: fmt.Errorf("request failed: %v", err)}
			return
		}
		defer func() { _ = resp.Body.Close() }()

		// Read Response
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			done <- core.NodeActionResult{Err: fmt.Errorf("failed to read response body: %v", err)}
			return
		}

		done <- core.NodeActionResult{
			Outputs: []core.FlowValue{
				core.NewInt64Value(int64(resp.StatusCode), 0),
				core.NewBytesValue(respBody),
			},
		}
	}()
	return done
}

func (a *HTTPRequestAction) RunContext(ctx context.Context, n *core.Node) <-chan core.NodeActionResult {
	// Reimplement logic with Context support
	done := make(chan core.NodeActionResult, 1)
	go func() {
		// Get Inputs
		valURL, _, errURL := n.GetInputValue(0)
		if errURL != nil {
			done <- core.NodeActionResult{Err: fmt.Errorf("bad URL input: %v", errURL)}
			return
		}
		urlStr := string(valURL.BytesValue)
		if urlStr == "" {
			done <- core.NodeActionResult{Err: fmt.Errorf("URL cannot be empty")}
			return
		}

		method := "GET"
		if valMethod, wired, err := n.GetInputValue(1); err == nil && wired {
			m := string(valMethod.BytesValue)
			if m != "" {
				method = strings.ToUpper(m)
			}
		} else if err != nil {
			done <- core.NodeActionResult{Err: fmt.Errorf("bad Method input: %v", err)}
			return
		}

		if core.DebugMode {
			fmt.Printf("DEBUG: HTTP %s %s\n", method, urlStr)
		}

		var bodyReader io.Reader
		if valBody, wired, err := n.GetInputValue(2); err == nil && wired {
			if len(valBody.BytesValue) > 0 {
				bodyReader = bytes.NewReader(valBody.BytesValue)
			}
		} else if err != nil {
			done <- core.NodeActionResult{Err: fmt.Errorf("bad Body input: %v", err)}
			return
		}

		req, err := http.NewRequestWithContext(ctx, method, urlStr, bodyReader)
		if err != nil {
			done <- core.NodeActionResult{Err: fmt.Errorf("failed to create request: %v", err)}
			return
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			if core.DebugMode {
				fmt.Printf("DEBUG: HTTP Request Failed: %v\n", err)
			}
			done <- core.NodeActionResult{Err: fmt.Errorf("request failed: %v", err)}
			return
		}
		if core.DebugMode {
			fmt.Printf("DEBUG: HTTP Response Status: %s\n", resp.Status)
		}
		defer func() { _ = resp.Body.Close() }()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			done <- core.NodeActionResult{Err: fmt.Errorf("failed to read response body: %v", err)}
			return
		}

		done <- core.NodeActionResult{
			Outputs: []core.FlowValue{
				core.NewInt64Value(int64(resp.StatusCode), 0),
				core.NewBytesValue(respBody),
			},
		}
	}()
	return done
}

func (a *HTTPRequestAction) Serialize(s *core.Serializer) bool {
	return s.Ok()
}
