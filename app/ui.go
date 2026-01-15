package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/bvisness/flowshell/app/core"
	"github.com/bvisness/flowshell/app/nodes"
	"github.com/bvisness/flowshell/clay"
	"github.com/bvisness/flowshell/util"
	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/lithammer/fuzzysearch/fuzzy"
)

type V2 = core.V2

const MenuMinWidth = 200
const MenuMaxWidth = 0 // no max

const GridSize = 20

func SnapToGrid(v V2) V2 {
	return V2{
		X: float32(int(v.X/GridSize) * GridSize),
		Y: float32(int(v.Y/GridSize) * GridSize),
	}
}

const NodeMinWidth = 360

var CurrentGraph = core.NewGraph()
var CurrentFilename string
var History *HistoryManager
var debugFrameCount int

func InitHistory() {
	History = NewHistoryManager(CurrentGraph)
	core.PushHistoryFunc = PushHistory
}

func PushHistory() {
	if History == nil {
		InitHistory()
	}
	History.Push(CurrentGraph)
}

type NodeType struct {
	Name         string
	Category     string
	Create       func() *core.Node
	ShortcutKey  int32
	ShortcutMods []int32
}

var nodeTypes = []NodeType{
	{Name: "Run Process", Category: "Core", Create: func() *core.Node { return nodes.NewRunProcessNode(util.Tern(runtime.GOOS == "Windows", "dir", "ls")) }},
	{Name: "List Files", Category: "File System", Create: func() *core.Node { return nodes.NewListFilesNode(".") }},
	{Name: "Copy File", Category: "File System", Create: func() *core.Node { return nodes.NewCopyFileNode() }},
	{Name: "Move File", Category: "File System", Create: func() *core.Node { return nodes.NewMoveFileNode() }},
	{Name: "Delete File", Category: "File System", Create: func() *core.Node { return nodes.NewDeleteFileNode() }},
	{Name: "Make Directory", Category: "File System", Create: func() *core.Node { return nodes.NewMakeDirNode() }},
	{Name: "Lines", Category: "Text", Create: func() *core.Node { return nodes.NewLinesNode() }},
	{Name: "Load File", Category: "File System", Create: func() *core.Node { return nodes.NewLoadFileNode("") }},
	{Name: "Save File", Category: "File System", Create: func() *core.Node { return nodes.NewSaveFileNode() }},
	{Name: "Trim Spaces", Category: "Text", Create: func() *core.Node { return nodes.NewTrimSpacesNode() }},
	{Name: "Min", Category: "Math", Create: func() *core.Node { return nodes.NewAggregateNode("Min") }},
	{Name: "Max", Category: "Math", Create: func() *core.Node { return nodes.NewAggregateNode("Max") }},
	{Name: "Mean (Average)", Category: "Math", Create: func() *core.Node { return nodes.NewAggregateNode("Mean") }},
	{Name: "Concatenate Tables", Category: "Table", Create: func() *core.Node { return nodes.NewConcatTablesNode() }},
	{Name: "Filter Empty", Category: "Table", Create: func() *core.Node { return nodes.NewFilterEmptyNode() }},
	{Name: "Sort", Category: "Table", Create: func() *core.Node { return nodes.NewSortNode() }},
	{Name: "Select Columns", Category: "Table", Create: func() *core.Node { return nodes.NewSelectColumnsNode() }},
	{Name: "Extract Column", Category: "Table", Create: func() *core.Node { return nodes.NewExtractColumnNode() }},
	{Name: "Add Column", Category: "Table", Create: func() *core.Node { return nodes.NewAddColumnNode() }},
	{Name: "Convert Type", Category: "Data", Create: func() *core.Node { return nodes.NewConvertNode() }},
	{Name: "Transpose", Category: "Table", Create: func() *core.Node { return nodes.NewTransposeNode() }},
	{Name: "Minify HTML", Category: "Text", Create: func() *core.Node { return nodes.NewMinifyHTMLNode() }},
	{Name: "Wait For Click", Category: "Core", Create: func() *core.Node { return nodes.NewWaitForClickNode() }},
	{Name: "Regex Match", Category: "Regex", Create: func() *core.Node { return nodes.NewRegexMatchNode() }},
	{Name: "Regex Find All", Category: "Regex", Create: func() *core.Node { return nodes.NewRegexFindAllNode() }},
	{Name: "Regex Replace", Category: "Regex", Create: func() *core.Node { return nodes.NewRegexReplaceNode() }},
	{Name: "Regex Split", Category: "Regex", Create: func() *core.Node { return nodes.NewRegexSplitNode() }},
	{Name: "HTTP Request", Category: "Network", Create: func() *core.Node { return nodes.NewHTTPRequestNode() }},
	{Name: "Formula", Category: "Math", Create: func() *core.Node { return nodes.NewFormulaNode() }},
	{Name: "Join Text", Category: "Text", Create: func() *core.Node { return nodes.NewJoinTextNode() }},
	{Name: "Split Text", Category: "Text", Create: func() *core.Node { return nodes.NewSplitTextNode() }},
	{Name: "Change Case", Category: "Text", Create: func() *core.Node { return nodes.NewCaseConvertNode() }},
	{Name: "Format String", Category: "Text", Create: func() *core.Node { return nodes.NewFormatStringNode() }},
	{Name: "Parse Time", Category: "Data", Create: func() *core.Node { return nodes.NewParseTimeNode() }},
	{Name: "JSON Query", Category: "Data", Create: func() *core.Node { return nodes.NewJsonQueryNode() }},
	{Name: "XML Query", Category: "Data", Create: func() *core.Node { return nodes.NewXmlQueryNode() }},
	{Name: "Get Variable", Category: "Core", Create: func() *core.Node { return nodes.NewGetVariableNode() }},
	{Name: "Map", Category: "Table", Create: func() *core.Node { return nodes.NewMapNode() }},
	{Name: "Graph Input", Category: "Graph", Create: func() *core.Node { return nodes.NewGraphInputNode() }},
	{Name: "Graph Output", Category: "Graph", Create: func() *core.Node { return nodes.NewGraphOutputNode() }},
	{Name: "Line Chart", Category: "Visualization", Create: func() *core.Node { return nodes.NewLineChartNode() }},
	{Name: "Bar Chart", Category: "Visualization", Create: func() *core.Node { return nodes.NewBarChartNode() }},
	{Name: "Scatter Plot", Category: "Visualization", Create: func() *core.Node { return nodes.NewScatterPlotNode() }},
	{Name: "Prompt User", Category: "Core", Create: func() *core.Node { return nodes.NewPromptUserNode() }},
	{Name: "If / Else", Category: "Logic", Create: func() *core.Node { return nodes.NewIfElseNode() }},
	{Name: "Gate", Category: "Logic", Create: func() *core.Node { return nodes.NewGateNode() }},
	{Name: "Merge", Category: "Logic", Create: func() *core.Node { return nodes.NewMergeNode() }},
}

func init() {
	slices.SortFunc(nodeTypes, func(a, b NodeType) int {
		if c := strings.Compare(a.Category, b.Category); c != 0 {
			return c
		}
		return strings.Compare(a.Name, b.Name)
	})
}

func CreateGroup() {
	core.PushHistory()
	// Calculate bounding box of selected nodes
	if len(SelectedNodes) == 0 {
		return
	}

	minX, minY := float32(1e9), float32(1e9)
	maxX, maxY := float32(-1e9), float32(-1e9)

	for id := range SelectedNodes {
		if n, ok := CurrentGraph.GetNode(id); ok {
			minX = min(minX, n.Pos.X)
			minY = min(minY, n.Pos.Y)
			maxX = max(maxX, n.Pos.X+NodeMinWidth) // Approx width
			// Height is variable, but let's assume a safe buffer or use n.DragRect if available
			// n.DragRect is updated in layout, so it might be stale or 0 if not rendered yet?
			// But since we are selecting them, they must be rendered.
			h := n.DragRect.Height
			if h == 0 {
				h = 100
			}
			maxY = max(maxY, n.Pos.Y+h)
		}
	}

	// Add padding
	pad := float32(20)
	rect := rl.Rectangle{
		X:      minX - pad,
		Y:      minY - pad - 40, // Extra top padding for header
		Width:  (maxX - minX) + pad*2,
		Height: (maxY - minY) + pad*2 + 40,
	}

	CurrentGraph.AddGroup(&core.Group{
		Title: "New core.Group",
		Rect:  rect,
		Color: clay.Color{R: 100, G: 100, B: 100, A: 255}, // Default gray
	})
}

func parseNodeSearch(input string) (string, int) {
	input = strings.TrimSpace(input)
	parts := strings.SplitN(input, " ", 2)
	if len(parts) > 1 {
		if count, err := strconv.Atoi(parts[0]); err == nil && count > 0 {
			return parts[1], count
		}
	}
	return input, 1
}

func SearchNodeTypes(searchInput string) []NodeType {
	search, _ := parseNodeSearch(searchInput)
	var names []string
	for _, t := range nodeTypes {
		names = append(names, t.Name)
	}

	if search == "" || search == "?" || search == "*" {
		return nodeTypes
	}

	ranks := fuzzy.RankFindFold(search, names)
	slices.SortStableFunc(ranks, func(a, b fuzzy.Rank) int {
		return a.Distance - b.Distance
	})
	var res []NodeType
nextrank:
	for _, rank := range ranks {
		for _, t := range nodeTypes {
			if t.Name == rank.Target {
				res = append(res, t)
				continue nextrank
			}
		}
	}
	return res
}

var selectedNodeID = 0
var SelectedNodes = make(map[int]struct{})

func IsNodeSelected(id int) bool {
	_, ok := SelectedNodes[id]
	return ok
}

func SelectNode(id int, multi bool) {
	if !multi {
		for k := range SelectedNodes {
			delete(SelectedNodes, k)
		}
	}
	SelectedNodes[id] = struct{}{}
	selectedNodeID = id
}

func ToggleSelectNode(id int) {
	if _, ok := SelectedNodes[id]; ok {
		delete(SelectedNodes, id)
		if selectedNodeID == id {
			selectedNodeID = 0
			// Pick another one?
			for k := range SelectedNodes {
				selectedNodeID = k
				break
			}
		}
	} else {
		SelectedNodes[id] = struct{}{}
		selectedNodeID = id
	}
}

func GetSelectedNode() (*core.Node, bool) {
	return CurrentGraph.GetNode(selectedNodeID)
}

func DeleteNode(id int) {
	CurrentGraph.DeleteNode(id)
}

func DeleteSelectedNodes() {
	core.PushHistory()
	for id := range SelectedNodes {
		DeleteNode(id)
	}
	for k := range SelectedNodes {
		delete(SelectedNodes, k)
	}
	selectedNodeID = 0
}

