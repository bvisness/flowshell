package app

import rl "github.com/gen2brain/raylib-go/raylib"

type V2 = rl.Vector2

type NodeCmd struct {
	Cmd string
}

type Node struct {
	ID  int
	Pos V2
	Cmd NodeCmd
}
