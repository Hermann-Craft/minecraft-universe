package world

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/go-gl/mathgl/mgl32"
	goecs "github.com/oneforx/go-ecs"

	"github.com/hermann-craft/unicube/internal/core/camera"
	"github.com/hermann-craft/unicube/internal/core/geom"
	"github.com/hermann-craft/unicube/internal/render"
)

// PlanetState represents the state of a planet (StateLogic principle)
type PlanetState int

const (
	PlanetStateUninitialized PlanetState = iota
	PlanetStateInitialized
	PlanetStatePopulating
	PlanetStateReady
	PlanetStateError
)

// Planet represents a cubic planet with explicit states
type Planet struct {
	// Identity
	ID       goecs.Identifier
	Position mgl32.Vec3
	Rotation mgl32.Quat
	Size     mgl32.Vec3 // Size in chunks

	// Explicit state (StateLogic)
	State PlanetState
	mu    sync.RWMutex

	// World data
	Chunks    [][][]*Chunk
	ChunkSize ChunkSize

	// Configuration
	Seed int64

	// Metadata
	LastUpdate time.Time
	Error      error

	BlockRegistry *BlockRegistry
	TextureAtlas  render.TextureAtlas
}

// NewPlanet creates a new planet
func NewPlanet(id goecs.Identifier, position mgl32.Vec3, size mgl32.Vec3, chunkSize ChunkSize, seed int64) *Planet {
	return &Planet{
		ID:        id,
		Position:  position,
		Rotation:  mgl32.QuatIdent(),
		Size:      size,
		ChunkSize: chunkSize,
		Seed:      seed,
		State:     PlanetStateUninitialized,
	}
}

// SetState changes the state of the planet in a thread-safe manner
func (p *Planet) SetState(state PlanetState) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.State = state
	p.LastUpdate = time.Now()
}

// GetState returns the current state of the planet
func (p *Planet) GetState() PlanetState {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.State
}

// IsReady checks if the planet is ready
func (p *Planet) IsReady() bool {
	return p.GetState() == PlanetStateReady
}

// SetError sets an error on the planet
func (p *Planet) SetError(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Error = err
	p.State = PlanetStateError
}

// GetError returns the planet's error
func (p *Planet) GetError() error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Error
}

// GetChunk returns a chunk at the given coordinates
func (p *Planet) GetChunk(x, y, z int) (*Chunk, error) {
	if x < 0 || x >= int(p.Size.X()) ||
		y < 0 || y >= int(p.Size.Y()) ||
		z < 0 || z >= int(p.Size.Z()) {
		return nil, fmt.Errorf("chunk coordinates out of bounds: (%d, %d, %d)", x, y, z)
	}
	return p.Chunks[x][y][z], nil
}

// SetChunk sets a chunk at the given coordinates
func (p *Planet) SetChunk(x, y, z int, chunk *Chunk) error {
	if x < 0 || x >= int(p.Size.X()) ||
		y < 0 || y >= int(p.Size.Y()) ||
		z < 0 || z >= int(p.Size.Z()) {
		return fmt.Errorf("chunk coordinates out of bounds: (%d, %d, %d)", x, y, z)
	}
	p.Chunks[x][y][z] = chunk
	return nil
}

// GetPosition returns the position of the planet
func (p *Planet) GetPosition() mgl32.Vec3 {
	return p.Position
}

// GetBoundingBox returns the bounding box of the planet
func (p *Planet) GetBoundingBox() geom.BoundingBox {
	totalWidth := float32(p.Size.X()) * float32(p.ChunkSize.Width)
	totalHeight := float32(p.Size.Y()) * float32(p.ChunkSize.Height)
	totalDepth := float32(p.Size.Z()) * float32(p.ChunkSize.Depth)

	return geom.NewBoundingBox(
		p.Position,
		p.Position.Add(mgl32.Vec3{totalWidth, totalHeight, totalDepth}),
	)
}