func DuplicateNode(original *core.Node) {
	core.PushHistory()
	// Clone via Serialization
	s := core.NewEncoder(1) // version 1
	original.Serialize(s)
	data := s.Bytes()

	sRead := core.NewDecoder(data)

	// Create new "empty" node structure
	clone := &core.Node{}
	// We need to initialize Action via helper or manually meta-alloc?
	// Serialize() on decode expects n.Action to be allocated?
	// No, the default Serialize implementation allocates it:
	// n.Action = meta.Alloc()
	// So we just need an empty core.Node.

	if clone.Serialize(sRead) {
		// Post-processing
		clone.ID = 0 // Will be assigned by AddNode
		clone.Graph = nil
		clone.Pos = V2(clay.V2(clone.Pos).Plus(clay.V2{X: 20, Y: 20})) // Offset
		clone.Pos = SnapToGrid(clone.Pos)
		// clone.outputState = nil // Reset state
		clone.Running = false
		clone.Valid = false

		CurrentGraph.AddNode(clone)
		SelectNode(clone.ID, false)
	} else {
		fmt.Println("Failed to duplicate node: serialization error")
	}
}

// var core.UICursor rl.MouseCursor

// Focus tracking for Undo/Redo
// var core.LastUIFocus clay.ElementID
// var core.LastUIFocusValid bool

// var core.UIFocus *clay.ElementID

// var core.CurrentZIndex int16

func processInput() {
	if rl.IsMouseButtonPressed(rl.MouseRightButton) && !core.IsHoveringPanel && !core.IsHoveringUI {
		// Check if we clicked on a node for Context Menu
		clickedNode := false
		for _, n := range CurrentGraph.Nodes {
			// Hit test using DragRect (header) or full body if we can guess it?
			// DragRect is safest for "Select/Interact".
			// But user might right click the body.
			// Let's rely on clay.GetElementData for the node's main ID "core.Node<ID>"
			if data, ok := clay.GetElementData(n.ClayID()); ok {
				if rl.CheckCollisionPointRec(rl.GetMousePosition(), rl.Rectangle(data.BoundingBox)) {
					clickedNode = true

					// Capture loop var
					node := n

					// Open Context Menu
					items := []ContextMenuItem{
						{Label: "Run", Action: func() { node.Run(context.Background(), false) }},
						{Label: util.Tern(node.Pinned, "Unpin", "Pin"), Action: func() {
							core.PushHistory()
							node.Pinned = !node.Pinned
						}},
						{Label: "Duplicate", Action: func() { DuplicateNode(node) }}, // DuplicateNode calls core.PushHistory
						{Label: "Delete", Action: func() {
							// DeleteSelectedNodes calls core.PushHistory, but here we might delete a single node
							// that isn't selected? Or we should select it first?
							// Logic below deletes node n.ID.
							core.PushHistory()
							DeleteNode(node.ID)
							if IsNodeSelected(node.ID) {
								delete(SelectedNodes, node.ID)
								selectedNodeID = 0
							}
						}},
					}

					ContextMenu = &ContextMenuState{
						Pos:    V2(rl.GetMousePosition()),
						NodeID: node.ID,
						Items:  items,
					}
					break
				}
			}
		}

		if !clickedNode {
			// If not clicking a node, check if we right-clicked the background to open global context menu
			if rl.IsMouseButtonPressed(rl.MouseButtonRight) && !core.IsHoveringUI {
				items := []ContextMenuItem{
					{Label: "Paste", Action: func() {
						core.PushHistory()
						Paste()
					}},
					{Label: "Auto Layout", Action: func() {
						core.PushHistory()
						LayoutGraph(CurrentGraph)
					}},
				}

				ContextMenu = &ContextMenuState{
					Pos:   V2(rl.GetMousePosition()),
					Items: items,
				}
			} else if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
				// Only clear context menu / focus on Left Click
				ContextMenu = nil

				isGroup := false
				for _, grp := range CurrentGraph.Groups {
					if rl.CheckCollisionPointRec(rl.GetMousePosition(), grp.Rect) {
						isGroup = true
						break
					}
				}

				if !isGroup {
					NewNodeName = ""
					id := clay.ID("NewNodeName")
					core.UIFocus = &id
				}
			}
		}
	} else if rl.IsMouseButtonPressed(rl.MouseLeftButton) && !core.IsHoveringPanel && !core.IsHoveringUI {
		// Close context menu on left click outside
		ContextMenu = nil
	}

	// Double check to clear focus if clicking background
	if rl.IsMouseButtonPressed(rl.MouseButtonLeft) && !core.IsHoveringPanel && !core.IsHoveringUI {
		core.UIFocus = nil
	}

	// Undo/Redo
	actions := rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl)
	if actions && rl.IsKeyPressed(rl.KeyZ) {
		if rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift) {
			if g := History.Redo(); g != nil {
				CurrentGraph = g
				// Clear selection on undo/redo to avoid ghost selections
				// Or try to restore? Restoring is hard.
				clear(SelectedNodes)
				selectedNodeID = 0
			}
		} else {
			if g := History.Undo(); g != nil {
				CurrentGraph = g
				clear(SelectedNodes)
				selectedNodeID = 0
			}
		}
	}
	if actions && rl.IsKeyPressed(rl.KeyY) {
		if g := History.Redo(); g != nil {
			CurrentGraph = g
			clear(SelectedNodes)
			selectedNodeID = 0
		}
	}

	// Save Graph
	if rl.IsKeyPressed(rl.KeyS) && rl.IsKeyDown(rl.KeyLeftControl) {
		initialDir, _ := os.Getwd()
		if CurrentFilename != "" {
			initialDir = filepath.Dir(CurrentFilename)
		}
		filename, ok, err := core.SaveFileDialog("Save Flow", initialDir, map[string]string{"flow": "Flow Files"})
		if err != nil {
			fmt.Printf("Save error: %v\n", err)
		} else if ok {
			// Ensure extension
			if filepath.Ext(filename) != ".flow" {
				filename += ".flow"
			}
			err := core.SaveGraph(filename, CurrentGraph)
			if err != nil {
				fmt.Printf("Failed to save: %v\n", err)
			}
		}
	}

	// Load Graph
	if rl.IsKeyPressed(rl.KeyL) && rl.IsKeyDown(rl.KeyLeftControl) {
		cwd, _ := os.Getwd()
		filename, ok, err := core.OpenFileDialog("Open Flow", cwd, map[string]string{"flow": "Flow Files"})
		if err != nil {
			fmt.Printf("Load error: %v\n", err)
			core.ShowInfoDialog("Error", fmt.Sprintf("Failed to open file dialog: %v", err))
		} else if ok {
			core.PushHistory()
			if g, err := core.LoadGraph(filename); err == nil {
				CurrentGraph = g
				clear(SelectedNodes)
				selectedNodeID = 0
			} else {
				fmt.Printf("Failed to load: %v\n", err)
			}
		}
	}

	if rl.IsKeyPressed(rl.KeyC) && (rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl)) {
		Copy()
	}
	if rl.IsKeyPressed(rl.KeyV) && (rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl)) {
		core.PushHistory()
		Paste()
	}

	if rl.IsKeyPressed(rl.KeyG) && (rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl)) {
		CreateGroup()
	}

	// Create node shortcuts
	if !core.IsHoveringUI && core.UIFocus == nil {
		for _, nt := range nodeTypes {
			if nt.ShortcutKey != 0 {
				modsDown := true
				for _, mod := range nt.ShortcutMods {
					if !rl.IsKeyDown(mod) {
						modsDown = false
						break
					}
				}
				if modsDown && rl.IsKeyPressed(nt.ShortcutKey) {
					core.PushHistory()
					newNode := nt.Create()
					// Create at mouse position
					newNode.Pos = SnapToGrid(V2(rl.GetMousePosition()))
					CurrentGraph.AddNode(newNode)
					selectedNodeID = newNode.ID
				}
			}
		}
	}

	if rl.IsKeyPressed(rl.KeyDelete) && core.UIFocus == nil && len(SelectedNodes) > 0 {
		DeleteSelectedNodes()
	}

	if rl.IsFileDropped() {
		files := rl.LoadDroppedFiles()
		handled := false

		// Check if we dropped onto a Load File node
		if len(files) > 0 && core.IsHoveringPanel {
			mousePos := rl.GetMousePosition()
			// Find the top-most node under the mouse
			for i := len(CurrentGraph.Nodes) - 1; i >= 0; i-- {
				n := CurrentGraph.Nodes[i]
				if data, ok := clay.GetElementData(n.ClayID()); ok {
					if rl.CheckCollisionPointRec(mousePos, rl.Rectangle(data.BoundingBox)) {
						if loadAction, ok := n.Action.(*nodes.LoadFileAction); ok {
							loadAction.Path = files[0]
							// Auto-select format based on extension
							if ext := strings.ToLower(filepath.Ext(files[0])); ext != "" {
								loadAction.Format.SelectByValue(ext[1:])
							}
							handled = true
							// Force re-validation
							n.ClearResult()
						}
						break // Only handle for the top-most node (or first found)
					}
				}
			}
		}

		if !handled && len(files) > 0 {
			// Check for .flow file import
			ext := strings.ToLower(filepath.Ext(files[0]))
			if ext == ".flow" {
				core.PushHistory()
				if g, err := core.LoadGraph(files[0]); err == nil {
					core.MergeGraph(CurrentGraph, g)
					handled = true
				} else {
					fmt.Printf("Failed to load graph: %v\n", err)
				}
			}
		}

		if !handled && !core.IsHoveringUI && !core.IsHoveringPanel {
			core.PushHistory()
			for i, filename := range files {
				n := nodes.NewLoadFileNode(filename)
				// Initial position at mouse
				pos := V2(rl.GetMousePosition())

				// Apply offset: 0 for first node, then move down (y-axis) for subsequent nodes
				// NodeMinWidth + GridSize for spacing, or just vertical spacing?
				// User asked for "align in vertical with a little space".
				// Let's assume a vertical stack.
				// Height is tricky without rendering, but let's assume ~150px per node.
				const EstimatedNodeHeight = 180

				pos = V2(clay.V2(pos).Plus(clay.V2{X: 0, Y: float32(i * EstimatedNodeHeight)}))

				if !rl.IsKeyDown(rl.KeyLeftShift) && !rl.IsKeyDown(rl.KeyRightShift) {
					n.Pos = SnapToGrid(pos)
				} else {
					n.Pos = pos
				}
				CurrentGraph.AddNode(n)
				SelectNode(n.ID, i > 0)
			}
		}
	}

	// Selection on Mouse Down (Immediate feedback)
	for _, n := range CurrentGraph.Nodes {
		if core.UIInput.IsPressed(n.ClayID()) {
			multi := rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift) || rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl)
			if !IsNodeSelected(n.ID) {
				SelectNode(n.ID, multi)
			}
		}
	}

	for _, n := range CurrentGraph.Nodes {
		// Starting new wires (Prioritize over node drag)
		for i, portPos := range n.OutputPortPositions {
			portRect := rl.Rectangle{
				X:      portPos.X - PortDragRadius,
				Y:      portPos.Y - PortDragRadius,
				Width:  PortDragRadius * 2,
				Height: PortDragRadius * 2,
			}
			if !core.IsHoveringUI && core.Drag.TryStartDrag(NewWireDragKey, portRect, V2{}) {
				NewWireSourceNode = n
				NewWireSourcePort = i
			}
		}
		for i, portPos := range n.InputPortPositions {
			portRect := rl.Rectangle{
				X:      portPos.X - PortDragRadius,
				Y:      portPos.Y - PortDragRadius,
				Width:  PortDragRadius * 2,
				Height: PortDragRadius * 2,
			}
			if wire, hasWire := n.GetInputWire(i); hasWire {
				if !core.IsHoveringUI && core.Drag.TryStartDrag(NewWireDragKey, portRect, V2{}) {
					CurrentGraph.Wires = slices.DeleteFunc(CurrentGraph.Wires, func(w *core.Wire) bool { return w == wire })
					NewWireSourceNode = wire.StartNode
					NewWireSourcePort = wire.StartPort
				}
			}
		}

		// core.Node drag and drop
		if !core.IsHoveringUI {
			if core.Drag.TryStartDrag(n, n.DragRect, n.Pos) {
				core.PushHistory()
			}
		}
		if draggingThisNode, done, canceled := core.Drag.State(n); draggingThisNode {
			// If we start dragging a node that isn't selected, select it (and deselect others)
			// UNLESS we are dragging a selected node, then we move all selected nodes.
			if !IsNodeSelected(n.ID) {
				SelectNode(n.ID, false)
			}

			// Calculate delta
			delta := rl.Vector2Subtract(core.Drag.NewObjPosition(), n.Pos)

			// Apply delta to all selected nodes
			for id := range SelectedNodes {
				if node, ok := CurrentGraph.GetNode(id); ok {
					node.Pos = rl.Vector2Add(node.Pos, delta)
					// Snap to grid (hold Shift to disable)
					if !rl.IsKeyDown(rl.KeyLeftShift) && !rl.IsKeyDown(rl.KeyRightShift) {
						node.Pos = SnapToGrid(node.Pos)
					}
				}

			}

			if done {
				if canceled {
					n.Pos = core.Drag.ObjStart
				}
			}
		}

		// Selected node keyboard shortcuts
		if IsNodeSelected(n.ID) {
			if rl.IsKeyPressed(rl.KeyR) && (rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl)) {
				n.Run(context.Background(), rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift))
			}
		}
	}

	// Dropping new wires
	if draggingNewWire, done, canceled := core.Drag.State(NewWireDragKey); draggingNewWire {
		if done && !canceled {
			// Loop over nodes to find any you may have dropped on
			for _, node := range CurrentGraph.Nodes {
				for port, portPos := range node.InputPortPositions {
					portRect := rl.Rectangle{
						X:      portPos.X - PortDragRadius,
						Y:      portPos.Y - PortDragRadius,
						Width:  PortDragRadius * 2,
						Height: PortDragRadius * 2,
					}

					if rl.CheckCollisionPointRec(rl.GetMousePosition(), portRect) && node != NewWireSourceNode {
						// Check types
						sourceType := NewWireSourceNode.OutputPorts[NewWireSourcePort].Type
						targetType := node.InputPorts[port].Type
						if err := core.Typecheck(sourceType, targetType); err != nil {
							fmt.Printf("Cannot connect: %v\n", err)
							ConnectionError = fmt.Sprintf("Cannot connect: %s", err.Error())
							ConnectionErrorTime = time.Now()
						} else {
							// Delete existing wires into that port
							if len(CurrentGraph.Wires) > 0 { // Optimization or check required?
								// Just push history on wire modify
								core.PushHistory()
							}
							CurrentGraph.Wires = slices.DeleteFunc(CurrentGraph.Wires, func(wire *core.Wire) bool {
								return wire.EndNode == node && wire.EndPort == port
							})
							CurrentGraph.Wires = append(CurrentGraph.Wires, &core.Wire{
								StartNode: NewWireSourceNode, EndNode: node,
								StartPort: NewWireSourcePort, EndPort: port,
							})
						}
						node.ClearResult()
					}
				}
			}
		}
	}

	// Dropping core.FlowValue
	if draggingValue, done, canceled := core.Drag.State("FLOW_VALUE_DRAG"); draggingValue {
		if done && !canceled {
			if dragValue, ok := core.Drag.Thing.(FlowValueDrag); ok {
				n := nodes.NewValueNode(dragValue.Value)
				n.Pos = V2(rl.GetMousePosition())
				if !rl.IsKeyDown(rl.KeyLeftShift) && !rl.IsKeyDown(rl.KeyRightShift) {
					n.Pos = SnapToGrid(n.Pos)
				}
				core.PushHistory()
				CurrentGraph.AddNode(n)
				selectedNodeID = n.ID
			}
		}
	}

	// Resizing output window
	if !IsOutputCollapsed {
		outputWindowRect := rl.Rectangle{
			X:      float32(rl.GetScreenWidth()) - OutputWindowWidth - OutputWindowDragWidth/2,
			Y:      0,
			Width:  OutputWindowDragWidth,
			Height: float32(rl.GetScreenHeight()),
		}
		core.Drag.TryStartDrag(OutputWindowDragKey, outputWindowRect, V2{X: OutputWindowWidth, Y: 0})

		resizing, done, canceled := core.Drag.State(OutputWindowDragKey)
		if resizing {
			if done {
				if canceled {
					OutputWindowWidth = core.Drag.ObjStart.X
				}
			} else {
				OutputWindowWidth = float32(rl.GetScreenWidth()) - float32(rl.GetMouseX())
			}
		}

		if rl.CheckCollisionPointRec(rl.GetMousePosition(), outputWindowRect) || resizing {
			core.UICursor = rl.MouseCursorResizeEW
		}
	}

	// Box Selection & Panning
	{
		if background, ok := clay.GetElementData(clay.ID("Background")); ok {
			isShift := rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift)

			if !core.IsHoveringUI && !core.IsHoveringPanel {
				if isShift {
					// Snapshot selection
					initial := make(map[int]struct{})
					for k, v := range SelectedNodes {
						initial[k] = v
					}
					core.Drag.TryStartDrag(BoxSelectDrag{InitialSelection: initial}, rl.Rectangle(background.BoundingBox), V2{})
				} else {
					if core.Drag.TryStartDrag(PanDragKey, rl.Rectangle(background.BoundingBox), V2{}) {
						LastPanMousePosition = rl.GetMousePosition()
					}
				}
			}

			if boxSelecting, _, canceled := core.Drag.State(BoxSelectDragKey); boxSelecting {
				if bs, ok := core.Drag.Thing.(BoxSelectDrag); ok {
					if canceled {
						// Restore
						clear(SelectedNodes)
						for k, v := range bs.InitialSelection {
							SelectedNodes[k] = v
						}
					} else {
						// Update selection
						start := core.Drag.MouseStart
						end := rl.GetMousePosition()

						x := min(start.X, end.X)
						y := min(start.Y, end.Y)
						w := max(start.X, end.X) - x
						h := max(start.Y, end.Y) - y
						box := rl.Rectangle{X: x, Y: y, Width: w, Height: h}

						clear(SelectedNodes)
						for k, v := range bs.InitialSelection {
							SelectedNodes[k] = v
						}

						for _, n := range CurrentGraph.Nodes {
							if rl.CheckCollisionRecs(box, n.DragRect) {
								SelectedNodes[n.ID] = struct{}{}
								selectedNodeID = n.ID
							}
						}
					}
				}
			}

			if panning, _, _ := core.Drag.State(PanDragKey); panning {
				mousePos := rl.GetMousePosition()
				delta := rl.Vector2Subtract(mousePos, LastPanMousePosition)
				for _, n := range CurrentGraph.Nodes {
					n.Pos = rl.Vector2Add(n.Pos, delta)
				}
				LastPanMousePosition = mousePos
			}
		}
	}
}

