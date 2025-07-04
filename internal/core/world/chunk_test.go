package world

import (
	"fmt"
	"testing"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/hermann-craft/unicube/internal/render"
	goecs "github.com/oneforx/go-ecs"
)

// Mock TextureAtlas pour les tests
type MockTextureAtlas struct{}

func (m *MockTextureAtlas) Bind(unit uint32) {}
func (m *MockTextureAtlas) GetTextureCoords(textureName string) (float32, float32, float32, float32) {
	return 0.0, 0.0, 1.0, 1.0
}
func (m *MockTextureAtlas) GetID() uint32 { return 0 }
func (m *MockTextureAtlas) Cleanup()      {}

// createMockRegistry crée un BlockRegistry pour les tests avec des modèles basiques
func createMockRegistry() *BlockRegistry {
	registry := NewBlockRegistry()

	// Créer des modèles basiques pour les tests
	stoneModel := &BlockModel{
		Textures: map[string]string{
			"all": "minecraft:block/stone",
		},
		Elements: []BlockElement{
			{
				From: []float32{0, 0, 0},
				To:   []float32{16, 16, 16},
				Faces: map[string]BlockFace{
					"north": {Texture: "minecraft:block/stone"},
					"south": {Texture: "minecraft:block/stone"},
					"east":  {Texture: "minecraft:block/stone"},
					"west":  {Texture: "minecraft:block/stone"},
					"up":    {Texture: "minecraft:block/stone"},
					"down":  {Texture: "minecraft:block/stone"},
				},
			},
		},
	}

	dirtModel := &BlockModel{
		Textures: map[string]string{
			"all": "minecraft:block/dirt",
		},
		Elements: []BlockElement{
			{
				From: []float32{0, 0, 0},
				To:   []float32{16, 16, 16},
				Faces: map[string]BlockFace{
					"north": {Texture: "minecraft:block/dirt"},
					"south": {Texture: "minecraft:block/dirt"},
					"east":  {Texture: "minecraft:block/dirt"},
					"west":  {Texture: "minecraft:block/dirt"},
					"up":    {Texture: "minecraft:block/dirt"},
					"down":  {Texture: "minecraft:block/dirt"},
				},
			},
		},
	}

	grassModel := &BlockModel{
		Textures: map[string]string{
			"top":    "minecraft:block/grass_block_top",
			"side":   "minecraft:block/grass_block_side",
			"bottom": "minecraft:block/dirt",
		},
		Elements: []BlockElement{
			{
				From: []float32{0, 0, 0},
				To:   []float32{16, 16, 16},
				Faces: map[string]BlockFace{
					"north": {Texture: "minecraft:block/grass_block_side"},
					"south": {Texture: "minecraft:block/grass_block_side"},
					"east":  {Texture: "minecraft:block/grass_block_side"},
					"west":  {Texture: "minecraft:block/grass_block_side"},
					"up":    {Texture: "minecraft:block/grass_block_top"},
					"down":  {Texture: "minecraft:block/dirt"},
				},
			},
		},
	}

	// Ajouter les modèles au registry
	registry.Models["minecraft:block/stone"] = stoneModel
	registry.Models["minecraft:block/dirt"] = dirtModel
	registry.Models["minecraft:block/grass_block"] = grassModel

	return registry
}

// createMockAtlas crée un TextureAtlas pour les tests
func createMockAtlas() render.TextureAtlas {
	return &MockTextureAtlas{}
}

func TestNewChunk(t *testing.T) {
	position := mgl32.Vec3{0, 0, 0}
	size := ChunkSize{Width: 16, Height: 32, Depth: 16}
	planet := &Planet{}
	registry := createMockRegistry()
	atlas := createMockAtlas()

	chunk := NewChunk(position, size, planet, registry, atlas)

	if chunk == nil {
		t.Fatal("NewChunk returned nil")
	}

	if chunk.GetState() != ChunkStateInitialized {
		t.Errorf("Expected initial state ChunkStateInitialized, got %v", chunk.GetState())
	}

	if chunk.Position != position {
		t.Errorf("Expected position %v, got %v", position, chunk.Position)
	}

	if chunk.Size != size {
		t.Errorf("Expected size %v, got %v", size, chunk.Size)
	}
}

