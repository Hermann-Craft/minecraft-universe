package world

import (
	"testing"

	"github.com/go-gl/mathgl/mgl32"
	goecs "github.com/oneforx/go-ecs"
)

func TestNewChunk(t *testing.T) {
	position := mgl32.Vec3{0, 0, 0}
	size := ChunkSize{Width: 16, Height: 32, Depth: 16}
	planet := &Planet{}

	chunk := NewChunk(position, size, planet)

	if chunk == nil {
		t.Fatal("NewChunk returned nil")
	}

	if chunk.GetState() != ChunkStateUninitialized {
		t.Errorf("Expected initial state ChunkStateUninitialized, got %v", chunk.GetState())
	}

	if chunk.Position != position {
		t.Errorf("Expected position %v, got %v", position, chunk.Position)
	}

	if chunk.Size != size {
		t.Errorf("Expected size %v, got %v", size, chunk.Size)
	}
}

func TestChunkStateTransitions(t *testing.T) {
	chunk := NewChunk(mgl32.Vec3{0, 0, 0}, ChunkSize{16, 32, 16}, &Planet{})

	// Test des transitions d'état
	chunk.SetState(ChunkStateInitialized)
	if chunk.GetState() != ChunkStateInitialized {
		t.Errorf("Expected state ChunkStateInitialized, got %v", chunk.GetState())
	}

	chunk.SetState(ChunkStateGenerating)
	if chunk.GetState() != ChunkStateGenerating {
		t.Errorf("Expected state ChunkStateGenerating, got %v", chunk.GetState())
	}

	chunk.SetState(ChunkStateReady)
	if !chunk.IsReady() {
		t.Error("Expected chunk to be ready")
	}
}

func TestChunkBlockOperations(t *testing.T) {
	chunk := NewChunk(mgl32.Vec3{0, 0, 0}, ChunkSize{16, 32, 16}, &Planet{})

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

	chunk := NewChunk(mgl32.Vec3{0, 0, 0}, ChunkSize{16, 32, 16}, planet)
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

	err := planet.InitializeChunks()
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

	// Vérifier que les chunks de bordure ont les bonnes faces
	chunk, _ := planet.GetChunk(0, 0, 0) // Coin inférieur gauche arrière
	if len(chunk.BoundaryFaces) != 3 {
		t.Errorf("Expected 3 boundary faces for corner chunk, got %d", len(chunk.BoundaryFaces))
	}

	// Vérifier les coordonnées hors limites
	_, err = planet.GetChunk(3, 0, 0)
	if err == nil {
		t.Error("Expected error for out of bounds chunk coordinates")
	}
}