const OutputWindowDragKey = "OUTPUT_WINDOW"
const OutputWindowDragWidth = 8

const PanDragKey = "PAN"
const BoxSelectDragKey = "BOX_SELECT"

type BoxSelectDrag struct {
	InitialSelection map[int]struct{}
}

func (b BoxSelectDrag) DragKey() string {
	return BoxSelectDragKey
}

type FlowValueDrag struct {
	Value core.FlowValue
}

func (f FlowValueDrag) DragKey() string {
	return "FLOW_VALUE_DRAG"
}

var LastPanMousePosition V2

const NewWireDragKey = "NEW_WIRE"
const PortDragRadius = 10

var NewWireSourceNode *core.Node
var NewWireSourcePort int

var NewNodeName string
var SelectedNodeCategory string

var ShowLoadConfirmation bool
var ShowVariables bool
var NewVarKey string
var NewVarValue string

var ConnectionError string
var ConnectionErrorTime time.Time

var ContextMenu *ContextMenuState

type ContextMenuItem struct {
	Label  string
	Action func()
}

type ContextMenuState struct {
	Pos    V2
	NodeID int
	Items  []ContextMenuItem
}

var OutputWindowWidth float32 = windowWidth * 0.30
var IsOutputCollapsed = false

func UpdateGraph() error {
	// Sweep the graph, validating all nodes
	sortedNodes, topoErr := core.Toposort(CurrentGraph.Nodes, CurrentGraph.Wires)
	if topoErr != nil {
		// If there is a cycle, we can't toposort. Just use the default order.
		sortedNodes = CurrentGraph.Nodes
	}
	for _, node := range sortedNodes {
		node.Action.UpdateAndValidate(node)
	}
	return topoErr
}

func UINodes(topoErr error) {
	debugFrameCount++
	clay.CLAY(clay.ID("NodeCanvas"), clay.EL{
		Layout: clay.LAY{Sizing: core.GROWALL},
	}, func() {
		for _, group := range CurrentGraph.Groups {
			UIGroup(group)
		}
		for _, node := range CurrentGraph.Nodes {
			UINode(node, topoErr != nil)
		}
	})
}

