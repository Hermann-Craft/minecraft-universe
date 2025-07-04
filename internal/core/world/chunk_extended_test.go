package world

import (
	"testing"

	"github.com/go-gl/mathgl/mgl32"
)

// TestChunkGenerateBlocks teste la génération de blocs
func TestChunkGenerateBlocks(t *testing.T) {
	registry := createMockRegistry()
	atlas := createMockAtlas()

	planet := &Planet{
		Size:      mgl32.Vec3{2, 2, 2},
		ChunkSize: ChunkSize{Width: 16, Height: 32, Depth: 16},
		Seed:      42,
	}

	chunk := NewChunk(mgl32.Vec3{0, 0, 0}, ChunkSize{16, 32, 16}, planet, registry, atlas)

	// Test initial state
	if chunk.GetState() != ChunkStateInitialized {
		t.Errorf("Expected ChunkStateInitialized, got %v", chunk.GetState())
	}

	// Generate blocks
	err := chunk.GenerateBlocks()
	if err != nil {
		t.Errorf("GenerateBlocks failed: %v", err)
	}

	// Check that blocks were generated
	block, err := chunk.GetBlock(0, 0, 0)
	if err != nil {
		t.Errorf("GetBlock failed: %v", err)
	}

	if block.Type == BlockTypeAir {
		t.Error("Expected non-air block after generation")
	}
}

// TestChunkGenerateVertices teste la génération de vertices pour le rendu
func TestChunkGenerateVertices(t *testing.T) {
	registry := createMockRegistry()
	atlas := createMockAtlas()

	planet := &Planet{
		Size:      mgl32.Vec3{2, 2, 2},
		ChunkSize: ChunkSize{Width: 16, Height: 32, Depth: 16},
		Seed:      42,
	}

	// Créer les chunks sans population pour contrôler le contenu
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

	chunk := NewChunk(mgl32.Vec3{0, 0, 0}, ChunkSize{16, 32, 16}, planet, registry, atlas)

	// Add some blocks manually
	chunk.SetBlock(0, 0, 0, NewStoneBlock())
	chunk.SetBlock(1, 0, 0, NewDirtBlock())
	chunk.SetBlock(0, 1, 0, NewGrassBlock())

	// Generate vertices
	vertices, indices := chunk.GenerateVertices()

	// Check that vertices were generated
	if len(vertices) == 0 {
		t.Error("Expected vertices to be generated")
	}

	if len(indices) == 0 {
		t.Error("Expected indices to be generated")
	}

	// Check that state changed
	if chunk.GetState() != ChunkStateGenerated {
		t.Errorf("Expected ChunkStateGenerated, got %v", chunk.GetState())
	}

	// Check vertices format (position + normal + UV = 8 floats per vertex)
	if len(vertices)%8 != 0 {
		t.Errorf("Expected vertices length to be multiple of 8, got %d", len(vertices))
	}

	// Check indices are in groups of 3 (triangles)
	if len(indices)%3 != 0 {
		t.Errorf("Expected indices length to be multiple of 3, got %d", len(indices))
	}
}

// TestChunkIsEdgeChunk teste la détection des chunks de bordure
func TestChunkIsEdgeChunk(t *testing.T) {
	registry := createMockRegistry()
	atlas := createMockAtlas()

	planet := &Planet{
		Size:      mgl32.Vec3{3, 3, 3},
		ChunkSize: ChunkSize{Width: 16, Height: 16, Depth: 16},
	}

	// Test chunk at edge (x=0)
	edgeChunk := NewChunk(mgl32.Vec3{0, 16, 16}, ChunkSize{16, 16, 16}, planet, registry, atlas)
	if !edgeChunk.IsEdgeChunk() {
		t.Error("Expected edge chunk to be detected as edge")
	}

	// Test chunk in center
	centerChunk := NewChunk(mgl32.Vec3{16, 16, 16}, ChunkSize{16, 16, 16}, planet, registry, atlas)
	if centerChunk.IsEdgeChunk() {
		t.Error("Expected center chunk to not be detected as edge")
	}

	// Test chunk without planet
	orphanChunk := NewChunk(mgl32.Vec3{0, 0, 0}, ChunkSize{16, 16, 16}, nil, registry, atlas)
	if orphanChunk.IsEdgeChunk() {
		t.Error("Expected orphan chunk to not be detected as edge")
	}
}