func TestChunkStateTransitions(t *testing.T) {
	registry := createMockRegistry()
	atlas := createMockAtlas()
	chunk := NewChunk(mgl32.Vec3{0, 0, 0}, ChunkSize{16, 32, 16}, &Planet{}, registry, atlas)

	// Test des transitions d'état
	if chunk.GetState() != ChunkStateInitialized {
		t.Errorf("Expected initial state ChunkStateInitialized, got %v", chunk.GetState())
	}

	chunk.SetState(ChunkStateGenerating)
	if chunk.GetState() != ChunkStateGenerating {
		t.Errorf("Expected state ChunkStateGenerating, got %v", chunk.GetState())
	}

	chunk.SetState(ChunkStateReady)
	if !chunk.IsReady() {
		t.Error("Expected chunk to be ready")
	}

	if chunk.IsGenerating() {
		t.Error("Expected chunk to not be generating")
	}
}

func TestChunkBlockOperations(t *testing.T) {
	registry := createMockRegistry()
	atlas := createMockAtlas()
	chunk := NewChunk(mgl32.Vec3{0, 0, 0}, ChunkSize{16, 32, 16}, &Planet{}, registry, atlas)

	// Test SetBlock et GetBlock
	block := NewStoneBlock()
	err := chunk.SetBlock(0, 0, 0, block)
	if err != nil {
		t.Errorf("SetBlock failed: %v", err)
	}

	retrievedBlock, err := chunk.GetBlock(0, 0, 0)
	if err != nil {
		t.Errorf("GetBlock failed: %v", err)
	}

	if retrievedBlock.Type != BlockTypeStone {
		t.Errorf("Expected BlockTypeStone, got %v", retrievedBlock.Type)
	}

	// Test des coordonnées hors limites
	_, err = chunk.GetBlock(20, 0, 0)
	if err == nil {
		t.Error("Expected error for out of bounds coordinates")
	}

	err = chunk.SetBlock(20, 0, 0, block)
	if err == nil {
		t.Error("Expected error for out of bounds coordinates")
	}
}

func TestChunkError(t *testing.T) {
	registry := createMockRegistry()
	atlas := createMockAtlas()
	chunk := NewChunk(mgl32.Vec3{0, 0, 0}, ChunkSize{16, 32, 16}, &Planet{}, registry, atlas)

	// Test error handling
	testError := "test error"
	chunk.SetError(fmt.Errorf(testError))

	if chunk.GetState() != ChunkStateError {
		t.Errorf("Expected state ChunkStateError, got %v", chunk.GetState())
	}

	if chunk.GetError() == nil {
		t.Error("Expected error to be set")
	}

	if chunk.GetError().Error() != testError {
		t.Errorf("Expected error message '%s', got '%s'", testError, chunk.GetError().Error())
	}
}

func TestChunkBoundingBox(t *testing.T) {
	registry := createMockRegistry()
	atlas := createMockAtlas()
	position := mgl32.Vec3{10, 20, 30}
	size := ChunkSize{Width: 16, Height: 32, Depth: 16}
	chunk := NewChunk(position, size, &Planet{}, registry, atlas)

	bbox := chunk.GetBoundingBox()
	expectedMin := position
	expectedMax := position.Add(mgl32.Vec3{float32(size.Width), float32(size.Height), float32(size.Depth)})

	if bbox.Min != expectedMin {
		t.Errorf("Expected min %v, got %v", expectedMin, bbox.Min)
	}

	if bbox.Max != expectedMax {
		t.Errorf("Expected max %v, got %v", expectedMax, bbox.Max)
	}
}