func UIOverlay(topoErr error) {
	clay.CLAY(clay.ID("OverlayRoot"), clay.EL{
		Layout: clay.LAY{Sizing: core.GROWALL},
		Floating: clay.FLOAT{
			AttachTo:           clay.AttachToRoot,
			PointerCaptureMode: clay.PointercaptureModePassthrough, // Important: let clicks pass through to nodes if not hitting UI
		},
	}, func() {
		if ShowMenuBar {
			UIMenuBar()
		}
		UIMinimap()

		if topoErr != nil {
			core.WithZIndex(core.Z_CYCLE_WARNING, func() {
				clay.CLAY(clay.ID("CycleWarning"), clay.EL{
					Layout:          clay.LAY{Sizing: clay.Sizing{Width: clay.SizingGrow(1, 0), Height: clay.SizingFixed(30)}, ChildAlignment: core.ALLCENTER},
					BackgroundColor: core.Red,
					Floating: clay.FLOAT{
						AttachTo: clay.AttachToParent,
						AttachPoints: clay.FloatingAttachPoints{
							Element: clay.AttachPointCenterTop,
							Parent:  clay.AttachPointCenterTop,
						},
						ZIndex: core.Z_CYCLE_WARNING,
					},
				}, func() {
					clay.TEXT("Cycle Detected! core.Graph execution disabled.", clay.TextElementConfig{TextColor: core.White, FontID: core.InterBold})
				})
			})
		}

		if ShowLoadConfirmation {
			core.WithZIndex(core.Z_MODAL, func() {
				clay.CLAY(clay.ID("LoadConfirmation"), clay.EL{
					Layout:          clay.LAY{LayoutDirection: clay.TopToBottom, Sizing: clay.Sizing{Width: clay.SizingFixed(300)}, ChildAlignment: core.ALLCENTER, Padding: core.PA3, ChildGap: core.S2},
					BackgroundColor: core.Charcoal,
					Border:          clay.BorderElementConfig{Width: core.BA2, Color: core.Red},
					CornerRadius:    core.RA2,
					Floating: clay.FLOAT{
						AttachTo: clay.AttachToParent,
						AttachPoints: clay.FloatingAttachPoints{
							Element: clay.AttachPointCenterCenter,
							Parent:  clay.AttachPointCenterCenter,
						},
						ZIndex:             core.Z_MODAL,
						PointerCaptureMode: clay.PointercaptureModeCapture,
					},
				}, func() {
					clay.OnHover(func(elementID clay.ElementID, pointerData clay.PointerData, userData any) {
						core.IsHoveringUI = true
					}, nil)
					clay.TEXT("Discard changes and load?", clay.TextElementConfig{TextColor: core.White, FontID: core.InterBold, FontSize: core.F2})
					clay.CLAY(clay.AUTO_ID, clay.EL{Layout: clay.LAY{ChildGap: core.S3}}, func() {
						core.UIButton(clay.ID("ConfirmLoad"), core.UIButtonConfig{
							El: clay.EL{Layout: clay.LAY{Padding: core.PA2}, BackgroundColor: core.Red, CornerRadius: core.RA1},
							OnClick: func(_ clay.ElementID, _ clay.PointerData, _ any) {
								ShowLoadConfirmation = false
								cwd, _ := os.Getwd()
								filename, ok, err := core.OpenFileDialog("Open Flow", cwd, map[string]string{"flow": "Flow Files"})
								if err != nil {
									fmt.Printf("Load error: %v\n", err)
									core.ShowInfoDialog("Error", fmt.Sprintf("Failed to open file dialog: %v", err))
								} else if ok {
									core.PushHistory()
									if g, err := core.LoadGraph(filename); err == nil {
										CurrentGraph = g
										CurrentFilename = filename
										clear(SelectedNodes)
										selectedNodeID = 0
									} else {
										core.ShowInfoDialog("Error", fmt.Sprintf("Failed to load file: %v", err))
									}
								}
							},
						}, func() {
							clay.TEXT("Load", clay.TextElementConfig{TextColor: core.White})
						})
						core.UIButton(clay.ID("CancelLoad"), core.UIButtonConfig{
							El: clay.EL{Layout: clay.LAY{Padding: core.PA2}, BackgroundColor: core.Gray, CornerRadius: core.RA1},
							OnClick: func(_ clay.ElementID, _ clay.PointerData, _ any) {
								ShowLoadConfirmation = false
							},
						}, func() {
							clay.TEXT("Cancel", clay.TextElementConfig{TextColor: core.White})
						})
					})
				})
			})
		}

		// Prompt Modal
		if core.CurrentPrompt != nil {
			core.WithZIndex(core.Z_MODAL, func() {
				// Blocking overlay background
				clay.CLAY(clay.ID("PromptOverlayBlocker"), clay.EL{
					Layout:          clay.LAY{Sizing: core.GROWALL, LayoutDirection: clay.TopToBottom, ChildAlignment: core.ALLCENTER},
					BackgroundColor: clay.Color{R: 0, G: 0, B: 0, A: 100}, // Semi-transparent dim
					Floating: clay.FLOAT{
						AttachTo:           clay.AttachToRoot,
						ZIndex:             core.Z_MODAL,
						PointerCaptureMode: clay.PointercaptureModeCapture, // Block clicks
					},
				}, func() {
					clay.CLAY(clay.ID("PromptModal"), clay.EL{
						Layout:          clay.LAY{LayoutDirection: clay.TopToBottom, Sizing: clay.Sizing{Width: clay.SizingFixed(400)}, ChildAlignment: core.ALLCENTER, Padding: core.PA3, ChildGap: core.S3},
						BackgroundColor: core.Charcoal,
						Border:          clay.BorderElementConfig{Width: core.BA2, Color: core.Blue},
						CornerRadius:    core.RA2,
					}, func() {
						clay.OnHover(func(elementID clay.ElementID, pointerData clay.PointerData, userData any) {
							core.IsHoveringUI = true
						}, nil)

						clay.TEXT(core.CurrentPrompt.Title, clay.TextElementConfig{TextColor: core.White, FontID: core.InterBold, FontSize: core.F2})
						clay.TEXT(core.CurrentPrompt.Message, clay.TextElementConfig{TextColor: core.LightGray})

						core.UITextBox(clay.ID("PromptInput"), &core.CurrentPrompt.DefaultValue, core.UITextBoxConfig{
							El:       clay.EL{Layout: clay.LAY{Sizing: core.GROWH, Padding: core.PA2}, BackgroundColor: core.DarkGray, CornerRadius: core.RA1},
							OnSubmit: func(val string) { core.RespondToPrompt(val) },
						})

						clay.CLAY(clay.AUTO_ID, clay.EL{Layout: clay.LAY{ChildGap: core.S3, LayoutDirection: clay.LeftToRight}}, func() {
							core.UIButton(clay.ID("PromptOK"), core.UIButtonConfig{
								El: clay.EL{Layout: clay.LAY{Padding: core.PA2}, BackgroundColor: core.Blue, CornerRadius: core.RA1},
								OnClick: func(_ clay.ElementID, _ clay.PointerData, _ any) {
									core.RespondToPrompt(core.CurrentPrompt.DefaultValue)
								},
							}, func() {
								clay.TEXT("OK", clay.TextElementConfig{TextColor: core.White})
							})
							core.UIButton(clay.ID("PromptCancel"), core.UIButtonConfig{
								El: clay.EL{Layout: clay.LAY{Padding: core.PA2}, BackgroundColor: core.Gray, CornerRadius: core.RA1},
								OnClick: func(_ clay.ElementID, _ clay.PointerData, _ any) {
									core.CancelPrompt()
								},
							}, func() {
								clay.TEXT("Cancel", clay.TextElementConfig{TextColor: core.White})
							})
						})
					})
				})
			})
		}

		if ShowVariables {
			core.WithZIndex(core.Z_MODAL, func() {
				clay.CLAY(clay.ID("OverlayVariablesPanel"), clay.EL{
					Layout:          clay.LAY{LayoutDirection: clay.TopToBottom, Sizing: clay.Sizing{Width: clay.SizingFixed(400)}, Padding: core.PA3, ChildGap: core.S2},
					BackgroundColor: core.Charcoal,
					Border:          clay.BorderElementConfig{Width: core.BA2, Color: core.Gray},
					CornerRadius:    core.RA2,
					Floating: clay.FLOAT{
						AttachTo: clay.AttachToParent,
						AttachPoints: clay.FloatingAttachPoints{
							Element: clay.AttachPointCenterCenter,
							Parent:  clay.AttachPointCenterCenter,
						},
						ZIndex:             core.Z_MODAL,
						PointerCaptureMode: clay.PointercaptureModeCapture,
					},
				}, func() {
					clay.OnHover(func(elementID clay.ElementID, pointerData clay.PointerData, userData any) {
						core.IsHoveringUI = true
					}, nil)

					// Header
					clay.CLAY(clay.AUTO_ID, clay.EL{Layout: clay.LAY{LayoutDirection: clay.LeftToRight, Sizing: clay.Sizing{Width: clay.SizingGrow(0, 400)}, ChildAlignment: core.YCENTER, ChildGap: core.S2}}, func() {
						clay.TEXT("Settings", clay.TextElementConfig{TextColor: core.White, FontID: core.InterBold, FontSize: core.F2})
						clay.CLAY(clay.AUTO_ID, clay.EL{Layout: clay.LAY{Sizing: core.GROWH}}) // Spacer
						core.UIButton(clay.ID("CloseVariables"), core.UIButtonConfig{          // This ID is inside unique container? No, ID() is global. Rename it?
							El: clay.EL{Layout: clay.LAY{Padding: core.PA1}, BackgroundColor: core.Red, CornerRadius: core.RA1},
							OnClick: func(_ clay.ElementID, _ clay.PointerData, _ any) {
								ShowVariables = false
							},
						}, func() {
							clay.TEXT("X", clay.TextElementConfig{TextColor: core.White})
						})
					})

					// List
					CurrentGraph.VarMutex.RLock()
					keys := make([]string, 0, len(CurrentGraph.Variables))
					for k := range CurrentGraph.Variables {
						keys = append(keys, k)
					}
					slices.Sort(keys)

					// Scrollable area? For now just list them, assuming not too many.
					clay.CLAY(clay.ID("OverlayVariablesList"), clay.EL{
						Layout: clay.LAY{LayoutDirection: clay.TopToBottom, Sizing: clay.Sizing{Width: clay.SizingGrow(0, 400)}, ChildGap: core.S1},
					}, func() {
						for _, k := range keys {
							v := CurrentGraph.Variables[k]
							key := k
							clay.CLAY(clay.ID("VarRow-"+key), clay.EL{
								Layout:          clay.LAY{LayoutDirection: clay.LeftToRight, Sizing: clay.Sizing{Width: clay.SizingGrow(0, 400)}, ChildAlignment: core.YCENTER, ChildGap: core.S2, Padding: core.PA1},
								BackgroundColor: clay.Color{R: 60, G: 60, B: 60, A: 255},
								CornerRadius:    core.RA1,
							}, func() {
								clay.TEXT(key, clay.TextElementConfig{TextColor: core.Yellow, FontID: core.InterBold})
								clay.TEXT("=", clay.TextElementConfig{TextColor: core.Gray})
								clay.TEXT(v, clay.TextElementConfig{TextColor: core.White})
								clay.CLAY(clay.AUTO_ID, clay.EL{Layout: clay.LAY{Sizing: core.GROWH}}) // Spacer
								core.UIButton(clay.ID("DeleteVar-"+key), core.UIButtonConfig{
									El: clay.EL{Layout: clay.LAY{Padding: core.PA1}, BackgroundColor: core.Red, CornerRadius: core.RA1},
									OnClick: func(_ clay.ElementID, _ clay.PointerData, _ any) {
										core.PushHistory()
										CurrentGraph.VarMutex.Lock()
										delete(CurrentGraph.Variables, key)
										CurrentGraph.VarMutex.Unlock()
									},
								}, func() {
									clay.TEXT("Del", clay.TextElementConfig{TextColor: core.White, FontSize: 12})
								})
							})
						}
					})
					CurrentGraph.VarMutex.RUnlock()

					// Add New
					clay.CLAY(clay.AUTO_ID, clay.EL{Layout: clay.LAY{LayoutDirection: clay.TopToBottom, Sizing: clay.Sizing{Width: clay.SizingGrow(0, 400)}, ChildGap: core.S1, Padding: core.PA1}, Border: clay.BorderElementConfig{Width: core.BA, Color: core.Gray}, CornerRadius: core.RA1}, func() {
						clay.TEXT("Add New Variable", clay.TextElementConfig{TextColor: core.Gray, FontSize: 12})

						clay.CLAY(clay.AUTO_ID, clay.EL{Layout: clay.LAY{LayoutDirection: clay.LeftToRight, ChildGap: core.S2, Sizing: core.GROWH}}, func() {
							// Key Input
							clay.CLAY(clay.AUTO_ID, clay.EL{Layout: clay.LAY{Sizing: clay.Sizing{Width: clay.SizingGrow(1, 0)}, ChildGap: core.S1}}, func() {
								clay.TEXT("Key", clay.TextElementConfig{TextColor: core.Gray, FontSize: 10})
								core.UITextBox(clay.ID("NewVarKeyInput"), &NewVarKey, core.UITextBoxConfig{
									El: clay.EL{Layout: clay.LAY{Sizing: core.GROWH, Padding: core.PA1}, BackgroundColor: clay.Color{R: 40, G: 40, B: 40, A: 255}, CornerRadius: core.RA1},
								})
							})
							// Value Input
							clay.CLAY(clay.AUTO_ID, clay.EL{Layout: clay.LAY{Sizing: clay.Sizing{Width: clay.SizingGrow(1, 0)}, ChildGap: core.S1}}, func() {
								clay.TEXT("Value", clay.TextElementConfig{TextColor: core.Gray, FontSize: 10})
								core.UITextBox(clay.ID("NewVarValueInput"), &NewVarValue, core.UITextBoxConfig{
									El: clay.EL{Layout: clay.LAY{Sizing: core.GROWH, Padding: core.PA1}, BackgroundColor: clay.Color{R: 40, G: 40, B: 40, A: 255}, CornerRadius: core.RA1},
								})
							})
						})

						core.UIButton(clay.ID("AddVariable"), core.UIButtonConfig{
							El: clay.EL{Layout: clay.LAY{Padding: core.PA2, Sizing: clay.Sizing{Width: clay.SizingGrow(0, 400)}, ChildAlignment: core.ALLCENTER}, BackgroundColor: core.Green, CornerRadius: core.RA1},
							OnClick: func(_ clay.ElementID, _ clay.PointerData, _ any) {
								if NewVarKey != "" {
									core.PushHistory()
									CurrentGraph.VarMutex.Lock()
									CurrentGraph.Variables[NewVarKey] = NewVarValue
									CurrentGraph.VarMutex.Unlock()
									NewVarKey = ""
									NewVarValue = ""
								}
							},
						}, func() {
							clay.TEXT("Add Variable", clay.TextElementConfig{TextColor: core.White, FontID: core.InterBold})
						})
					})
				})
			})
		}

		if topoErr != nil {
			core.WithZIndex(core.Z_CYCLE_WARNING, func() {
				clay.CLAY(clay.AUTO_ID, clay.EL{
					Layout:          clay.LAY{Padding: clay.PaddingAll(8)},
					BackgroundColor: core.Red,
					CornerRadius:    core.RA1,
					Floating: clay.FLOAT{
						AttachTo:           clay.AttachToRoot,
						AttachPoints:       clay.FloatingAttachPoints{Element: clay.AttachPointRightTop, Parent: clay.AttachPointRightTop},
						Offset:             clay.Vector2{X: -20, Y: 20},
						PointerCaptureMode: clay.PointercaptureModeCapture,
						ZIndex:             core.Z_CYCLE_WARNING,
					},
				}, func() {
					clay.TEXT("Cycle Detected! Cannot Run.", clay.TextElementConfig{TextColor: core.White, FontID: core.InterBold, FontSize: 16})
				})
			})
		} else {

			var (
				RunCtx    context.Context
				RunCancel context.CancelFunc
			)

			// ... (In RunButton)
			// Run Button
			core.WithZIndex(core.Z_CONTEXT_MENU, func() {
				// Manually implement button to ensure specific styling (Green bg) is preserved.
				// Use CLAY_LATE to allow checking Hovered() state during config creation.
				id := clay.ID("RunButton")
				isRunning := RunCtx != nil

				clay.CLAY_LATE(id, func() clay.EL {
					bgColor := core.NavyBlue
					if isRunning {
						bgColor = core.Red
					} else if clay.Hovered() {
						bgColor = core.NavyBlueHover
					}

					return clay.EL{
						Layout:          clay.LAY{Padding: clay.PaddingAll(8)},
						BackgroundColor: bgColor,
						CornerRadius:    core.RA1,
						Floating: clay.FLOAT{
							AttachTo:           clay.AttachToRoot,
							AttachPoints:       clay.FloatingAttachPoints{Element: clay.AttachPointRightTop, Parent: clay.AttachPointRightTop},
							Offset:             clay.Vector2{X: -20, Y: 20},
							PointerCaptureMode: clay.PointercaptureModeCapture,
							ZIndex:             core.Z_CONTEXT_MENU,
						},
					}
				}, func() {
					// Icon
					if isRunning {
						// Stop Icon (White Square)
						clay.CLAY(clay.AUTO_ID, clay.EL{
							Layout:          clay.LAY{Sizing: core.WH(12, 12)},
							BackgroundColor: core.White,
						})
					} else {
						// Play Icon
						clay.CLAY(clay.AUTO_ID, clay.EL{
							Layout: clay.LAY{Sizing: core.WH(16, 16)},
							Image:  clay.IMG{ImageData: core.ImgPlay},
						})
					}

					// Tooltip
					if clay.Hovered() {
						core.UITooltip(util.Tern(isRunning, "Stop Flow", "Run Flow"))
					}

					clay.OnHover(func(elementID clay.ElementID, pointerData clay.PointerData, _ any) {
						core.IsHoveringUI = true
						core.UIInput.RegisterPointerDown(elementID, pointerData, core.Z_CONTEXT_MENU)
						if core.UIInput.IsClick(elementID, pointerData) {
							if isRunning {
								if RunCancel != nil {
									RunCancel()
								}
							} else {
								var ctx context.Context
								ctx, RunCancel = context.WithCancel(context.Background())
								RunCtx = ctx
								err := RunGraph(ctx, CurrentGraph, func(err error) {
									// Reset state on UI thread?
									// Warning: onComplete is called from goroutine.
									// We are accessing global vars RunCtx/RunCancel.
									// This is racey if UI checks them?
									// UI reads them every frame.
									// We should use a lock or atomic?
									// Or just set them to nil. Go pointer writes are atomic-ish for nil?
									// Better: Enqueue a task to main thread?
									// For now, let's just write them. Raylib loop runs on main, this runs on background.
									// Strict correctness requires synchronization.
									// But for a simple boolean flag it might be "okay" for MVP.
									// Ideally we use a mutex. CurrentGraph has a mutex? No.
									// Let's rely on the fact that UI just checks != nil.
									RunCtx = nil
									RunCancel = nil
									if err != nil && err != context.Canceled {
										fmt.Printf("Run error: %v\n", err)
									}
								})
								if err != nil {
									fmt.Printf("Run error (immediate): %v\n", err)
									RunCtx = nil
									RunCancel = nil
								}
							}
						}
					}, nil)
				})
			})
		}

		// Check expansion state for New core.Node Menu
		textboxID := clay.ID("NewNodeName")
		isFocused := core.IsFocused(textboxID)

		shortcut := rl.IsKeyPressed(rl.KeySpace) && (rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl))

		if rl.IsKeyPressed(rl.KeyEscape) {
			core.UIFocus = nil
			isFocused = false
		}

		containerWidth := clay.SizingFit(0, 0)
		if isFocused || shortcut {
			containerWidth = clay.SizingFixed(600)
		}

		core.WithZIndex(core.Z_NODE_SELECTED, func() {
			clay.CLAY(clay.ID("NewNodeContainer"), clay.EL{
				Layout: clay.LAY{
					Sizing:  clay.Sizing{Width: containerWidth, Height: clay.SizingFit(0, 0)},
					Padding: core.PA3,
				},
				Floating: clay.FLOAT{
					AttachTo: clay.AttachToParent,
					AttachPoints: clay.FloatingAttachPoints{
						Element: clay.AttachPointLeftBottom,
						Parent:  clay.AttachPointLeftBottom,
					},
					ZIndex: core.Z_NODE_SELECTED,
				},
			}, func() {
				clay.CLAY_AUTO_ID(clay.EL{
					Layout: clay.LAY{
						ChildGap: core.S2,
						Sizing:   clay.Sizing{Width: core.GROWALL.Width, Height: clay.SizingFit(0, 0)},
					},
				}, func() {
					clay.OnHover(func(elementID clay.ElementID, pointerData clay.PointerData, userData any) {
						core.IsHoveringUI = true
					}, nil)

					// textboxID already defined above
					// shortcut already defined above

					// Before UI: defocus textbox
					if shortcut && isFocused { // Was core.IsFocused(textboxID)
						core.UIFocus = nil
					}

					core.UIButton(clay.ID("NewNode"), core.UIButtonConfig{
						El: clay.EL{
							Layout: clay.LAY{
								Sizing:         core.WH(36, 36),
								ChildAlignment: core.ALLCENTER,
							},
							Border: clay.B{Width: core.BA, Color: core.Gray},
						},
						OnClick: func(elementID clay.ElementID, pointerData clay.PointerData, userData any) {
							NewNodeName = ""
							id := textboxID
							core.UIFocus = &id
						},
					}, func() {
						clay.TEXT("+", clay.T{FontID: core.InterBold, FontSize: 36, TextColor: core.White})
					})

					if isFocused || shortcut {
						addNodeFromMatch := func(nt NodeType) {
							_, count := parseNodeSearch(NewNodeName)

							prevPos := V2{}

							for i := 0; i < count; i++ {
								newNode := nt.Create()

								var pos V2
								if i == 0 {
									centerWorld := Camera.ScreenToWorld(rl.Vector2{X: float32(rl.GetScreenWidth()) / 2, Y: float32(rl.GetScreenHeight()) / 2})
									pos = SnapToGrid(centerWorld)
									// Collision avoidance with bounding box check
									testRect := rl.Rectangle{X: pos.X, Y: pos.Y, Width: NodeMinWidth, Height: 150} // 150 is estimated height
									for j := 0; j < 50; j++ {
										overlap := false
										for _, n := range CurrentGraph.Nodes {
											// Use the actual drag rect if available (it might be 0 if not rendered yet, but we have n.Pos)
											// Or fallback to a default size check
											otherRect := n.DragRect
											if otherRect.Width == 0 {
												otherRect = rl.Rectangle{X: n.Pos.X, Y: n.Pos.Y, Width: NodeMinWidth, Height: 150}
											}
											if rl.CheckCollisionRecs(testRect, otherRect) {
												overlap = true
												break
											}
										}
										if !overlap {
											break
										}
										pos.X += GridSize
										pos.Y += GridSize
										testRect.X = pos.X
										testRect.Y = pos.Y
									}
								} else {
									// Place below previous (vertical alignment)
									// We don't know the exact height until first frame, but we can guess or use a standard height
									pos = prevPos
									pos.Y += 110 // Vertical spacing
								}

								newNode.Pos = pos
								prevPos = pos

								// Special handling for batch SaveFile nodes
								if count > 1 {
									if saveAction, ok := newNode.Action.(*nodes.SaveFileAction); ok {
										// Split extension
										ext := filepath.Ext(saveAction.Path)
										base := strings.TrimSuffix(saveAction.Path, ext)
										saveAction.Path = fmt.Sprintf("%s_%d%s", base, i+1, ext)
									}
								}

								CurrentGraph.AddNode(newNode)
								selectedNodeID = newNode.ID
							}
						}

						core.UITextBox(textboxID, &NewNodeName, core.UITextBoxConfig{
							El: clay.ElementDeclaration{
								Layout: clay.LAY{
									Sizing: core.GROWALL,
								},
							},
							OnSubmit: func(val string) {
								matches := SearchNodeTypes(val)
								if len(matches) > 0 {
									addNodeFromMatch(matches[0])
								}
								core.UIFocus = nil
							},
						}, func() {
							core.WithZIndex(core.Z_NEW_NODE_POPUP, func() {
								clay.CLAY(clay.ID("NewNodeMatches"), clay.EL{
									Layout: clay.LAY{
										LayoutDirection: clay.TopToBottom,
										Sizing:          clay.Sizing{Width: clay.SizingPercent(1.0), Height: clay.SizingAxis{Type: clay.SizingTypeFit, MinMax: clay.SizingMinMax{Min: 400, Max: float32(rl.GetScreenHeight()) * 0.5}}},
										Padding:         core.PA2,
										ChildGap:        core.S2,
									},
									BackgroundColor: core.Night,
									Border:          clay.B{Width: core.BA, Color: core.Gray},
									CornerRadius:    core.RA(4),
									Floating: clay.FLOAT{
										AttachTo: clay.AttachToElementWithID,
										ParentID: textboxID.ID,
										AttachPoints: clay.FloatingAttachPoints{
											Parent:  clay.AttachPointLeftTop,
											Element: clay.AttachPointLeftBottom,
										},
										ZIndex:             core.Z_NEW_NODE_POPUP,
										PointerCaptureMode: clay.PointercaptureModeCapture,
									},
								}, func() {
									core.WithZIndex(core.Z_NEW_NODE_POPUP, func() {
										clay.OnHover(func(elementID clay.ElementID, pointerData clay.PointerData, userData any) {
											core.IsHoveringUI = true
										}, nil)

										// Scrollable Content Area
										clay.CLAY(clay.ID("NewNodeMatchesContent"), clay.EL{
											Layout: clay.LAY{
												LayoutDirection: clay.TopToBottom,
												Sizing:          clay.Sizing{Width: core.GROWALL.Width, Height: clay.SizingAxis{Type: clay.SizingTypeFit}},
											},
											Clip: clay.ClipElementConfig{
												Vertical:    true,
												Horizontal:  false,
												ChildOffset: clay.GetScrollOffset(),
											},
										}, func() {
											matches := SearchNodeTypes(NewNodeName)
											showCategories := NewNodeName == "" || NewNodeName == "?" || NewNodeName == "*"

											if showCategories {
												if SelectedNodeCategory == "" && len(nodeTypes) > 0 {
													SelectedNodeCategory = nodeTypes[0].Category
												}

												clay.CLAY(clay.AUTO_ID, clay.EL{
													Layout: clay.LAY{
														LayoutDirection: clay.LeftToRight,
														Sizing:          core.GROWH,
													},
												}, func() {
													// Left Column: Categories
													clay.CLAY(clay.ID("CategoryList"), clay.EL{
														Layout: clay.LAY{
															LayoutDirection: clay.TopToBottom,
															Sizing:          clay.Sizing{Width: clay.SizingFixed(140)},
															ChildGap:        core.S1,
															Padding:         core.PA1,
														},
														BackgroundColor: core.Charcoal,
														CornerRadius:    core.RA(4),
														// Border:          clay.B{Width: clay.BW{Right: 1}, Color: core.Gray},
													}, func() {
														currentCat := ""
														for _, nt := range nodeTypes {
															if nt.Category != currentCat {
																cat := nt.Category
																currentCat = cat
																thisCat := cat

																core.UIButton(clay.ID("CatBtn"+thisCat), core.UIButtonConfig{
																	El: clay.EL{
																		Layout:          clay.LAY{Padding: core.PVH(core.S1, core.S2), Sizing: core.GROWH},
																		BackgroundColor: util.Tern(thisCat == SelectedNodeCategory, core.Blue, util.Tern(clay.Hovered(), core.HoverWhite, clay.Color{})),
																		CornerRadius:    core.RA(4),
																	},
																	OnHover: func(_ clay.ElementID, _ clay.PointerData, _ any) {
																		SelectedNodeCategory = thisCat
																	},
																	OnClick: func(_ clay.ElementID, _ clay.PointerData, _ any) {
																		SelectedNodeCategory = thisCat
																	},
																}, func() {
																	clay.TEXT(thisCat, clay.T{TextColor: core.White, FontID: core.InterBold, FontSize: 14})
																})
															}

														}
													})

													// Right Column: Nodes
													clay.CLAY(clay.ID("CategoryNodes"), clay.EL{
														Layout: clay.LAY{
															LayoutDirection: clay.TopToBottom,
															Sizing:          core.GROWH,
															ChildGap:        core.S1,
															Padding:         core.PA1,
														},
													}, func() {
														for i, nt := range matches {
															if nt.Category == SelectedNodeCategory {
																core.UIButton(clay.IDI("MatchButton_Cat", i), core.UIButtonConfig{
																	El: clay.EL{
																		Layout:          clay.LAY{Padding: core.PVH(core.S2, core.S3), Sizing: core.GROWH, ChildGap: core.S2, ChildAlignment: clay.ChildAlignment{Y: clay.AlignYCenter}},
																		BackgroundColor: util.Tern(clay.Hovered(), core.HoverWhite, clay.Color{}),
																		CornerRadius:    core.RA(4),
																	},
																	OnClick: func(_ clay.ElementID, _ clay.PointerData, userData any) {
																		addNodeFromMatch(matches[userData.(int)])
																		core.UIFocus = nil
																	},
																	OnClickUserData: i,
																}, func() {
																	clay.TEXT(nt.Name, clay.T{TextColor: core.White, FontSize: 14})
																	clay.CLAY(clay.AUTO_ID, clay.EL{Layout: clay.LAY{Sizing: core.GROWH}}) // Spacer to push shortcut right
																	if nt.ShortcutKey != 0 {
																		keyName := ""
																		if nt.ShortcutKey >= 32 && nt.ShortcutKey <= 126 {
																			keyName = string(rune(nt.ShortcutKey))
																		} else {
																			keyName = fmt.Sprintf("K%d", nt.ShortcutKey)
																		}
																		modStr := ""
																		for _, mod := range nt.ShortcutMods {
																			switch mod {
																			case rl.KeyLeftControl, rl.KeyRightControl:
																				modStr += "Ctrl+"
																			case rl.KeyLeftAlt, rl.KeyRightAlt:
																				modStr += "Alt+"
																			case rl.KeyLeftShift, rl.KeyRightShift:
																				modStr += "Shift+"
																			}
																		}
																		clay.TEXT(modStr+keyName, clay.T{TextColor: core.LightGray, FontSize: 10})
																	}
																})
															}
														}
													})
												})
											} else {
												// Original Flat List (Search Results)
												for i := 0; i < len(matches); i++ {
													nt := matches[i]
													core.UIButton(clay.IDI("MatchButton_Flat", i), core.UIButtonConfig{
														El: clay.EL{
															Layout:          clay.LAY{Padding: core.PVH(core.S2, core.S3), Sizing: core.GROWH, ChildGap: core.S2, ChildAlignment: clay.ChildAlignment{Y: clay.AlignYCenter}},
															BackgroundColor: util.Tern(clay.Hovered(), core.HoverWhite, clay.Color{}),
															CornerRadius:    core.RA(6),
														},
														OnClick: func(_ clay.ElementID, _ clay.PointerData, userData any) {
															addNodeFromMatch(matches[userData.(int)])
															core.UIFocus = nil
														},
														OnClickUserData: i,
													}, func() {
														clay.TEXT(nt.Name, clay.T{TextColor: core.White, FontSize: 14})
														clay.CLAY(clay.AUTO_ID, clay.EL{Layout: clay.LAY{Sizing: core.GROWH}}) // Spacer
														_, count := parseNodeSearch(NewNodeName)
														if count > 1 {
															clay.TEXT(fmt.Sprintf(" (x%d)", count), clay.T{TextColor: core.Yellow, FontSize: 14})
														}
														if nt.ShortcutKey != 0 {
															keyName := ""
															if nt.ShortcutKey >= 32 && nt.ShortcutKey <= 126 {
																keyName = string(rune(nt.ShortcutKey))
															} else {
																keyName = fmt.Sprintf("K%d", nt.ShortcutKey)
															}
															modStr := ""
															for _, mod := range nt.ShortcutMods {
																switch mod {
																case rl.KeyLeftControl, rl.KeyRightControl:
																	modStr += "Ctrl+"
																case rl.KeyLeftAlt, rl.KeyRightAlt:
																	modStr += "Alt+"
																case rl.KeyLeftShift, rl.KeyRightShift:
																	modStr += "Shift+"
																}
															}
															clay.TEXT("  "+modStr+keyName, clay.T{TextColor: core.Gray, FontSize: 10})
														}
													})
												}
											}
										})

										// Footer with Settings
										clay.CLAY(clay.ID("NewNodeMatchesFooter"), clay.EL{
											Layout: clay.LAY{LayoutDirection: clay.LeftToRight, Sizing: clay.Sizing{Width: core.GROWALL.Width, Height: clay.SizingFixed(32)}, ChildAlignment: core.YCENTER, Padding: core.PVH(0, core.S2), ChildGap: core.S2},
										}, func() {
											clay.CLAY(clay.AUTO_ID, clay.EL{Layout: clay.LAY{Sizing: core.GROWH}}) // Spacer
											core.UIButton(clay.ID("SaveFlow"), core.UIButtonConfig{
												El: clay.EL{
													Layout: clay.LAY{Padding: core.PVH(core.S1, core.S2)},
												},
												OnHover: func(elementID clay.ElementID, pointerData clay.PointerData, userData any) {
													core.UITooltip("Save Flow (Ctrl+S)")
												},
												OnClick: func(elementID clay.ElementID, pointerData clay.PointerData, userData any) {
													core.PushHistory()
													_ = core.SaveGraph("saved.flow", CurrentGraph)
													core.UIFocus = nil // Close menu
												},
											}, func() {
												clay.TEXT("Save Flow", clay.T{TextColor: core.LightGray, FontSize: 12})
											})
											core.UIButton(clay.ID("LoadFlow"), core.UIButtonConfig{
												El: clay.EL{
													Layout: clay.LAY{Padding: core.PVH(core.S1, core.S2)},
												},
												OnHover: func(elementID clay.ElementID, pointerData clay.PointerData, userData any) {
													core.UITooltip("Load Flow (Ctrl+L)")
												},
												OnClick: func(elementID clay.ElementID, pointerData clay.PointerData, userData any) {
													ShowLoadConfirmation = true
													core.UIFocus = nil // Close menu
												},
											}, func() {
												clay.TEXT("Load Flow", clay.T{TextColor: core.LightGray, FontSize: 12})
											})
											core.UIButton(clay.ID("OpenSettings"), core.UIButtonConfig{
												El: clay.EL{
													Layout: clay.LAY{Padding: core.PVH(core.S1, core.S2)},
												},
												OnHover: func(elementID clay.ElementID, pointerData clay.PointerData, userData any) {
													core.UITooltip("Toggle Settings")
												},
												OnClick: func(elementID clay.ElementID, pointerData clay.PointerData, userData any) {
													ShowVariables = !ShowVariables
													core.UIFocus = nil // Close menu
												},
											}, func() {
												clay.TEXT("Settings", clay.T{TextColor: core.LightGray, FontSize: 12})
											})
										})
									})
								})
							})
						})
					}

					// After UI: focus textbox
					if shortcut && !core.IsFocused(textboxID) {
						core.UIFocus = &textboxID
						NewNodeName = ""
					}
				})
			})
		})

		// Spacer to push Output to the right
		clay.CLAY(clay.AUTO_ID, clay.EL{Layout: clay.LAY{Sizing: core.GROWH}})

		clay.CLAY_LATE(clay.ID("OverlayOutput"), func() clay.EL {
			width := OutputWindowWidth
			if IsOutputCollapsed {
				width = 40
			}
			return clay.EL{
				Layout: clay.LAY{
					LayoutDirection: clay.TopToBottom,
					Sizing:          clay.Sizing{Width: clay.SizingFixed(width), Height: clay.SizingGrow(1, 0)},
					Padding:         core.PA2,
				},
				BackgroundColor: core.Charcoal,
				Border:          clay.B{Width: core.BL, Color: core.Gray},
				Clip: clay.ClipElementConfig{
					Vertical:    true,
					Horizontal:  true,
					ChildOffset: clay.GetScrollOffset(),
				},
			}
		}, func() {
			clay.OnHover(func(elementID clay.ElementID, pointerData clay.PointerData, userData any) {
				core.IsHoveringUI = true
			}, nil)

			// Header / Toggle
			clay.CLAY_AUTO_ID(clay.EL{
				Layout: clay.LAY{
					LayoutDirection: clay.LeftToRight,
					ChildAlignment:  core.XRIGHT,
					Sizing:          core.GROWH,
					Padding:         clay.Padding{Bottom: core.S2},
				},
			}, func() {
				core.UIButton(clay.AUTO_ID, core.UIButtonConfig{
					OnClick: func(elementID clay.ElementID, pointerData clay.PointerData, userData any) {
						IsOutputCollapsed = !IsOutputCollapsed
					},
				}, func() {
					img := core.ImgToggleDown
					if IsOutputCollapsed {
						img = core.ImgToggleRight
					}
					core.UIImage(clay.AUTO_ID, img, clay.EL{})
				})
			})

			if !IsOutputCollapsed {
				if selectedNode, ok := GetSelectedNode(); ok {
					if result, ok := selectedNode.GetResult(); ok {
						if result.Err == nil {
							for outputIndex, output := range result.Outputs {
								port := selectedNode.OutputPorts[outputIndex]
								if err := core.Typecheck(*output.Type, port.Type); err != nil {
									// panic(err)
									clay.TEXT(fmt.Sprintf("Type Error: %v", err), clay.TextElementConfig{TextColor: core.Red})
									continue
								}

								outputState := selectedNode.GetOutputState(port.Name)

								clay.CLAY_AUTO_ID(clay.EL{
									Layout: clay.LAY{ChildGap: core.S1, ChildAlignment: core.YCENTER},
								}, func() {
									core.UIButton(clay.AUTO_ID, core.UIButtonConfig{
										OnClick: func(elementID clay.ElementID, pointerData clay.PointerData, userData any) {
											outputState.Collapsed = !outputState.Collapsed
										},
									}, func() {
										core.UIImage(clay.AUTO_ID, util.Tern(outputState.Collapsed, core.ImgToggleRight, core.ImgToggleDown), clay.EL{})
									})
									clay.TEXT(port.Name, clay.TextElementConfig{FontID: core.InterSemibold, TextColor: core.White})
								})
								if !outputState.Collapsed {
									clay.CLAY_AUTO_ID(clay.EL{
										Layout: clay.LAY{ChildGap: core.S1},
									}, func() {
										clay.CLAY_AUTO_ID(clay.EL{
											Layout: clay.LAY{
												Sizing:         clay.Sizing{Width: core.PX(float32(core.ImgToggleDown.Width)), Height: core.GROWV.Height},
												ChildAlignment: core.XCENTER,
											},
										}, func() {
											clay.CLAY_AUTO_ID(clay.EL{
												Layout: clay.LAY{
													Sizing: clay.Sizing{Width: core.PX(1), Height: core.GROWV.Height},
												},
												Border: clay.B{Color: core.Gray, Width: core.BR},
											})
										})
										core.UIFlowValue(clay.IDIWith(selectedNode.ID, "OutputVal", outputIndex), output)
									})
								}
							}
						} else {
							clay.TEXT(result.Err.Error(), clay.TextElementConfig{TextColor: core.Red})
						}
					}
				}
			}
		})

		// Box Selection & Panning
		if overlay, ok := clay.GetElementData(clay.ID("OverlayRoot")); ok {
			isShift := rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift)

			if !core.IsHoveringUI && !core.IsHoveringPanel {
				if isShift {
					// Snapshot selection
					initial := make(map[int]struct{})
					for k, v := range SelectedNodes {
						initial[k] = v
					}
					core.Drag.TryStartDrag(BoxSelectDrag{InitialSelection: initial}, rl.Rectangle(overlay.BoundingBox), V2{})
				} else {
					if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
						clear(SelectedNodes)
						selectedNodeID = 0
						core.UIFocus = nil
					}
					if core.Drag.TryStartDrag(PanDragKey, rl.Rectangle(overlay.BoundingBox), V2{}) {
						LastPanMousePosition = rl.GetMousePosition()
					}
				}
			}

			if boxSelecting, _, canceled := core.Drag.State(BoxSelectDragKey); boxSelecting {
				if bs, ok := core.Drag.Thing.(BoxSelectDrag); ok {
					if canceled {
						// Restore
						clear(SelectedNodes)
						for k, v := range bs.InitialSelection {
							SelectedNodes[k] = v
						}
					} else {
						// Update selection
						start := core.Drag.MouseStart
						end := rl.GetMousePosition()

						startWorld := Camera.ScreenToWorld(start)
						endWorld := Camera.ScreenToWorld(end)

						x := min(startWorld.X, endWorld.X)
						y := min(startWorld.Y, endWorld.Y)
						w := max(startWorld.X, endWorld.X) - x
						h := max(startWorld.Y, endWorld.Y) - y
						box := rl.Rectangle{X: x, Y: y, Width: w, Height: h}

						clear(SelectedNodes)
						for k, v := range bs.InitialSelection {
							SelectedNodes[k] = v
						}

						for _, n := range CurrentGraph.Nodes {
							if rl.CheckCollisionRecs(box, n.DragRect) {
								SelectedNodes[n.ID] = struct{}{}
								selectedNodeID = n.ID
							}
						}
					}
				}
			}

			if panning, _, _ := core.Drag.State(PanDragKey); panning {
				mousePos := rl.GetMousePosition()
				delta := rl.Vector2Subtract(mousePos, LastPanMousePosition)
				Camera.Pan(delta)
				LastPanMousePosition = mousePos
			}

			// Render Connection Error
			if time.Since(ConnectionErrorTime) < 2*time.Second {
				clay.CLAY_AUTO_ID(clay.EL{
					Floating: clay.FloatingElementConfig{
						AttachTo: clay.AttachToRoot,
						Offset:   clay.V2(rl.GetMousePosition()).Plus(clay.V2{X: 16, Y: 16}),
						ZIndex:   core.ZTOP,
					},
					Layout:          clay.LAY{Padding: core.PA1},
					BackgroundColor: core.Red,
					Border:          clay.BorderElementConfig{Color: core.White, Width: core.BA},
					CornerRadius:    core.RA1,
				}, func() {
					clay.TEXT(ConnectionError, clay.TextElementConfig{TextColor: core.White, FontID: core.InterBold})
				})
			}

			// Render Context Menu
			if ContextMenu != nil {
				// Close if clicking outside (handled in processInput generally, but let's be safe?)
				// Actually clay.OnHover only works if we cover screen?
				// processInput handles the "click outside" via global mouse check.
				UIContextMenu()
			}
		}
	})

	rl.SetMouseCursor(core.UICursor)
	core.UICursor = rl.MouseCursorDefault
}

