package core

import (
	"fmt"
	"os"
)

func SerializeGraph(g *Graph) ([]byte, error) {
	s := NewEncoder(4)

	// Nodes
	nodeCount := len(g.Nodes)
	SInt(s, &nodeCount)
	for _, n := range g.Nodes {
		SThing(s, n)
	}

	// Wires
	wireCount := len(g.Wires)
	SInt(s, &wireCount)
	for _, w := range g.Wires {
		SInt(s, &w.StartNode.ID)
		SInt(s, &w.StartPort)
		SInt(s, &w.EndNode.ID)
		SInt(s, &w.EndPort)
	}

	// Groups
	groupCount := len(g.Groups)
	SInt(s, &groupCount)
	for _, grp := range g.Groups {
		SThing(s, grp)
	}

	// Variables
	SMapStrStr(s, &g.Variables)

	if !s.Ok() {
		return nil, fmt.Errorf("serialization failed: %v", s.Errs)
	}

	return s.Bytes(), nil
}

func DeserializeGraph(data []byte) (*Graph, error) {
	s := NewDecoder(data)

	// Read into temporary variables first to avoid destroying state on failure
	g := NewGraph()

	// Nodes
	var nodeCount int
	if !SInt(s, &nodeCount) {
		return nil, fmt.Errorf("failed to read node count: %v", s.Errs)
	}

	nodeMap := make(map[int]*Node)
	seenIDs := make(map[int]bool)
	maxID := 0

	for range nodeCount {
		n := &Node{}
		if !SThing(s, n) {
			return nil, fmt.Errorf("failed to read node: %v", s.Errs)
		}

		// Deduplicate IDs to prevent Clay crashes
		if seenIDs[n.ID] {
			oldID := n.ID
			// Heuristic: Shift ID by large amount + current count to find a likely free slot.
			// This isn't perfect (could still collide with future nodes), but fixes the "concatenated file" case.
			n.ID = n.ID + 100000 + len(g.Nodes)
			fmt.Printf("Warning: Remapped duplicate Node ID %d to %d\n", oldID, n.ID)
		}
		seenIDs[n.ID] = true

		g.AddNode(n)
		nodeMap[n.ID] = n
		if n.ID > maxID {
			maxID = n.ID
		}
	}
	g.NextNodeID = maxID

	// Wires
	var wireCount int
	if !SInt(s, &wireCount) {
		return nil, fmt.Errorf("failed to read wire count: %v", s.Errs)
	}

	for range wireCount {
		var startNodeID, startPort, endNodeID, endPort int
		SInt(s, &startNodeID)
		SInt(s, &startPort)
		SInt(s, &endNodeID)
		SInt(s, &endPort)

		if !s.Ok() {
			return nil, fmt.Errorf("failed to read wire: %v", s.Errs)
		}

		startNode, ok1 := nodeMap[startNodeID]
		endNode, ok2 := nodeMap[endNodeID]

		if ok1 && ok2 {
			g.Wires = append(g.Wires, &Wire{
				StartNode: startNode,
				StartPort: startPort,
				EndNode:   endNode,
				EndPort:   endPort,
			})
		} else {
			fmt.Printf("Warning: skipping wire with missing nodes (%d -> %d)\n", startNodeID, endNodeID)
		}
	}

	// Groups
	if s.Version >= 3 {
		var groupCount int
		if !SInt(s, &groupCount) {
			return nil, fmt.Errorf("failed to read group count: %v", s.Errs)
		}

		maxGroupID := 0
		for range groupCount {
			grp := &Group{}
			if !SThing(s, grp) {
				return nil, fmt.Errorf("failed to read group: %v", s.Errs)
			}
			g.AddGroup(grp)
			if grp.ID > maxGroupID {
				maxGroupID = grp.ID
			}
		}
		g.NextGroupID = maxGroupID
	}

	// Variables
	if s.Version >= 4 {
		SMapStrStr(s, &g.Variables)
	}

	if !s.Ok() {
		return nil, fmt.Errorf("deserialization failed: %v", s.Errs)
	}

	return g, nil
}

func SaveGraph(path string, g *Graph) error {
	data, err := SerializeGraph(g)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func LoadGraph(path string) (*Graph, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return DeserializeGraph(data)
}

func MergeGraph(target *Graph, source *Graph) {
	if source == nil {
		return
	}

	// Map old IDs to new nodes (pointers in target)
	nodeMap := make(map[int]*Node)

	// 1. Remap and Add Groups
	for _, grp := range source.Groups {
		target.NextGroupID++
		newID := target.NextGroupID

		newGrp := *grp // Copy
		newGrp.ID = newID
		target.Groups = append(target.Groups, &newGrp)
	}

	// 2. Remap and Add Nodes
	for _, n := range source.Nodes {
		target.NextNodeID++
		newID := target.NextNodeID

		// Use Clone to safely copy node (avoids copying Mutex)
		newNode := n.Clone()
		newNode.ID = newID
		newNode.Graph = target

		target.Nodes = append(target.Nodes, newNode)
		nodeMap[n.ID] = newNode // Save mapping from OLD ID to NEW Node
	}

	// 3. Remap and Add Wires
	for _, w := range source.Wires {
		startNode := nodeMap[w.StartNode.ID]
		endNode := nodeMap[w.EndNode.ID]

		if startNode != nil && endNode != nil {
			newWire := &Wire{
				StartNode: startNode,
				StartPort: w.StartPort,
				EndNode:   endNode,
				EndPort:   w.EndPort,
			}
			target.Wires = append(target.Wires, newWire)
		}
	}

	// 4. Merge Variables
	if target.Variables == nil {
		target.Variables = make(map[string]string)
	}
	for k, v := range source.Variables {
		if _, exists := target.Variables[k]; !exists {
			target.Variables[k] = v
		}
	}
}
