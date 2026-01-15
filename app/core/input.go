package core

import (
	"github.com/bvisness/flowshell/clay"
)

// Global state to track if the mouse is currently hovering over any HIGH PRIORITY UI element
// (buttons, textboxes, dropdowns).
// This blocks ALL global actions (Node Drag, Wire Drag, Panning).
var IsHoveringUI bool

// Global state to track if the mouse is currently hovering over any LOW PRIORITY UI element
// (panels, backgrounds, node bodies).
// This blocks Panning, but allows specific global actions like Node Dragging (if the geometry matches).
var IsHoveringPanel bool

var UIInput InputManager

type InputManager struct {
	pointerDown              bool
	pointerPressedThisFrame  bool
	pointerReleasedThisFrame bool

	pointerDownOwnerID  uint32
	pointerDownOwnerSet bool
	pointerDownMaxZ     int16

	MouseWheelConsumed bool
}

func (m *InputManager) BeginFrame(pointerDown bool) {
	m.MouseWheelConsumed = false
	m.pointerPressedThisFrame = pointerDown && !m.pointerDown
	m.pointerReleasedThisFrame = !pointerDown && m.pointerDown
	m.pointerDown = pointerDown

	if m.pointerPressedThisFrame {
		m.pointerDownOwnerID = 0
		m.pointerDownOwnerSet = false
		m.pointerDownMaxZ = -32768
	}
}

func (m *InputManager) EndFrame() {
	m.pointerPressedThisFrame = false
	m.pointerReleasedThisFrame = false
}

func (m *InputManager) RegisterPointerDown(owner clay.ElementID, pointerData clay.PointerData, zIndex int16) {
	if !m.pointerPressedThisFrame {
		return
	}
	if !m.pointerDownOwnerSet || zIndex >= m.pointerDownMaxZ {
		m.pointerDownOwnerID = owner.ID
		m.pointerDownOwnerSet = true
		m.pointerDownMaxZ = zIndex
	}
}
func (m *InputManager) IsClick(owner clay.ElementID, pointerData clay.PointerData) bool {
	if !m.pointerReleasedThisFrame {
		return false
	}
	if !m.pointerDownOwnerSet {
		return false
	}
	return m.pointerDownOwnerID == owner.ID
}

func (m *InputManager) IsPressed(owner clay.ElementID) bool {
	if !m.pointerPressedThisFrame {
		return false
	}
	if !m.pointerDownOwnerSet {
		return false
	}
	return m.pointerDownOwnerID == owner.ID
}
