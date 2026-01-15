package nodes

import (
	"context"

	"github.com/bvisness/flowshell/app/core"
	"github.com/bvisness/flowshell/clay"
	hook "github.com/robotn/gohook"
)

// GEN:NodeAction
type GetMousePositionAction struct{}

func NewGetMousePositionNode() *core.Node {
	return &core.Node{
		Name: "Get Mouse Position",
		OutputPorts: []core.NodePort{
			{Name: "X", Type: core.FlowType{Kind: core.FSKindInt64}},
			{Name: "Y", Type: core.FlowType{Kind: core.FSKindInt64}},
		},
		Action: &GetMousePositionAction{},
	}
}

var _ core.NodeAction = &GetMousePositionAction{}

func (a *GetMousePositionAction) UpdateAndValidate(n *core.Node) {
	n.Valid = true
}

func (a *GetMousePositionAction) UI(n *core.Node) {
	clay.CLAY(clay.IDI("GetMousePosition", n.ID), clay.EL{
		Layout: clay.LAY{LayoutDirection: clay.TopToBottom, Sizing: core.GROWH, ChildGap: core.S2},
	}, func() {
		clay.CLAY(clay.IDI("Row1", n.ID), clay.EL{
			Layout: clay.LAY{Sizing: core.GROWH, ChildAlignment: core.YCENTER, ChildGap: core.S2},
		}, func() {
			core.UISpacer(clay.IDI("PosSpacer1", n.ID), core.GROWH)
			clay.TEXT("X", clay.TextElementConfig{TextColor: core.White})
			core.UIOutputPort(n, 0)
		})
		clay.CLAY(clay.IDI("Row2", n.ID), clay.EL{
			Layout: clay.LAY{Sizing: core.GROWH, ChildAlignment: core.YCENTER, ChildGap: core.S2},
		}, func() {
			core.UISpacer(clay.IDI("PosSpacer2", n.ID), core.GROWH)
			clay.TEXT("Y", clay.TextElementConfig{TextColor: core.White})
			core.UIOutputPort(n, 1)
		})
	})
}

func (a *GetMousePositionAction) Run(n *core.Node) <-chan core.NodeActionResult {
	// Not used since we use RunContext
	return nil
}

func (a *GetMousePositionAction) RunContext(ctx context.Context, n *core.Node) <-chan core.NodeActionResult {
	// Not implemented yet
	return nil
}

func (a *GetMousePositionAction) Serialize(s *core.Serializer) bool {
	return s.Ok()
}

// GEN:NodeAction
type WaitForClickAction struct{}

func NewWaitForClickNode() *core.Node {
	return &core.Node{
		Name: "Wait For Click",
		OutputPorts: []core.NodePort{
			{Name: "X", Type: core.FlowType{Kind: core.FSKindInt64}},
			{Name: "Y", Type: core.FlowType{Kind: core.FSKindInt64}},
		},
		Action: &WaitForClickAction{},
	}
}

var _ core.NodeAction = &WaitForClickAction{}

func (a *WaitForClickAction) UpdateAndValidate(n *core.Node) {
	n.Valid = true
}

func (a *WaitForClickAction) UI(n *core.Node) {
	clay.CLAY(clay.IDI("WaitForClick", n.ID), clay.EL{
		Layout: clay.LAY{LayoutDirection: clay.TopToBottom, Sizing: core.GROWH, ChildGap: core.S2},
	}, func() {
		clay.CLAY(clay.IDI("Row1", n.ID), clay.EL{
			Layout: clay.LAY{Sizing: core.GROWH, ChildAlignment: core.YCENTER, ChildGap: core.S2},
		}, func() {
			core.UISpacer(clay.IDI("ClickSpacer1", n.ID), core.GROWH)
			clay.TEXT("X", clay.TextElementConfig{TextColor: core.White})
			core.UIOutputPort(n, 0)
		})
		clay.CLAY(clay.IDI("Row2", n.ID), clay.EL{
			Layout: clay.LAY{Sizing: core.GROWH, ChildAlignment: core.YCENTER, ChildGap: core.S2},
		}, func() {
			core.UISpacer(clay.IDI("ClickSpacer2", n.ID), core.GROWH)
			clay.TEXT("Y", clay.TextElementConfig{TextColor: core.White})
			core.UIOutputPort(n, 1)
		})
	})
}

func (a *WaitForClickAction) Run(n *core.Node) <-chan core.NodeActionResult {
	return nil
}

func (a *WaitForClickAction) RunContext(ctx context.Context, n *core.Node) <-chan core.NodeActionResult {
	done := make(chan core.NodeActionResult, 1)
	go func() {
		// Just reuse the logic above?
		// Wait, GetMousePosition should ideally result *immediately*.
		// But without robotgo (automation), we can't query.
		// So `NewGetMousePositionNode` is misnamed if it waits.
		// Let's rename the first one to "Wait For Click" and maybe drop the second,
		// or make one that waits for *ANY* mouse movement (Wait For Move).

		// I'll implement "Wait For Click" specifically.

		hooks := hook.Start()
		defer hook.End()

		var x, y int16

		found := false
		for !found {
			select {
			case <-ctx.Done():
				done <- core.NodeActionResult{Err: ctx.Err()}
				return
			case ev := <-hooks:
				if ev.Kind == hook.MouseDown && ev.Button == hook.MouseMap["left"] {
					x = ev.X
					y = ev.Y
					found = true
				}
			}
		}

		done <- core.NodeActionResult{
			Outputs: []core.FlowValue{
				core.NewInt64Value(int64(x), 0),
				core.NewInt64Value(int64(y), 0),
			},
		}
	}()
	return done
}

func (a *WaitForClickAction) Serialize(s *core.Serializer) bool {
	return s.Ok()
}
