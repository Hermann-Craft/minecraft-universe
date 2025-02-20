package main

import (
	"fmt"

	"github.com/go-gl/mathgl/mgl32"
	goecs "github.com/oneforx/go-ecs"
)

type Planet struct {
	*goecs.Identifier
	UniversePosition *mgl32.Vec3
	UniverseRotation *mgl32.Quat

	// Technicals

	// PlanetBlock is the far representation of the planet
	// as we dont want to render the chunks if we are far awau
	planetBlock *Block

	// Chunks is used when the camera is near of the planet by 500 bloc of distance
	chunks *[][][]*Chunk
	isInit bool
}

func (planet *Planet) Init(identifier goecs.Identifier, position mgl32.Vec3, rotation mgl32.Quat) error {
	planet.Identifier = &identifier
	planet.UniversePosition = &position
	planet.UniverseRotation = &rotation

	planet.isInit = true
	return nil
}

func (planet *Planet) Render() {

}

func (planet *Planet) PlanetBlockIsInit() bool {
	return planet.planetBlock != nil
}

func (planet *Planet) ChunkIsInit() bool {
	return planet.chunks != nil
}

func (planet *Planet) IsInit() bool {
	return planet.isInit
}

var ErrPlanetNotInitialized = fmt.Errorf("the planet is not initialized")
var ErrPlanetAlreadyExist = fmt.Errorf("a planet with the same identifier already exist")