func afterLayout() {
	for _, node := range CurrentGraph.Nodes {
		node.UpdateLayoutInfo()
	}
}

func renderWorldOverlays() {
	// Render wires
	// Render wires
	for _, wire := range CurrentGraph.Wires {
		// Guard against uninitialized layout info (e.g. immediately after load)
		if wire.StartPort >= len(wire.StartNode.OutputPortPositions) || wire.EndPort >= len(wire.EndNode.InputPortPositions) {
			continue
		}

		res, ok := wire.StartNode.GetResult()
		isErr := ok && res.Err != nil
		color := util.Tern(isErr, core.Red, core.LightGray)
		rl.DrawLineBezier(
			rl.Vector2(wire.StartNode.OutputPortPositions[wire.StartPort]),
			rl.Vector2(wire.EndNode.InputPortPositions[wire.EndPort]),
			1,
			color.RGBA(),
		)
	}
	if draggingNewWire, _, _ := core.Drag.State(NewWireDragKey); draggingNewWire {
		// Calculate end position in screen space
		mousePos := rl.GetMousePosition()

		rl.DrawLineBezier(
			rl.Vector2(NewWireSourceNode.OutputPortPositions[NewWireSourcePort]),
			mousePos,
			1,
			core.LightGray.RGBA(),
		)
	}

	for _, node := range CurrentGraph.Nodes {
		for _, portPos := range append(node.InputPortPositions, node.OutputPortPositions...) {
			rl.DrawCircle(int32(portPos.X), int32(portPos.Y), 4, core.White.RGBA())
		}
	}
}

