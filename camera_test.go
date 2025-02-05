package main

import (
	"testing"

	"github.com/go-gl/mathgl/mgl32"
	goecs "github.com/oneforx/go-ecs"
	"github.com/stretchr/testify/assert"
)

func TestGetGalaxyPosition(test *testing.T) {
	cameraTest := &Camera{
		Position: mgl32.Vec3{90, 10, 90},
	}

	galaxyTest := &Galaxy{}

	galaxyTest.Init(
		goecs.Identifier{Namespace: "unicube", Path: "galaxyTest"},
		mgl32.Vec3{100, 0, 100},
		mgl32.Quat{W: 0, V: mgl32.Vec3{0, 0, 0}},
	)

	actualPosition := cameraTest.Position.Sub(*galaxyTest.UniversePosition)

	assert.Equal(test, mgl32.Vec3{-10, 10, -10}, actualPosition)
	// I guess Universe.Pos + Galaxy.UniversePosition - Camera.UniversePosition
	// Universe.Pos == 0,0,0
	// Galaxy.Pos ==
}