// GetBlockAt retrieves a block at the given world coordinates.
// It returns the block, the chunk it belongs to, and its local coordinates.
func (p *Planet) GetBlockAt(x, y, z int) (*Block, *Chunk, error) {
	chunkX, chunkY, chunkZ := x/p.ChunkSize.Width, y/p.ChunkSize.Height, z/p.ChunkSize.Depth
	bx, by, bz := x%p.ChunkSize.Width, y%p.ChunkSize.Height, z%p.ChunkSize.Depth
	if bx < 0 {
		bx += p.ChunkSize.Width
	}
	if by < 0 {
		by += p.ChunkSize.Height
	}
	if bz < 0 {
		bz += p.ChunkSize.Depth
	}

	chunk, err := p.GetChunk(chunkX, chunkY, chunkZ)
	if err != nil {
		return nil, nil, err
	}
	block, err := chunk.GetBlock(bx, by, bz)
	return &block, chunk, err
}

// SetBlockAt sets a block at the given world coordinates
func (p *Planet) SetBlockAt(x, y, z int, block Block) error {
	chunkX, chunkY, chunkZ := x/p.ChunkSize.Width, y/p.ChunkSize.Height, z/p.ChunkSize.Depth
	bx, by, bz := x%p.ChunkSize.Width, y%p.ChunkSize.Height, z%p.ChunkSize.Depth
	if bx < 0 {
		bx += p.ChunkSize.Width
	}
	if by < 0 {
		by += p.ChunkSize.Height
	}
	if bz < 0 {
		bz += p.ChunkSize.Depth
	}

	chunk, err := p.GetChunk(chunkX, chunkY, chunkZ)
	if err != nil {
		return err
	}
	return chunk.SetBlock(bx, by, bz, block)
}

// InitializeChunks initializes the chunk grid
func (p *Planet) InitializeChunks(registry *BlockRegistry, atlas render.TextureAtlas) error {
	p.BlockRegistry = registry
	p.TextureAtlas = atlas
	sizeX, sizeY, sizeZ := int(p.Size.X()), int(p.Size.Y()), int(p.Size.Z())
	p.Chunks = make([][][]*Chunk, sizeX)
	for x := 0; x < sizeX; x++ {
		p.Chunks[x] = make([][]*Chunk, sizeY)
		for y := 0; y < sizeY; y++ {
			p.Chunks[x][y] = make([]*Chunk, sizeZ)
			for z := 0; z < sizeZ; z++ {
				// Calculate the chunk's world position relative to the planet's Position
				offset := mgl32.Vec3{float32(x * p.ChunkSize.Width), float32(y * p.ChunkSize.Height), float32(z * p.ChunkSize.Depth)}
				globalPos := p.Position.Add(offset)
				p.Chunks[x][y][z] = NewChunk(globalPos, p.ChunkSize, p, registry, atlas)
			}
		}
	}
	p.SetState(PlanetStateInitialized)
	p.Populate() // Populate right after initialization
	return nil
}

// Populate populates the planet with blocks
func (p *Planet) Populate() {
	p.SetState(PlanetStatePopulating)
	log.Println("Populating planet...")
	for x := 0; x < int(p.Size.X())*p.ChunkSize.Width; x++ {
		for z := 0; z < int(p.Size.Z())*p.ChunkSize.Depth; z++ {
			height := 30 // Simplified height
			for y := 0; y < int(p.Size.Y())*p.ChunkSize.Height; y++ {
				var block Block
				if y < height-1 {
					block.Type = BlockTypeStone
				} else if y < height {
					block.Type = BlockTypeDirt
				} else if y == height {
					block.Type = BlockTypeGrass
				} else {
					block.Type = BlockTypeAir
				}
				p.SetBlockAt(x, y, z, block)
			}
		}
	}
	p.SetState(PlanetStateReady)
	log.Println("Planet populated.")
}