func TestChunkGenerator(t *testing.T) {
	config := GenerationConfig{
		Base:        10.0,
		Amplitude:   5.0,
		NoiseScale:  20.0,
		Octaves:     4,
		Persistence: 0.5,
		Lacunarity:  2.0,
	}

	generator := NewChunkGenerator(42, config)
	if generator == nil {
		t.Fatal("NewChunkGenerator returned nil")
	}

	planet := &Planet{
		Size: mgl32.Vec3{3, 3, 3},
	}

	registry := createMockRegistry()
	atlas := createMockAtlas()
	chunk := NewChunk(mgl32.Vec3{0, 0, 0}, ChunkSize{16, 32, 16}, planet, registry, atlas)
	chunk.BoundaryFaces = []WorldFace{WorldFaceTop}

	err := generator.GenerateChunk(chunk)
	if err != nil {
		t.Errorf("GenerateChunk failed: %v", err)
	}

	if chunk.GetState() != ChunkStateGenerated {
		t.Errorf("Expected state ChunkStateGenerated, got %v", chunk.GetState())
	}
}

func TestNewPlanet(t *testing.T) {
	id := goecs.Identifier{Namespace: "core", Path: "test-planet"}
	position := mgl32.Vec3{0, 0, 0}
	size := mgl32.Vec3{3, 3, 3}
	chunkSize := ChunkSize{Width: 16, Height: 32, Depth: 16}
	seed := int64(42)

	planet := NewPlanet(id, position, size, chunkSize, seed)

	if planet == nil {
		t.Fatal("NewPlanet returned nil")
	}

	if planet.GetState() != PlanetStateUninitialized {
		t.Errorf("Expected initial state PlanetStateUninitialized, got %v", planet.GetState())
	}

	if planet.ID != id {
		t.Errorf("Expected ID %v, got %v", id, planet.ID)
	}

	if planet.Position != position {
		t.Errorf("Expected position %v, got %v", position, planet.Position)
	}

	if planet.Size != size {
		t.Errorf("Expected size %v, got %v", size, planet.Size)
	}

	if planet.Seed != seed {
		t.Errorf("Expected seed %d, got %d", seed, planet.Seed)
	}
}

func TestPlanetInitializeChunks(t *testing.T) {
	planet := NewPlanet(
		goecs.Identifier{Namespace: "core", Path: "test-planet"},
		mgl32.Vec3{0, 0, 0},
		mgl32.Vec3{2, 2, 2}, // 2x2x2 chunks
		ChunkSize{Width: 16, Height: 32, Depth: 16},
		42,
	)

	registry := createMockRegistry()
	atlas := createMockAtlas()

	err := planet.InitializeChunks(registry, atlas)
	if err != nil {
		t.Errorf("InitializeChunks failed: %v", err)
	}

	if planet.GetState() != PlanetStateInitialized {
		t.Errorf("Expected state PlanetStateInitialized, got %v", planet.GetState())
	}

	// Vérifier que tous les chunks ont été créés
	for x := 0; x < 2; x++ {
		for y := 0; y < 2; y++ {
			for z := 0; z < 2; z++ {
				chunk, err := planet.GetChunk(x, y, z)
				if err != nil {
					t.Errorf("GetChunk(%d, %d, %d) failed: %v", x, y, z, err)
				}
				if chunk == nil {
					t.Errorf("Chunk at (%d, %d, %d) is nil", x, y, z)
				}
				if chunk.GetState() != ChunkStateInitialized {
					t.Errorf("Expected chunk state ChunkStateInitialized, got %v", chunk.GetState())
				}
			}
		}
	}

	// Vérifier les coordonnées hors limites
	_, err = planet.GetChunk(3, 0, 0)
	if err == nil {
		t.Error("Expected error for out of bounds chunk coordinates")
	}
}