// TestChunkIsFaceVisible teste la visibilité des faces
func TestChunkIsFaceVisible(t *testing.T) {
	registry := createMockRegistry()
	atlas := createMockAtlas()

	planet := &Planet{
		Size:      mgl32.Vec3{2, 2, 2},
		ChunkSize: ChunkSize{Width: 16, Height: 32, Depth: 16},
	}

	// Créer les chunks sans les populer (pour garder les blocs Air par défaut)
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

	chunk := planet.Chunks[0][0][0]

	// Set up some blocks for testing
	chunk.SetBlock(5, 5, 5, NewStoneBlock()) // Center block
	chunk.SetBlock(6, 5, 5, NewStoneBlock()) // Adjacent block to the east
	chunk.SetBlock(5, 6, 5, NewAirBlock())   // Air above (explicitly air)

	// Test face visibility
	// Face next to air should be visible
	if !chunk.isFaceVisible(5, 5, 5, "up") {
		t.Error("Expected face next to air to be visible")
	}

	// Face next to solid block should not be visible
	if chunk.isFaceVisible(5, 5, 5, "east") {
		t.Error("Expected face next to solid block to not be visible")
	}

	// Face at edge should be visible because neighboring chunk has air blocks
	if !chunk.isFaceVisible(0, 0, 0, "west") {
		t.Error("Expected face at edge to be visible when neighbor is air")
	}

	// Test edge case where chunk boundary leads to air
	// Place an air block at the edge position in the neighbor chunk
	neighborChunk := planet.Chunks[0][0][0]         // Same chunk for this test
	neighborChunk.SetBlock(15, 0, 0, NewAirBlock()) // Ensure the neighbor position is air

	if !chunk.isFaceVisible(0, 0, 0, "west") {
		t.Error("Expected face at edge to be visible when neighbor chunk has air")
	}
}

