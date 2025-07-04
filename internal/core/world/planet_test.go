package world

import (
	"fmt"
	"testing"

	"github.com/go-gl/mathgl/mgl32"
	goecs "github.com/oneforx/go-ecs"
)

// TestPlanetCreation teste la création d'une planète
func TestPlanetCreation(t *testing.T) {
	id := goecs.Identifier{Namespace: "test", Path: "planet"}
	position := mgl32.Vec3{10, 20, 30}
	size := mgl32.Vec3{4, 4, 4}
	chunkSize := ChunkSize{Width: 16, Height: 32, Depth: 16}
	seed := int64(12345)

	planet := NewPlanet(id, position, size, chunkSize, seed)

	if planet == nil {
		t.Fatal("NewPlanet returned nil")
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

	if planet.ChunkSize != chunkSize {
		t.Errorf("Expected chunk size %v, got %v", chunkSize, planet.ChunkSize)
	}

	if planet.Seed != seed {
		t.Errorf("Expected seed %d, got %d", seed, planet.Seed)
	}

	if planet.GetState() != PlanetStateUninitialized {
		t.Errorf("Expected initial state PlanetStateUninitialized, got %v", planet.GetState())
	}
}

// TestPlanetStateTransitions teste les transitions d'état
func TestPlanetStateTransitions(t *testing.T) {
	planet := NewPlanet(
		goecs.Identifier{Namespace: "test", Path: "planet"},
		mgl32.Vec3{0, 0, 0},
		mgl32.Vec3{2, 2, 2},
		ChunkSize{16, 32, 16},
		42,
	)

	// Test initial state
	if planet.GetState() != PlanetStateUninitialized {
		t.Errorf("Expected initial state PlanetStateUninitialized, got %v", planet.GetState())
	}

	// Test state transitions
	planet.SetState(PlanetStateInitialized)
	if planet.GetState() != PlanetStateInitialized {
		t.Errorf("Expected state PlanetStateInitialized, got %v", planet.GetState())
	}

	planet.SetState(PlanetStatePopulating)
	if planet.GetState() != PlanetStatePopulating {
		t.Errorf("Expected state PlanetStatePopulating, got %v", planet.GetState())
	}

	planet.SetState(PlanetStateReady)
	if !planet.IsReady() {
		t.Error("Expected planet to be ready")
	}

	// Test error state
	testError := fmt.Errorf("test error")
	planet.SetError(testError)
	if planet.GetState() != PlanetStateError {
		t.Errorf("Expected state PlanetStateError, got %v", planet.GetState())
	}

	if planet.GetError() != testError {
		t.Errorf("Expected error %v, got %v", testError, planet.GetError())
	}
}

// TestPlanetChunkManagement teste la gestion des chunks
func TestPlanetChunkManagement(t *testing.T) {
	planet := NewPlanet(
		goecs.Identifier{Namespace: "test", Path: "planet"},
		mgl32.Vec3{0, 0, 0},
		mgl32.Vec3{3, 3, 3},
		ChunkSize{16, 32, 16},
		42,
	)

	registry := createMockRegistry()
	atlas := createMockAtlas()

	// Initialize chunks
	err := planet.InitializeChunks(registry, atlas)
	if err != nil {
		t.Errorf("InitializeChunks failed: %v", err)
	}

	// Test chunk access
	chunk, err := planet.GetChunk(1, 1, 1)
	if err != nil {
		t.Errorf("GetChunk failed: %v", err)
	}

	if chunk == nil {
		t.Error("Expected chunk to exist")
	}

	// Test chunk position
	expectedPos := mgl32.Vec3{16, 32, 16} // 1*16, 1*32, 1*16
	if chunk.Position != expectedPos {
		t.Errorf("Expected chunk position %v, got %v", expectedPos, chunk.Position)
	}

	// Test out of bounds
	_, err = planet.GetChunk(5, 5, 5)
	if err == nil {
		t.Error("Expected error for out of bounds chunk")
	}

	// Test SetChunk
	newChunk := NewChunk(mgl32.Vec3{0, 0, 0}, ChunkSize{16, 32, 16}, planet, registry, atlas)
	err = planet.SetChunk(2, 2, 2, newChunk)
	if err != nil {
		t.Errorf("SetChunk failed: %v", err)
	}

	retrievedChunk, err := planet.GetChunk(2, 2, 2)
	if err != nil {
		t.Errorf("GetChunk after SetChunk failed: %v", err)
	}

	if retrievedChunk != newChunk {
		t.Error("Expected retrieved chunk to be the same as set chunk")
	}
}

// TestPlanetBlockOperations teste les opérations sur les blocs
func TestPlanetBlockOperations(t *testing.T) {
	planet := NewPlanet(
		goecs.Identifier{Namespace: "test", Path: "planet"},
		mgl32.Vec3{0, 0, 0},
		mgl32.Vec3{2, 2, 2},
		ChunkSize{16, 32, 16},
		42,
	)

	registry := createMockRegistry()
	atlas := createMockAtlas()

	err := planet.InitializeChunks(registry, atlas)
	if err != nil {
		t.Errorf("InitializeChunks failed: %v", err)
	}

	// Test SetBlockAt and GetBlockAt
	block := NewStoneBlock()
	err = planet.SetBlockAt(5, 10, 7, block)
	if err != nil {
		t.Errorf("SetBlockAt failed: %v", err)
	}

	retrievedBlock, chunk, err := planet.GetBlockAt(5, 10, 7)
	if err != nil {
		t.Errorf("GetBlockAt failed: %v", err)
	}

	if retrievedBlock == nil {
		t.Error("Expected retrieved block to not be nil")
	}

	if retrievedBlock.Type != BlockTypeStone {
		t.Errorf("Expected BlockTypeStone, got %v", retrievedBlock.Type)
	}

	if chunk == nil {
		t.Error("Expected chunk to not be nil")
	}

	// Test block coordinates at chunk boundaries
	// Block at (16, 32, 16) should be in chunk (1, 1, 1)
	err = planet.SetBlockAt(16, 32, 16, NewDirtBlock())
	if err != nil {
		t.Errorf("SetBlockAt at chunk boundary failed: %v", err)
	}

	retrievedBlock, chunk, err = planet.GetBlockAt(16, 32, 16)
	if err != nil {
		t.Errorf("GetBlockAt at chunk boundary failed: %v", err)
	}

	if retrievedBlock.Type != BlockTypeDirt {
		t.Errorf("Expected BlockTypeDirt, got %v", retrievedBlock.Type)
	}

	// Verify we got the right chunk
	expectedChunkPos := mgl32.Vec3{16, 32, 16}
	if chunk.Position != expectedChunkPos {
		t.Errorf("Expected chunk position %v, got %v", expectedChunkPos, chunk.Position)
	}
}

// TestPlanetBoundingBox teste la boîte englobante de la planète
func TestPlanetBoundingBox(t *testing.T) {
	position := mgl32.Vec3{100, 200, 300}
	size := mgl32.Vec3{3, 2, 4}
	chunkSize := ChunkSize{Width: 16, Height: 32, Depth: 16}

	planet := NewPlanet(
		goecs.Identifier{Namespace: "test", Path: "planet"},
		position,
		size,
		chunkSize,
		42,
	)

	bbox := planet.GetBoundingBox()

	expectedMin := position
	expectedMax := position.Add(mgl32.Vec3{
		float32(size.X()) * float32(chunkSize.Width),
		float32(size.Y()) * float32(chunkSize.Height),
		float32(size.Z()) * float32(chunkSize.Depth),
	})

	if bbox.Min != expectedMin {
		t.Errorf("Expected min %v, got %v", expectedMin, bbox.Min)
	}

	if bbox.Max != expectedMax {
		t.Errorf("Expected max %v, got %v", expectedMax, bbox.Max)
	}
}

// TestPlanetSurfaceHeight teste la détection de la hauteur de surface
func TestPlanetSurfaceHeight(t *testing.T) {
	planet := NewPlanet(
		goecs.Identifier{Namespace: "test", Path: "planet"},
		mgl32.Vec3{0, 0, 0},
		mgl32.Vec3{2, 2, 2},
		ChunkSize{16, 32, 16},
		42,
	)

	registry := createMockRegistry()
	atlas := createMockAtlas()

	err := planet.InitializeChunks(registry, atlas)
	if err != nil {
		t.Errorf("InitializeChunks failed: %v", err)
	}

	// Set up a simple surface
	planet.SetBlockAt(5, 10, 7, NewStoneBlock())
	planet.SetBlockAt(5, 11, 7, NewDirtBlock())
	planet.SetBlockAt(5, 12, 7, NewGrassBlock())
	// Leave blocks above as air

	height, err := planet.GetSurfaceHeight(5, 7)
	if err != nil {
		t.Errorf("GetSurfaceHeight failed: %v", err)
	}

	// Should return the position above the topmost block (12 + 1 = 13)
	if height != 13 {
		t.Errorf("Expected surface height 13, got %d", height)
	}

	// Test with no solid blocks
	height, err = planet.GetSurfaceHeight(10, 10)
	if err != nil {
		t.Errorf("GetSurfaceHeight failed: %v", err)
	}

	// Should return default height
	if height != 32 {
		t.Errorf("Expected default surface height 32, got %d", height)
	}
}

// TestPlanetSafeSpawnPosition teste le calcul de position de spawn sécurisée
func TestPlanetSafeSpawnPosition(t *testing.T) {
	planet := NewPlanet(
		goecs.Identifier{Namespace: "test", Path: "planet"},
		mgl32.Vec3{0, 0, 0},
		mgl32.Vec3{2, 2, 2},
		ChunkSize{16, 32, 16},
		42,
	)

	registry := createMockRegistry()
	atlas := createMockAtlas()

	err := planet.InitializeChunks(registry, atlas)
	if err != nil {
		t.Errorf("InitializeChunks failed: %v", err)
	}

	// Test before planet is ready
	_, err = planet.GetSafeSpawnPosition()
	if err == nil {
		t.Error("Expected error when planet is not ready")
	}

	// Set planet to ready state
	planet.SetState(PlanetStateReady)

	// Create a safe spawn area
	centerX := int(planet.Size.X()) * planet.ChunkSize.Width / 2
	centerZ := int(planet.Size.Z()) * planet.ChunkSize.Depth / 2

	// Set up ground blocks
	for y := 10; y < 15; y++ {
		planet.SetBlockAt(centerX, y, centerZ, NewStoneBlock())
	}
	planet.SetBlockAt(centerX, 15, centerZ, NewGrassBlock())

	// Get spawn position
	spawnPos, err := planet.GetSafeSpawnPosition()
	if err != nil {
		t.Errorf("GetSafeSpawnPosition failed: %v", err)
	}

	// Should be above the surface
	if spawnPos.Y() <= 15 {
		t.Errorf("Expected spawn position above ground, got Y=%f", spawnPos.Y())
	}

	// Should be around the center
	if spawnPos.X() < float32(centerX-5) || spawnPos.X() > float32(centerX+5) {
		t.Errorf("Expected spawn position around center X=%d, got X=%f", centerX, spawnPos.X())
	}

	if spawnPos.Z() < float32(centerZ-5) || spawnPos.Z() > float32(centerZ+5) {
		t.Errorf("Expected spawn position around center Z=%d, got Z=%f", centerZ, spawnPos.Z())
	}
}

// TestPlanetPopulate teste le peuplement de la planète
func TestPlanetPopulate(t *testing.T) {
	planet := NewPlanet(
		goecs.Identifier{Namespace: "test", Path: "planet"},
		mgl32.Vec3{0, 0, 0},
		mgl32.Vec3{2, 2, 2},
		ChunkSize{16, 32, 16},
		42,
	)

	registry := createMockRegistry()
	atlas := createMockAtlas()

	err := planet.InitializeChunks(registry, atlas)
	if err != nil {
		t.Errorf("InitializeChunks failed: %v", err)
	}

	// Test populate
	planet.Populate()

	if planet.GetState() != PlanetStateReady {
		t.Errorf("Expected state PlanetStateReady after populate, got %v", planet.GetState())
	}

	// Check that blocks were populated
	block, _, err := planet.GetBlockAt(5, 10, 7)
	if err != nil {
		t.Errorf("GetBlockAt failed after populate: %v", err)
	}

	if block == nil {
		t.Error("Expected block to exist after populate")
	}

	// Should have terrain (not all air)
	if block.Type == BlockTypeAir {
		// Check a few more blocks to make sure there's some terrain
		foundTerrain := false
		for y := 0; y < 35; y++ {
			testBlock, _, err := planet.GetBlockAt(5, y, 7)
			if err == nil && testBlock != nil && testBlock.Type != BlockTypeAir {
				foundTerrain = true
				break
			}
		}
		if !foundTerrain {
			t.Error("Expected some terrain blocks after populate")
		}
	}
}

// TestPlanetUpdate teste la mise à jour de la planète
func TestPlanetUpdate(t *testing.T) {
	planet := NewPlanet(
		goecs.Identifier{Namespace: "test", Path: "planet"},
		mgl32.Vec3{0, 0, 0},
		mgl32.Vec3{2, 2, 2},
		ChunkSize{16, 32, 16},
		42,
	)

	registry := createMockRegistry()
	atlas := createMockAtlas()

	// Créer manuellement les chunks pour éviter la population automatique
	planet.BlockRegistry = registry
	planet.TextureAtlas = atlas
	sizeX, sizeY, sizeZ := int(planet.Size.X()), int(planet.Size.Y()), int(planet.Size.Z())
	planet.Chunks = make([][][]*Chunk, sizeX)
	for x := 0; x < sizeX; x++ {
		planet.Chunks[x] = make([][]*Chunk, sizeY)
		for y := 0; y < sizeY; y++ {
			planet.Chunks[x][y] = make([]*Chunk, sizeZ)
			for z := 0; z < sizeZ; z++ {
				offset := mgl32.Vec3{float32(x * planet.ChunkSize.Width), float32(y * planet.ChunkSize.Height), float32(z * planet.ChunkSize.Depth)}
				globalPos := planet.Position.Add(offset)
				planet.Chunks[x][y][z] = NewChunk(globalPos, planet.ChunkSize, planet, registry, atlas)
			}
		}
	}

	// Test update when not ready (should do nothing)
	planet.SetState(PlanetStateInitialized)
	planet.Update(1.0 / 60.0)

	// Set planet to ready pour tester la logique d'update
	planet.SetState(PlanetStateReady)

	// Mettre quelques chunks dans l'état initialized pour simuler le besoin de mise à jour
	chunk00 := planet.Chunks[0][0][0]
	chunk00.SetState(ChunkStateInitialized)
	chunk01 := planet.Chunks[0][0][1]
	chunk01.SetState(ChunkStateInitialized)

	// IMPORTANT: On ne peut pas appeler planet.Update() car ça utilise OpenGL
	// Au lieu de ça, on teste manuellement la logique sans les appels OpenGL

	// Vérifier que les chunks sont accessible
	chunk, err := planet.GetChunk(0, 0, 0)
	if err != nil {
		t.Errorf("GetChunk failed: %v", err)
	}

	if chunk == nil {
		t.Error("Expected chunk to exist")
	}

	// Vérifier que les chunks sont dans l'état attendu pour la mise à jour
	if chunk.GetState() != ChunkStateInitialized {
		t.Errorf("Expected ChunkStateInitialized, got %v", chunk.GetState())
	}

	// Simuler la génération de vertices (sans OpenGL)
	vertices, indices := chunk.GenerateVertices()
	if chunk.GetState() != ChunkStateGenerated {
		t.Errorf("Expected ChunkStateGenerated after vertex generation, got %v", chunk.GetState())
	}

	// Vérifier que les données de vertex sont cohérentes
	if len(vertices) == 0 && len(indices) == 0 {
		// C'est normal pour un chunk vide (tous les blocs sont air)
		t.Log("Chunk is empty (all air blocks), which is expected")
	}
}

// TestPlanetConcurrentAccess teste l'accès concurrent à la planète
func TestPlanetConcurrentAccess(t *testing.T) {
	planet := NewPlanet(
		goecs.Identifier{Namespace: "test", Path: "planet"},
		mgl32.Vec3{0, 0, 0},
		mgl32.Vec3{2, 2, 2},
		ChunkSize{16, 32, 16},
		42,
	)

	registry := createMockRegistry()
	atlas := createMockAtlas()

	// Créer manuellement les chunks pour éviter la population automatique
	planet.BlockRegistry = registry
	planet.TextureAtlas = atlas
	sizeX, sizeY, sizeZ := int(planet.Size.X()), int(planet.Size.Y()), int(planet.Size.Z())
	planet.Chunks = make([][][]*Chunk, sizeX)
	for x := 0; x < sizeX; x++ {
		planet.Chunks[x] = make([][]*Chunk, sizeY)
		for y := 0; y < sizeY; y++ {
			planet.Chunks[x][y] = make([]*Chunk, sizeZ)
			for z := 0; z < sizeZ; z++ {
				offset := mgl32.Vec3{float32(x * planet.ChunkSize.Width), float32(y * planet.ChunkSize.Height), float32(z * planet.ChunkSize.Depth)}
				globalPos := planet.Position.Add(offset)
				planet.Chunks[x][y][z] = NewChunk(globalPos, planet.ChunkSize, planet, registry, atlas)
			}
		}
	}
	planet.SetState(PlanetStateReady)

	// Test concurrent state changes
	done := make(chan bool)

	go func() {
		for i := 0; i < 50; i++ {
			planet.SetState(PlanetStatePopulating)
			planet.SetState(PlanetStateReady)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 50; i++ {
			_ = planet.GetState()
			_ = planet.IsReady()
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// Check that planet is in valid state
	state := planet.GetState()
	if state != PlanetStateReady && state != PlanetStatePopulating {
		t.Errorf("Expected valid state, got %v", state)
	}
}

// TestPlanetBlockCoordinateCalculation teste le calcul des coordonnées de blocs
func TestPlanetBlockCoordinateCalculation(t *testing.T) {
	planet := NewPlanet(
		goecs.Identifier{Namespace: "test", Path: "planet"},
		mgl32.Vec3{0, 0, 0},
		mgl32.Vec3{2, 2, 2},
		ChunkSize{16, 32, 16},
		42,
	)

	registry := createMockRegistry()
	atlas := createMockAtlas()

	// Créer manuellement les chunks pour éviter la population automatique
	planet.BlockRegistry = registry
	planet.TextureAtlas = atlas
	sizeX, sizeY, sizeZ := int(planet.Size.X()), int(planet.Size.Y()), int(planet.Size.Z())
	planet.Chunks = make([][][]*Chunk, sizeX)
	for x := 0; x < sizeX; x++ {
		planet.Chunks[x] = make([][]*Chunk, sizeY)
		for y := 0; y < sizeY; y++ {
			planet.Chunks[x][y] = make([]*Chunk, sizeZ)
			for z := 0; z < sizeZ; z++ {
				offset := mgl32.Vec3{float32(x * planet.ChunkSize.Width), float32(y * planet.ChunkSize.Height), float32(z * planet.ChunkSize.Depth)}
				globalPos := planet.Position.Add(offset)
				planet.Chunks[x][y][z] = NewChunk(globalPos, planet.ChunkSize, planet, registry, atlas)
			}
		}
	}
	planet.SetState(PlanetStateReady)

	// Test various block coordinates
	testCases := []struct {
		x, y, z                                        int
		expectedChunkX, expectedChunkY, expectedChunkZ int
	}{
		{0, 0, 0, 0, 0, 0},
		{15, 31, 15, 0, 0, 0},
		{16, 32, 16, 1, 1, 1},
		{17, 33, 17, 1, 1, 1},
		{31, 63, 31, 1, 1, 1},
	}

	for _, tc := range testCases {
		// Set a unique block
		block := Block{Type: BlockTypeStone}
		err := planet.SetBlockAt(tc.x, tc.y, tc.z, block)
		if err != nil {
			t.Errorf("SetBlockAt(%d, %d, %d) failed: %v", tc.x, tc.y, tc.z, err)
			continue
		}

		// Get the block back
		retrievedBlock, chunk, err := planet.GetBlockAt(tc.x, tc.y, tc.z)
		if err != nil {
			t.Errorf("GetBlockAt(%d, %d, %d) failed: %v", tc.x, tc.y, tc.z, err)
			continue
		}

		// Check block type
		if retrievedBlock.Type != BlockTypeStone {
			t.Errorf("Expected BlockTypeStone at (%d, %d, %d), got %v", tc.x, tc.y, tc.z, retrievedBlock.Type)
		}

		// Check chunk coordinates
		expectedChunkPos := mgl32.Vec3{
			float32(tc.expectedChunkX * planet.ChunkSize.Width),
			float32(tc.expectedChunkY * planet.ChunkSize.Height),
			float32(tc.expectedChunkZ * planet.ChunkSize.Depth),
		}

		if chunk.Position != expectedChunkPos {
			t.Errorf("Block at (%d, %d, %d) expected chunk position %v, got %v",
				tc.x, tc.y, tc.z, expectedChunkPos, chunk.Position)
		}
	}
}

// TestPlanetErrorHandling teste la gestion d'erreurs
func TestPlanetErrorHandling(t *testing.T) {
	planet := NewPlanet(
		goecs.Identifier{Namespace: "test", Path: "planet"},
		mgl32.Vec3{0, 0, 0},
		mgl32.Vec3{2, 2, 2},
		ChunkSize{16, 32, 16},
		42,
	)

	// Test error without initialization
	_, err := planet.GetChunk(0, 0, 0)
	if err == nil {
		t.Error("Expected error when accessing chunk before initialization")
	}

	_, _, err = planet.GetBlockAt(0, 0, 0)
	if err == nil {
		t.Error("Expected error when accessing block before initialization")
	}

	// Test error with invalid coordinates
	registry := createMockRegistry()
	atlas := createMockAtlas()

	// Créer manuellement les chunks pour éviter la population automatique
	planet.BlockRegistry = registry
	planet.TextureAtlas = atlas
	sizeX, sizeY, sizeZ := int(planet.Size.X()), int(planet.Size.Y()), int(planet.Size.Z())
	planet.Chunks = make([][][]*Chunk, sizeX)
	for x := 0; x < sizeX; x++ {
		planet.Chunks[x] = make([][]*Chunk, sizeY)
		for y := 0; y < sizeY; y++ {
			planet.Chunks[x][y] = make([]*Chunk, sizeZ)
			for z := 0; z < sizeZ; z++ {
				offset := mgl32.Vec3{float32(x * planet.ChunkSize.Width), float32(y * planet.ChunkSize.Height), float32(z * planet.ChunkSize.Depth)}
				globalPos := planet.Position.Add(offset)
				planet.Chunks[x][y][z] = NewChunk(globalPos, planet.ChunkSize, planet, registry, atlas)
			}
		}
	}
	planet.SetState(PlanetStateReady)

	// Test invalid chunk coordinates
	_, err = planet.GetChunk(-1, 0, 0)
	if err == nil {
		t.Error("Expected error for negative chunk coordinate")
	}

	_, err = planet.GetChunk(5, 0, 0)
	if err == nil {
		t.Error("Expected error for out-of-bounds chunk coordinate")
	}

	// Test invalid block coordinates
	err = planet.SetBlockAt(-1, 0, 0, NewStoneBlock())
	if err == nil {
		t.Error("Expected error for negative block coordinate")
	}

	// Test error state
	testError := fmt.Errorf("test error")
	planet.SetError(testError)

	if planet.GetState() != PlanetStateError {
		t.Errorf("Expected PlanetStateError, got %v", planet.GetState())
	}

	if planet.GetError() == nil {
		t.Error("Expected error to be set")
	}

	if planet.GetError().Error() != testError.Error() {
		t.Errorf("Expected error message '%s', got '%s'", testError.Error(), planet.GetError().Error())
	}
}
