package main

import (
	"fmt"

	goecs "github.com/oneforx/go-ecs"
)

type Planet struct {
	*goecs.Identifier
	*GalaxyPosition
	*GalaxyRotation

	// Technicals

	// PlanetBlock is the far representation of the planet
	// as we dont want to render the chunks if we are far awau
	planetBlock *Block

	// Chunks is used when the camera is near of the planet by 500 bloc of distance
	chunks *[][][]*Chunk
	isInit bool
}

func (planet *Planet) Init(identifier goecs.Identifier, position GalaxyPosition, rotation GalaxyRotation) error {
	planet.Identifier = &identifier
	planet.GalaxyPosition = &position
	planet.GalaxyRotation = &rotation

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