// Update updates the state of the planet
func (p *Planet) Update(dt float64) {
	if p.GetState() != PlanetStateReady {
		return
	}
	for x := 0; x < int(p.Size.X()); x++ {
		for y := 0; y < int(p.Size.Y()); y++ {
			for z := 0; z < int(p.Size.Z()); z++ {
				chunk := p.Chunks[x][y][z]
				if chunk.GetState() == ChunkStateInitialized {
					vertices, indices := chunk.GenerateVertices()
					if chunk.GetState() == ChunkStateGenerated {
						chunk.GenerateMesh(vertices, indices)
					}
				}
			}
		}
	}
}

// Render renders the planet
func (p *Planet) Render(currentTime float64, renderer *render.Renderer, camera *camera.Camera, projection mgl32.Mat4) {
	for x := 0; x < int(p.Size.X()); x++ {
		for y := 0; y < int(p.Size.Y()); y++ {
			for z := 0; z < int(p.Size.Z()); z++ {
				p.Chunks[x][y][z].Render(currentTime, renderer, camera, projection)
			}
		}
	}
}

// GetSurfaceHeight finds the surface height (first non-air block from top) at the given x,z coordinates
func (p *Planet) GetSurfaceHeight(x, z int) (int, error) {
	maxY := int(p.Size.Y()) * p.ChunkSize.Height

	// Start from the top and go down to find the first non-air block
	for y := maxY - 1; y >= 0; y-- {
		block, _, err := p.GetBlockAt(x, y, z)
		if err != nil {
			continue // Skip invalid coordinates
		}
		if block != nil && block.Type != BlockTypeAir {
			return y + 1, nil // Return the position above the block (spawn position)
		}
	}

	// If no surface found, return a default height
	return 32, nil
}

// GetSafeSpawnPosition calculates a safe spawn position for the player
func (p *Planet) GetSafeSpawnPosition() (mgl32.Vec3, error) {
	if p.GetState() != PlanetStateReady {
		return mgl32.Vec3{}, fmt.Errorf("planet is not ready for spawn calculation")
	}

	// Try to find a safe spot around the center of the world
	centerX := int(p.Size.X()) * p.ChunkSize.Width / 2
	centerZ := int(p.Size.Z()) * p.ChunkSize.Depth / 2

	log.Printf("Calculating spawn position around center: (%d, %d)", centerX, centerZ)

	// Search in a small area around the center for a suitable spawn location
	for dx := -3; dx <= 3; dx++ {
		for dz := -3; dz <= 3; dz++ {
			x := centerX + dx
			z := centerZ + dz

			surfaceY, err := p.GetSurfaceHeight(x, z)
			if err != nil {
				continue
			}

			// Check if there's enough vertical space (3 blocks high for safety) for the player
			if surfaceY+3 < int(p.Size.Y())*p.ChunkSize.Height {
				blockAbove, _, err1 := p.GetBlockAt(x, surfaceY, z)
				blockAbove2, _, err2 := p.GetBlockAt(x, surfaceY+1, z)
				blockAbove3, _, err3 := p.GetBlockAt(x, surfaceY+2, z)

				if err1 == nil && err2 == nil && err3 == nil &&
					blockAbove != nil && blockAbove.Type == BlockTypeAir &&
					blockAbove2 != nil && blockAbove2.Type == BlockTypeAir &&
					blockAbove3 != nil && blockAbove3.Type == BlockTypeAir {
					// Add a small safety margin: spawn 0.2 blocks above the surface
					spawnPos := mgl32.Vec3{float32(x) + 0.5, float32(surfaceY) + 0.2, float32(z) + 0.5}
					log.Printf("Found safe spawn position: (%.1f, %.1f, %.1f)", spawnPos.X(), spawnPos.Y(), spawnPos.Z())
					return spawnPos, nil
				}
			}
		}
	}

	// Fallback to a default position if no safe spot found
	defaultY, _ := p.GetSurfaceHeight(centerX, centerZ)
	fallbackPos := mgl32.Vec3{float32(centerX) + 0.5, float32(defaultY) + 0.2, float32(centerZ) + 0.5}
	log.Printf("Using fallback spawn position: (%.1f, %.1f, %.1f)", fallbackPos.X(), fallbackPos.Y(), fallbackPos.Z())
	return fallbackPos, nil
}
