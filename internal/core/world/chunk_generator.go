package world

import (
	"fmt"
	"math"

	perlin "github.com/aquilax/go-perlin"
)

// ChunkGenerator gère la génération procédurale des chunks
type ChunkGenerator struct {
	noise  *perlin.Perlin
	config GenerationConfig
}

// GenerationConfig configuration pour la génération
type GenerationConfig struct {
	Base        float64 // Distance de base depuis le bord
	Amplitude   float64 // Amplitude du relief
	NoiseScale  float64 // Échelle du noise
	Octaves     int32   // Nombre d'octaves pour le noise
	Persistence float64 // Persistance du noise
	Lacunarity  float64 // Lacunarité du noise
}

// NewChunkGenerator crée un nouveau générateur de chunks
func NewChunkGenerator(seed int64, config GenerationConfig) *ChunkGenerator {
	noise := perlin.NewPerlin(2.0, 2.0, config.Octaves, seed)

	return &ChunkGenerator{
		noise:  noise,
		config: config,
	}
}

// GenerateChunk génère le contenu d'un chunk
func (cg *ChunkGenerator) GenerateChunk(chunk *Chunk) error {
	if chunk == nil {
		return fmt.Errorf("chunk cannot be nil")
	}

	chunk.SetState(ChunkStateGenerating)

	// Initialiser tous les blocs comme solides
	cg.initializeBlocks(chunk)

	// Si le chunk n'a pas de faces de bord, le laisser plein
	if len(chunk.BoundaryFaces) == 0 {
		chunk.SetState(ChunkStateGenerated)
		return nil
	}

	// Sculpter le terrain pour les chunks en bordure
	cg.sculptTerrain(chunk)

	// Ajouter de l'herbe sur les faces extérieures
	cg.addGrassOnFaces(chunk)

	chunk.SetState(ChunkStateGenerated)
	return nil
}

// initializeBlocks initialise tous les blocs comme solides
func (cg *ChunkGenerator) initializeBlocks(chunk *Chunk) {
	for x := 0; x < chunk.Size.Width; x++ {
		for y := 0; y < chunk.Size.Height; y++ {
			for z := 0; z < chunk.Size.Depth; z++ {
				chunk.Blocks[x][y][z] = NewStoneBlock()
			}
		}
	}
}

// sculptTerrain sculpte le terrain selon les faces de bordure
func (cg *ChunkGenerator) sculptTerrain(chunk *Chunk) {
	planetSize := chunk.Planet.Size
	totalSizeX := float64(planetSize.X()) * float64(chunk.Size.Width)
	totalSizeY := float64(planetSize.Y()) * float64(chunk.Size.Height)
	totalSizeZ := float64(planetSize.Z()) * float64(chunk.Size.Depth)

	for x := 0; x < chunk.Size.Width; x++ {
		for y := 0; y < chunk.Size.Height; y++ {
			for z := 0; z < chunk.Size.Depth; z++ {
				// Position globale du bloc
				globalX := float64(x) + float64(chunk.Position.X())
				globalY := float64(y) + float64(chunk.Position.Y())
				globalZ := float64(z) + float64(chunk.Position.Z())

				shouldCarve := false

				for _, face := range chunk.BoundaryFaces {
					if cg.shouldCarveForFace(face, globalX, globalY, globalZ, totalSizeX, totalSizeY, totalSizeZ) {
						shouldCarve = true
						break
					}
				}

				if shouldCarve {
					chunk.Blocks[x][y][z] = NewAirBlock()
				}
			}
		}
	}
}

// shouldCarveForFace détermine si on doit sculpter pour une face donnée
func (cg *ChunkGenerator) shouldCarveForFace(face WorldFace, globalX, globalY, globalZ, totalSizeX, totalSizeY, totalSizeZ float64) bool {
	noiseValue := cg.noise.Noise3D(globalX/cg.config.NoiseScale, globalY/cg.config.NoiseScale, globalZ/cg.config.NoiseScale)
	threshold := cg.config.Base + noiseValue*cg.config.Amplitude

	switch face {
	case WorldFaceTop:
		// Face supérieure : on garde les blocs près du sommet
		distFromTop := totalSizeY - globalY
		return distFromTop > threshold
	case WorldFaceBottom:
		// Face inférieure : on creuse depuis le bas
		distFromBottom := globalY
		return distFromBottom < threshold
	case WorldFaceRight:
		// Face droite : on creuse depuis la droite
		distFromRight := totalSizeX - globalX
		return distFromRight < threshold
	case WorldFaceLeft:
		// Face gauche : on creuse depuis la gauche
		distFromLeft := globalX
		return distFromLeft < threshold
	case WorldFaceFront:
		// Face avant : on creuse depuis l'avant
		distFromFront := totalSizeZ - globalZ
		return distFromFront < threshold
	case WorldFaceBack:
		// Face arrière : on creuse depuis l'arrière
		distFromBack := globalZ
		return distFromBack < threshold
	default:
		return false
	}
}

// addGrassOnFaces ajoute de l'herbe sur les faces extérieures
func (cg *ChunkGenerator) addGrassOnFaces(chunk *Chunk) {
	for _, face := range chunk.BoundaryFaces {
		cg.addGrassOnFace(chunk, face)
	}
}

