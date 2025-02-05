package main

import (
	"fmt"

	"github.com/go-gl/mathgl/mgl32"
	goecs "github.com/oneforx/go-ecs"
)

type UniversePosition mgl32.Vec3
type UniverseRotation mgl32.Quat

type Universe struct {
	galaxies map[string]*Galaxy

	// Technicals
	isInit bool
}

func (universe *Universe) Init() error {
	universe.galaxies = make(map[string]*Galaxy)

	universe.isInit = true
	return nil
}

func (universe *Universe) Render() {
	for _, galaxy := range universe.galaxies {
		galaxy.Render()
	}
}

// Do nothing if a galaxy with the same identifier already exist
func (universe *Universe) AddGalaxy(galaxy Galaxy) error {
	if !universe.isInit {
		return ErrUniverseNotInitialized
	}

	if !galaxy.IsInit() {
		return ErrGalaxyNotInitialized
	}

	if universe.galaxies[galaxy.Identifier.String()] != nil {
		return ErrGalaxyAlreadyExist
	}

	universe.galaxies[galaxy.Identifier.String()] = &galaxy

	return nil
}

// Can return null
func (universe *Universe) GetGalaxyById(galaxyId goecs.Identifier) *Galaxy {
	return universe.galaxies[galaxyId.String()]
}

// ERRORS
var ErrUniverseNotInitialized = fmt.Errorf("the universe is not initialized")
var ErrGalaxyAlreadyExist = fmt.Errorf("a galaxy with the same identifier already exist")
