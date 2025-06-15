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

	Sun     *Sun
	Planets map[string]*Planet

	// Technicals
	isInit bool
}

func (galaxy *Galaxy) Init(identifier goecs.Identifier, position mgl32.Vec3, rotation mgl32.Quat) error {
	galaxy.Identifier = &identifier
	galaxy.UniversePosition = &position
	galaxy.UniverseRotation = &rotation

	galaxy.Planets = make(map[string]*Planet)

	// Initialiser le soleil
	sun := &Sun{}
	err := sun.Init(
		goecs.Identifier{Namespace: "core", Path: "sun"},
		mgl32.Vec3{100, 100, 100},                // Position du soleil
		mgl32.Quat{W: 1, V: mgl32.Vec3{0, 0, 0}}, // Rotation du soleil
	)
	if err != nil {
		return fmt.Errorf("failed to initialize sun: %v", err)
	}
	galaxy.Sun = sun

	galaxy.isInit = true
	return nil
}

func (galaxy *Galaxy) IsInit() bool {
	return galaxy.isInit
}

func (galaxy *Galaxy) Render(currentTime float64) {
	// Rendre le soleil
	if galaxy.Sun != nil {
		galaxy.Sun.Render(currentTime, galaxy)
	}

	// Rendre les planètes
	for _, planet := range galaxy.Planets {
		if !planet.IsInit() {
			return
		}

		planet.Render(currentTime)
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

	galaxy.Planets[planet.Identifier.String()] = planet
	return nil
}

var ErrGalaxyHasNoIdentifier = fmt.Errorf("the galaxy has no identifier")
var ErrGalaxyNotInitialized = fmt.Errorf("the galaxy is not initialized")
