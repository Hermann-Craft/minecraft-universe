package main

import (
	"testing"

	"github.com/go-gl/mathgl/mgl32"
	goecs "github.com/oneforx/go-ecs"
)

func TestAddPlanet(test *testing.T) {
	galaxyTest := &Galaxy{}

	if err := galaxyTest.Init(
		goecs.Identifier{Namespace: "unicube", Path: "galaxyTest"},
		mgl32.Vec3{0, 0, 0},
		mgl32.Quat{W: 0, V: mgl32.Vec3{0, 0, 0}},
	); err != nil {
		test.Fatal(err)
	}

	planetTest := &Planet{}

	planetTest.Init(
		goecs.Identifier{Namespace: "unicube", Path: "planetEarth"},
		mgl32.Vec3{0, 0, 0},
		mgl32.Quat{W: 0, V: mgl32.Vec3{0, 0, 0}},
	)

	if err := galaxyTest.AddPlanet(planetTest); err != nil {
		test.Fatal(err)
	}
}