// TestChunkConcurrentAccess teste l'accès concurrent aux chunks
func TestChunkConcurrentAccess(t *testing.T) {
	registry := createMockRegistry()
	atlas := createMockAtlas()

	planet := &Planet{
		ChunkSize: ChunkSize{Width: 16, Height: 32, Depth: 16},
	}

	chunk := NewChunk(mgl32.Vec3{0, 0, 0}, ChunkSize{16, 32, 16}, planet, registry, atlas)

	// Test concurrent state changes
	done := make(chan bool)

	go func() {
		for i := 0; i < 100; i++ {
			chunk.SetState(ChunkStateGenerating)
			chunk.SetState(ChunkStateGenerated)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			_ = chunk.GetState()
			_ = chunk.IsReady()
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// Check that chunk is in valid state
	state := chunk.GetState()
	if state != ChunkStateGenerated && state != ChunkStateGenerating {
		t.Errorf("Expected valid state, got %v", state)
	}
}

// TestChunkBlockModification teste les modifications de blocs
func TestChunkBlockModification(t *testing.T) {
	registry := createMockRegistry()
	atlas := createMockAtlas()

	planet := &Planet{
		ChunkSize: ChunkSize{Width: 16, Height: 32, Depth: 16},
	}

	chunk := NewChunk(mgl32.Vec3{0, 0, 0}, ChunkSize{16, 32, 16}, planet, registry, atlas)

	// Set initial state
	chunk.SetState(ChunkStateReady)

	// Modify a block
	err := chunk.SetBlock(5, 5, 5, NewStoneBlock())
	if err != nil {
		t.Errorf("SetBlock failed: %v", err)
	}

	// Check that state changed to require remeshing
	if chunk.GetState() != ChunkStateInitialized {
		t.Errorf("Expected state to change to ChunkStateInitialized after block modification, got %v", chunk.GetState())
	}

	// Verify block was set
	block, err := chunk.GetBlock(5, 5, 5)
	if err != nil {
		t.Errorf("GetBlock failed: %v", err)
	}

	if block.Type != BlockTypeStone {
		t.Errorf("Expected BlockTypeStone, got %v", block.Type)
	}
}

// TestChunkBoundaryFaces teste la gestion des faces de bordure
func TestChunkBoundaryFaces(t *testing.T) {
	registry := createMockRegistry()
	atlas := createMockAtlas()

	planet := &Planet{
		ChunkSize: ChunkSize{Width: 16, Height: 32, Depth: 16},
	}

	chunk := NewChunk(mgl32.Vec3{0, 0, 0}, ChunkSize{16, 32, 16}, planet, registry, atlas)

	// Initially no boundary faces
	if len(chunk.BoundaryFaces) != 0 {
		t.Errorf("Expected 0 boundary faces initially, got %d", len(chunk.BoundaryFaces))
	}

	// Set boundary faces
	chunk.BoundaryFaces = []WorldFace{WorldFaceTop, WorldFaceBottom}

	if len(chunk.BoundaryFaces) != 2 {
		t.Errorf("Expected 2 boundary faces, got %d", len(chunk.BoundaryFaces))
	}

	// Check specific faces
	expectedFaces := map[WorldFace]bool{
		WorldFaceTop:    true,
		WorldFaceBottom: true,
	}

	for _, face := range chunk.BoundaryFaces {
		if !expectedFaces[face] {
			t.Errorf("Unexpected boundary face: %v", face)
		}
	}
}

// MockChunk pour éviter les appels OpenGL
type MockChunk struct {
	*Chunk
}

func (mc *MockChunk) GenerateMesh(vertices []float32, indices []uint32) {
	// Mock implementation without OpenGL calls
	mc.SetState(ChunkStateMeshBuilding)
	mc.IndexCount = int32(len(indices))
	mc.SetState(ChunkStateReady)
}

// TestChunkVertexGeneration teste la génération de vertices avec différents types de blocs
func TestChunkVertexGeneration(t *testing.T) {
	registry := createMockRegistry()
	atlas := createMockAtlas()

	planet := &Planet{
		Size:      mgl32.Vec3{2, 2, 2},
		ChunkSize: ChunkSize{Width: 16, Height: 32, Depth: 16},
		Seed:      42,
	}

	// Créer une planète simple sans population automatique
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

	chunk := NewChunk(mgl32.Vec3{0, 0, 0}, ChunkSize{16, 32, 16}, planet, registry, atlas)

	// Create a simple pattern with explicit air blocks
	chunk.SetBlock(0, 0, 0, NewStoneBlock())
	chunk.SetBlock(1, 0, 0, NewDirtBlock())
	chunk.SetBlock(0, 1, 0, NewGrassBlock())
	// Ensure surrounding blocks are air for face visibility
	chunk.SetBlock(0, 0, 1, NewAirBlock())
	chunk.SetBlock(1, 0, 1, NewAirBlock())

	// Generate vertices
	vertices, indices := chunk.GenerateVertices()

	// Verify some basic properties
	if len(vertices) == 0 {
		t.Error("Expected vertices to be generated")
	}

	if len(indices) == 0 {
		t.Error("Expected indices to be generated")
	}

	// Store for mesh generation
	chunk.Vertices = vertices

	// Use mock chunk to avoid OpenGL calls
	mockChunk := &MockChunk{Chunk: chunk}
	mockChunk.GenerateMesh(vertices, indices)

	if mockChunk.GetState() != ChunkStateReady {
		t.Errorf("Expected ChunkStateReady after mesh generation, got %v", mockChunk.GetState())
	}

	if mockChunk.IndexCount <= 0 {
		t.Errorf("Expected positive index count, got %d", mockChunk.IndexCount)
	}
}

// TestChunkEmptyMesh teste la génération de mesh vide
func TestChunkEmptyMesh(t *testing.T) {
	registry := createMockRegistry()
	atlas := createMockAtlas()

	planet := &Planet{
		ChunkSize: ChunkSize{Width: 16, Height: 32, Depth: 16},
	}

	chunk := NewChunk(mgl32.Vec3{0, 0, 0}, ChunkSize{16, 32, 16}, planet, registry, atlas)

	// All blocks are air by default
	vertices, indices := chunk.GenerateVertices()

	// Should have no vertices for air blocks
	if len(vertices) != 0 {
		t.Errorf("Expected 0 vertices for air blocks, got %d", len(vertices))
	}

	if len(indices) != 0 {
		t.Errorf("Expected 0 indices for air blocks, got %d", len(indices))
	}

	// Use mock to test mesh generation with empty data
	mockChunk := &MockChunk{Chunk: chunk}
	mockChunk.GenerateMesh(vertices, indices)

	if mockChunk.GetState() != ChunkStateReady {
		t.Errorf("Expected ChunkStateReady even with empty mesh, got %v", mockChunk.GetState())
	}

	if mockChunk.IndexCount != 0 {
		t.Errorf("Expected 0 index count for empty mesh, got %d", mockChunk.IndexCount)
	}
}