func renderScreenOverlays() {
	if draggingBox, _, _ := core.Drag.State(BoxSelectDragKey); draggingBox {
		start := core.Drag.MouseStart
		end := rl.GetMousePosition()

		x := min(start.X, end.X)
		y := min(start.Y, end.Y)
		w := max(start.X, end.X) - x
		h := max(start.Y, end.Y) - y
		rect := rl.Rectangle{X: x, Y: y, Width: w, Height: h}

		c := rl.Color{R: uint8(core.Blue.R), G: uint8(core.Blue.G), B: uint8(core.Blue.B), A: uint8(core.Blue.A)}
		rl.DrawRectangleRec(rect, rl.Fade(c, 0.2))
		rl.DrawRectangleLinesEx(rect, 1, c)
	}
}

const GroupDragKey = "GROUP_DRAG"

type GroupDrag struct {
	Group *core.Group
	Nodes []*core.Node
}

func (g GroupDrag) DragKey() string { return GroupDragKey }

func UIGroup(group *core.Group) {
	clay.CLAY(clay.IDI("core.Group", group.ID), clay.EL{
		Floating: clay.FloatingElementConfig{
			AttachTo: clay.AttachToParent,
			Offset:   clay.Vector2(Camera.WorldToScreen(group.XY())),
			ClipTo:   clay.ClipToAttachedParent,
		},
		Layout: clay.LAY{
			LayoutDirection: clay.TopToBottom,
			Sizing:          clay.Sizing{Width: clay.SizingFixed(group.Rect.Width * Camera.Zoom), Height: clay.SizingFixed(group.Rect.Height * Camera.Zoom)},
		},
		BackgroundColor: clay.Color{R: group.Color.R, G: group.Color.G, B: group.Color.B, A: 50},
		Border:          clay.B{Color: group.Color, Width: core.BA},
		CornerRadius:    core.RA2,
	}, func() {
		clay.OnHover(func(elementID clay.ElementID, pointerData clay.PointerData, userData any) {
			// Don't set core.IsHoveringPanel so we can still pan if we click the background of a group
			// But we do want to catch clicks to select/drag
			core.UIInput.RegisterPointerDown(elementID, pointerData, 0)
		}, nil)

		// Header
		clay.CLAY_AUTO_ID(clay.EL{
			Layout: clay.LAY{
				Sizing:         core.GROWH,
				Padding:        core.PD(1, 0, 0, 0, core.PVH(core.S1, core.S2)),
				ChildAlignment: clay.ChildAlignment{Y: clay.AlignYCenter},
			},
			BackgroundColor: clay.Color{R: group.Color.R, G: group.Color.G, B: group.Color.B, A: 200},
		}, func() {
			clay.OnHover(func(elementID clay.ElementID, pointerData clay.PointerData, _ any) {
				core.UIInput.RegisterPointerDown(elementID, pointerData, 0)

				if core.Drag.TryStartDrag(GroupDrag{Group: group}, rl.Rectangle{}, V2{}) {
					// Find nodes inside
					var nodes []*core.Node
					for _, n := range CurrentGraph.Nodes {
						// Simple center point check
						center := rl.Vector2Add(n.Pos, rl.Vector2{X: NodeMinWidth / 2, Y: 50}) // Approx center
						if rl.CheckCollisionPointRec(center, group.Rect) {
							nodes = append(nodes, n)
						}
					}
					// Update the drag thing with found nodes
					core.Drag.Thing = GroupDrag{Group: group, Nodes: nodes}
				}
			}, nil)

			clay.TEXT(group.Title, clay.TextElementConfig{FontID: core.InterSemibold, FontSize: core.F3, TextColor: core.White})
		})
	})

	if dragging, _, _ := core.Drag.State(GroupDragKey); dragging {
		if gd, ok := core.Drag.Thing.(GroupDrag); ok && gd.Group.ID == group.ID {
			targetPos := core.Drag.NewObjPosition()
			if !rl.IsKeyDown(rl.KeyLeftShift) && !rl.IsKeyDown(rl.KeyRightShift) {
				targetPos = SnapToGrid(targetPos)
			}

			delta := rl.Vector2Subtract(targetPos, group.XY())
			group.Rect.X = targetPos.X
			group.Rect.Y = targetPos.Y

			for _, n := range gd.Nodes {
				n.Pos = rl.Vector2Add(n.Pos, delta)
			}
		}
	}
}

