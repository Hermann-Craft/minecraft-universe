package main

import (
	"testing"

	"github.com/go-gl/mathgl/mgl32"
	goecs "github.com/oneforx/go-ecs"
)

func TestAddGalaxy(test *testing.T) {
	universeTest := &Universe{}

	universeTest.Init()

	universeTest.AddGalaxy(Galaxy{})
}

func TestGetGalaxyById(test *testing.T) {
	universeTest := &Universe{}

	if err := universeTest.Init(); err != nil {
		test.Fatal(err)
	}

	galaxyTest := &Galaxy{}

	if err := galaxyTest.Init(
		goecs.Identifier{Namespace: "unicube", Path: "galaxyTest"},
		mgl32.Vec3{0, 0, 0},
		mgl32.Quat{W: 0, V: mgl32.Vec3{0, 0, 0}},
	); err != nil {
		test.Fatal(err)
	}

	if err := universeTest.AddGalaxy(*galaxyTest); err != nil {
		test.Fatal(err)
	}

	if universeTest.GetGalaxyById(*galaxyTest.Identifier) == nil {
		test.Fatal("The universe should contain a galaxy but it did not")
	}
}

func TestUniverseInit(t *testing.T) {
	// TODO: Test Universe.Init logic
}
