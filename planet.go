package main

type Planet struct {
	GalaxyPosition
	GalaxyRotation

	// Technicals
	PlanetBlock *Block
	Chunks      [][][]*Chunk
}