func UINode(node *core.Node, disabled bool) {
	if core.DebugMode && debugFrameCount%60 == 0 {
		fmt.Printf("DEBUG: Rendering Node %d (%s) ClayID=%v\n", node.ID, node.Name, node.ClayID())
	}
	// core.Node header font size
	headerFontSize := uint16(core.F3)

	border := clay.B{
		Color: core.Gray,
		Width: core.BA,
	}
	res, _ := node.GetResult()
	if res.Err != nil {
		border = clay.B{
			Color: core.Red,
			Width: core.BA2,
		}
	} else if IsNodeSelected(node.ID) {
		border = clay.B{
			Color: core.Blue,
			Width: core.BA2,
		}
	}

	zIndex := int16(0)
	if IsNodeSelected(node.ID) {
		zIndex = 10
	}

	core.WithZIndex(zIndex, func() {
		clay.CLAY(node.ClayID(), clay.EL{
			Floating: clay.FloatingElementConfig{
				AttachTo: clay.AttachToParent,
				Offset:   clay.Vector2(Camera.WorldToScreen(node.Pos)),
				ClipTo:   clay.ClipToAttachedParent,
				ZIndex:   zIndex,
			},

			Layout: clay.LAY{
				LayoutDirection: clay.TopToBottom,
				Sizing:          clay.Sizing{Width: clay.SizingFit(NodeMinWidth, 0)},
			},
			BackgroundColor: core.DarkGray,
			Border:          border,
			CornerRadius:    core.RA2,
		}, func() {
			clay.OnHover(func(elementID clay.ElementID, pointerData clay.PointerData, userData any) {
				core.IsHoveringPanel = true
				core.UIInput.RegisterPointerDown(elementID, pointerData, zIndex)
				if core.UIInput.IsClick(elementID, pointerData) && !core.Drag.WasDragging {
					core.UIFocus = nil
					multi := rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift) || rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl)
					SelectNode(node.ID, multi)
				}
			}, nil)

			clay.CLAY(clay.IDI("NodeHeader", node.ID), clay.EL{ // core.Node header
				Layout: clay.LAY{
					Sizing:         core.GROWH,
					Padding:        core.PD(1, 0, 0, 0, core.PVH(core.S1, core.S2)),
					ChildAlignment: clay.ChildAlignment{Y: clay.AlignYCenter},
				},
				BackgroundColor: core.Charcoal,
			}, func() {
				clay.OnHover(func(elementID clay.ElementID, pointerData clay.PointerData, _ any) {
					core.IsHoveringPanel = true
					core.UIInput.RegisterPointerDown(elementID, pointerData, zIndex)
					if core.UIInput.IsClick(elementID, pointerData) && !core.Drag.WasDragging {
						core.UIFocus = nil
						multi := rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift) || rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl)
						SelectNode(node.ID, multi)
					}
				}, nil)

				clay.TEXT(node.Name, clay.TextElementConfig{FontID: core.InterSemibold, FontSize: headerFontSize, TextColor: core.White})
				core.UISpacer(node.DragHandleClayID(), core.GROWALL)
				// ... (rest of header using scaled vars if needed)
				// For now, let's just make sure the main container and header scale.
				// Buttons might need scaling too.

				// ...
				if node.Running {
					clay.TEXT("Running...", clay.TextElementConfig{TextColor: core.White})
				}

				playButtonDisabled := !node.Valid || node.Running || disabled

				core.UIButton(clay.IDI("NodePin", node.ID), // Pin button
					core.UIButtonConfig{
						El:     clay.EL{Layout: clay.LAY{Padding: core.PA1}},
						ZIndex: zIndex + 100,
						OnHover: func(elementID clay.ElementID, pointerData clay.PointerData, userData any) {
							core.UITooltip("Pin command (prevent automatic re-runs)")
						},
						OnClick: func(elementID clay.ElementID, pointerData clay.PointerData, userData any) {
							node.Pinned = !node.Pinned
						},
					},
					func() {
						core.UIImage(clay.IDI("NodePinIcon", node.ID), util.Tern(node.Pinned, core.ImgPushpin, core.ImgPushpinOutline), clay.EL{
							BackgroundColor: core.Red,
						})
					},
				)
				core.UIButton(clay.IDI("NodeRetry", node.ID), // Retry / play-all button
					core.UIButtonConfig{
						El:       clay.EL{Layout: clay.LAY{Padding: core.PA1}},
						Disabled: playButtonDisabled,
						ZIndex:   zIndex + 100,
						OnHover: func(elementID clay.ElementID, pointerData clay.PointerData, userData any) {
							core.UITooltip("Run command and inputs")
						},
						OnClick: func(elementID clay.ElementID, pointerData clay.PointerData, userData any) {
							node.Run(context.Background(), true)
						},
					},
					func() {
						core.UIImage(clay.IDI("NodeRetryIcon", node.ID), core.ImgRetry, clay.EL{
							BackgroundColor: util.Tern(playButtonDisabled, core.LightGray, core.Blue),
						})
					},
				)
				core.UIButton(clay.IDI("NodePlay", node.ID), // Play button
					core.UIButtonConfig{
						El:       clay.EL{Layout: clay.LAY{Padding: core.PA1}},
						Disabled: playButtonDisabled,
						ZIndex:   zIndex + 100,
						OnHover: func(elementID clay.ElementID, pointerData clay.PointerData, userData any) {
							core.UITooltip("Run command")
						},
						OnClick: func(elementID clay.ElementID, pointerData clay.PointerData, userData any) {
							node.Run(context.Background(), false)
						},
					},
					func() {
						core.UIImage(clay.IDI("NodePlayIcon", node.ID), core.ImgPlay, clay.EL{
							BackgroundColor: util.Tern(playButtonDisabled, core.LightGray, core.PlayButtonGreen),
						})
					},
				)
				core.UIButton(clay.IDI("NodeDelete", node.ID), // Delete button
					core.UIButtonConfig{
						El: clay.EL{Layout: clay.LAY{Padding: core.PA1}},
						// Ensure delete button is always clickable above other node elements
						ZIndex: zIndex + 100,
						OnHover: func(elementID clay.ElementID, pointerData clay.PointerData, userData any) {
							core.UITooltip("Delete core.Node")
						},
						OnClick: func(elementID clay.ElementID, pointerData clay.PointerData, userData any) {
							DeleteNode(node.ID)
							delete(SelectedNodes, node.ID)
							if selectedNodeID == node.ID {
								selectedNodeID = 0
							}
						},
					},
					func() {
						clay.TEXT("X", clay.TextElementConfig{TextColor: util.Tern(clay.Hovered(), core.Red, core.LightGray), FontID: core.InterBold})
					},
				)
			})
			clay.CLAY(clay.IDI("NodeBody", node.ID), clay.EL{ // core.Node body
				Layout: clay.LAY{Sizing: core.GROWH, Padding: core.PA2},
			}, func() {
				node.Action.UI(node)
			})
		})
	})
}

