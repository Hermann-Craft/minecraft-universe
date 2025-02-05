package main

import (
	"fmt"

	"github.com/go-gl/mathgl/mgl32"
	goecs "github.com/oneforx/go-ecs"
)

type GalaxyPosition mgl32.Vec3

type GalaxyRotation mgl32.Quat

type Galaxy struct {
	*goecs.Identifier
	UniversePosition *mgl32.Vec3
	UniverseRotation *mgl32.Quat

	Planets map[string]*Planet

	// Technicals
	isInit bool
}

func (galaxy *Galaxy) Init(identifier goecs.Identifier, position mgl32.Vec3, rotation mgl32.Quat) error {
	galaxy.Identifier = &identifier
	galaxy.UniversePosition = &position
	galaxy.UniverseRotation = &rotation

	galaxy.Planets = make(map[string]*Planet)

	galaxy.isInit = true
	return nil
}

func (galaxy *Galaxy) IsInit() bool {
	return galaxy.isInit
}

func (galaxy *Galaxy) Render() {
	for _, planet := range galaxy.Planets {
		if !planet.isInit {
			return
		}

		planet.Render()
	}
}

func (galaxy *Galaxy) AddPlanet(planet *Planet) error {
	if !galaxy.isInit {
		return ErrGalaxyNotInitialized
	}

	if !planet.IsInit() {
		return ErrPlanetNotInitialized
	}

	if galaxy.Planets[planet.Identifier.String()] != nil {
		return ErrPlanetAlreadyExist
	}

	return nil
}

var ErrGalaxyHasNoIdentifier = fmt.Errorf("the galaxy has no identifier")
var ErrGalaxyNotInitialized = fmt.Errorf("the galaxy is not initialized")