// addGrassOnFace ajoute de l'herbe sur une face spécifique
func (cg *ChunkGenerator) addGrassOnFace(chunk *Chunk, face WorldFace) {
	switch face {
	case WorldFaceTop:
		for x := 0; x < chunk.Size.Width; x++ {
			for z := 0; z < chunk.Size.Depth; z++ {
				for y := chunk.Size.Height - 1; y >= 0; y-- {
					if chunk.Blocks[x][y][z].Type != BlockTypeAir {
						if chunk.Blocks[x][y][z].Type != BlockTypeGrass {
							chunk.Blocks[x][y][z] = NewGrassBlock()
							// Mettre de la dirt juste en dessous si c'est de la stone
							if y > 0 && chunk.Blocks[x][y-1][z].Type == BlockTypeStone {
								chunk.Blocks[x][y-1][z] = NewDirtBlock()
							}
						}
						break
					}
				}
			}
		}
	case WorldFaceBottom:
		for x := 0; x < chunk.Size.Width; x++ {
			for z := 0; z < chunk.Size.Depth; z++ {
				for y := 0; y < chunk.Size.Height; y++ {
					if chunk.Blocks[x][y][z].Type != BlockTypeAir {
						if chunk.Blocks[x][y][z].Type != BlockTypeGrass {
							chunk.Blocks[x][y][z] = NewGrassBlock()
							if y < chunk.Size.Height-1 && chunk.Blocks[x][y+1][z].Type == BlockTypeStone {
								chunk.Blocks[x][y+1][z] = NewDirtBlock()
							}
						}
						break
					}
				}
			}
		}
	case WorldFaceLeft:
		for y := 0; y < chunk.Size.Height; y++ {
			for z := 0; z < chunk.Size.Depth; z++ {
				for x := 0; x < chunk.Size.Width; x++ {
					if chunk.Blocks[x][y][z].Type != BlockTypeAir {
						if chunk.Blocks[x][y][z].Type != BlockTypeGrass {
							chunk.Blocks[x][y][z] = NewGrassBlock()
							if x < chunk.Size.Width-1 && chunk.Blocks[x+1][y][z].Type == BlockTypeStone {
								chunk.Blocks[x+1][y][z] = NewDirtBlock()
							}
						}
						break
					}
				}
			}
		}
	case WorldFaceRight:
		for y := 0; y < chunk.Size.Height; y++ {
			for z := 0; z < chunk.Size.Depth; z++ {
				for x := chunk.Size.Width - 1; x >= 0; x-- {
					if chunk.Blocks[x][y][z].Type != BlockTypeAir {
						if chunk.Blocks[x][y][z].Type != BlockTypeGrass {
							chunk.Blocks[x][y][z] = NewGrassBlock()
							if x > 0 && chunk.Blocks[x-1][y][z].Type == BlockTypeStone {
								chunk.Blocks[x-1][y][z] = NewDirtBlock()
							}
						}
						break
					}
				}
			}
		}
	case WorldFaceFront:
		for x := 0; x < chunk.Size.Width; x++ {
			for y := 0; y < chunk.Size.Height; y++ {
				for z := chunk.Size.Depth - 1; z >= 0; z-- {
					if chunk.Blocks[x][y][z].Type != BlockTypeAir {
						if chunk.Blocks[x][y][z].Type != BlockTypeGrass {
							chunk.Blocks[x][y][z] = NewGrassBlock()
							if z > 0 && chunk.Blocks[x][y][z-1].Type == BlockTypeStone {
								chunk.Blocks[x][y][z-1] = NewDirtBlock()
							}
						}
						break
					}
				}
			}
		}
	case WorldFaceBack:
		for x := 0; x < chunk.Size.Width; x++ {
			for y := 0; y < chunk.Size.Height; y++ {
				for z := 0; z < chunk.Size.Depth; z++ {
					if chunk.Blocks[x][y][z].Type != BlockTypeAir {
						if chunk.Blocks[x][y][z].Type != BlockTypeGrass {
							chunk.Blocks[x][y][z] = NewGrassBlock()
							if z < chunk.Size.Depth-1 && chunk.Blocks[x][y][z+1].Type == BlockTypeStone {
								chunk.Blocks[x][y][z+1] = NewDirtBlock()
							}
						}
						break
					}
				}
			}
		}
	}
}

// fractalNoise2D génère du bruit fractal 2D
func (cg *ChunkGenerator) fractalNoise2D(x, z float64, octaves int, persistence, lacunarity float64) float64 {
	total := 0.0
	frequency := 1.0
	amplitude := 1.0
	maxValue := 0.0

	for i := 0; i < octaves; i++ {
		total += cg.noise.Noise2D(x*frequency, z*frequency) * amplitude
		maxValue += amplitude
		amplitude *= persistence
		frequency *= lacunarity
	}

	// Normaliser le résultat
	noise := total / maxValue
	normalized := (noise + 1) * 0.5

	// Appliquer une transformation exponentielle
	flat := math.Pow(normalized, 2.0)

	// Appliquer une fonction smootherstep
	t := flat
	smoother := t * t * t * (t*(t*6-15) + 10)

	// Remapper le résultat en [-1, 1]
	return smoother*2 - 1
}