// Moved UIInputPort and others to core
func UIContextMenu() {
	if ContextMenu == nil {
		return
	}
	// Use core.ZTOP for the menu. Since core.ZTOP is MaxInt16, we can't go higher.
	// We rely on the fact that this is rendered late in the frame (overlay).
	core.WithZIndex(core.ZTOP, func() {
		clay.CLAY(clay.ID("ContextMenu"), clay.EL{
			Floating: clay.FloatingElementConfig{
				AttachTo: clay.AttachToRoot,
				Offset:   clay.Vector2(ContextMenu.Pos),
				ZIndex:   core.ZTOP,
			},
			Layout:          clay.LAY{LayoutDirection: clay.TopToBottom, Padding: core.PA1},
			BackgroundColor: core.DarkGray,
			Border:          clay.BorderElementConfig{Color: core.Gray, Width: core.BA},
			CornerRadius:    core.RA1,
		}, func() {
			clay.OnHover(func(elementID clay.ElementID, pointerData clay.PointerData, userData any) {
				core.IsHoveringUI = true
			}, nil)

			for i, item := range ContextMenu.Items {
				core.UIButton(clay.IDI("ContextItem", i), core.UIButtonConfig{
					El: clay.EL{
						Layout:          clay.LAY{Padding: core.PVH(core.S2, core.S3), Sizing: core.GROWH},
						BackgroundColor: util.Tern(clay.Hovered(), core.HoverWhite, clay.Color{}),
					},
					OnClick: func(_ clay.ElementID, _ clay.PointerData, _ any) {
						item.Action()
						ContextMenu = nil
					},
					ZIndex: core.ZTOP,
				}, func() {
					clay.TEXT(item.Label, clay.TextElementConfig{TextColor: core.White})
				})
			}
		})
	})
}
