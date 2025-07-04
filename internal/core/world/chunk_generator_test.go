package world

import (
	"testing"

	"github.com/go-gl/mathgl/mgl32"
)

// TestNewChunkGenerator teste la création d'un générateur de chunks
func TestNewChunkGenerator(t *testing.T) {
	config := GenerationConfig{
		Base:        10.0,
		Amplitude:   5.0,
		NoiseScale:  20.0,
		Octaves:     4,
		Persistence: 0.5,
		Lacunarity:  2.0,
	}

	generator := NewChunkGenerator(12345, config)

	if generator == nil {
		t.Fatal("NewChunkGenerator returned nil")
	}

	if generator.config != config {
		t.Errorf("Expected config %v, got %v", config, generator.config)
	}

	if generator.noise == nil {
		t.Error("Expected noise generator to be initialized")
	}
}

// TestChunkGeneratorWithBoundaryFaces teste la génération avec des faces de bordure
func TestChunkGeneratorWithBoundaryFaces(t *testing.T) {
	config := GenerationConfig{
		Base:        5.0,
		Amplitude:   3.0,
		NoiseScale:  10.0,
		Octaves:     3,
		Persistence: 0.4,
		Lacunarity:  1.8,
	}

	generator := NewChunkGenerator(42, config)
	registry := createMockRegistry()
	atlas := createMockAtlas()

	planet := &Planet{
		Size:      mgl32.Vec3{3, 3, 3},
		ChunkSize: ChunkSize{Width: 16, Height: 32, Depth: 16},
		Seed:      42,
	}

	chunk := NewChunk(mgl32.Vec3{0, 0, 0}, ChunkSize{16, 32, 16}, planet, registry, atlas)

	// Test avec face de bordure
	chunk.BoundaryFaces = []WorldFace{WorldFaceTop}

	err := generator.GenerateChunk(chunk)
	if err != nil {
		t.Errorf("GenerateChunk failed: %v", err)
	}

	if chunk.GetState() != ChunkStateGenerated {
		t.Errorf("Expected ChunkStateGenerated, got %v", chunk.GetState())
	}

	// Vérifier que des blocs ont été générés
	hasNonAir := false
	for x := 0; x < chunk.Size.Width; x++ {
		for y := 0; y < chunk.Size.Height; y++ {
			for z := 0; z < chunk.Size.Depth; z++ {
				block, _ := chunk.GetBlock(x, y, z)
				if block.Type != BlockTypeAir {
					hasNonAir = true
					break
				}
			}
			if hasNonAir {
				break
			}
		}
		if hasNonAir {
			break
		}
	}

	if !hasNonAir {
		t.Error("Expected some non-air blocks after generation")
	}
}

// TestChunkGeneratorWithoutBoundaryFaces teste la génération sans faces de bordure
func TestChunkGeneratorWithoutBoundaryFaces(t *testing.T) {
	config := GenerationConfig{
		Base:        5.0,
		Amplitude:   3.0,
		NoiseScale:  10.0,
		Octaves:     3,
		Persistence: 0.4,
		Lacunarity:  1.8,
	}

	generator := NewChunkGenerator(42, config)
	registry := createMockRegistry()
	atlas := createMockAtlas()

	planet := &Planet{
		Size:      mgl32.Vec3{3, 3, 3},
		ChunkSize: ChunkSize{Width: 16, Height: 32, Depth: 16},
		Seed:      42,
	}

	chunk := NewChunk(mgl32.Vec3{16, 32, 16}, ChunkSize{16, 32, 16}, planet, registry, atlas)

	// Pas de faces de bordure (chunk interne)
	chunk.BoundaryFaces = []WorldFace{}

	err := generator.GenerateChunk(chunk)
	if err != nil {
		t.Errorf("GenerateChunk failed: %v", err)
	}

	if chunk.GetState() != ChunkStateGenerated {
		t.Errorf("Expected ChunkStateGenerated, got %v", chunk.GetState())
	}

	// Vérifier que le chunk est rempli de blocs solides
	for x := 0; x < chunk.Size.Width; x++ {
		for y := 0; y < chunk.Size.Height; y++ {
			for z := 0; z < chunk.Size.Depth; z++ {
				block, _ := chunk.GetBlock(x, y, z)
				if block.Type != BlockTypeStone {
					t.Errorf("Expected BlockTypeStone at (%d, %d, %d), got %v", x, y, z, block.Type)
				}
			}
		}
	}
}

// TestChunkGeneratorMultipleFaces teste la génération avec plusieurs faces de bordure
func TestChunkGeneratorMultipleFaces(t *testing.T) {
	config := GenerationConfig{
		Base:        8.0,
		Amplitude:   4.0,
		NoiseScale:  15.0,
		Octaves:     4,
		Persistence: 0.6,
		Lacunarity:  2.2,
	}

	generator := NewChunkGenerator(789, config)
	registry := createMockRegistry()
	atlas := createMockAtlas()

	planet := &Planet{
		Size:      mgl32.Vec3{4, 4, 4},
		ChunkSize: ChunkSize{Width: 16, Height: 32, Depth: 16},
		Seed:      789,
	}

	chunk := NewChunk(mgl32.Vec3{0, 0, 0}, ChunkSize{16, 32, 16}, planet, registry, atlas)

	// Chunk au coin avec plusieurs faces de bordure
	chunk.BoundaryFaces = []WorldFace{WorldFaceTop, WorldFaceLeft, WorldFaceBack}

	err := generator.GenerateChunk(chunk)
	if err != nil {
		t.Errorf("GenerateChunk failed: %v", err)
	}

	if chunk.GetState() != ChunkStateGenerated {
		t.Errorf("Expected ChunkStateGenerated, got %v", chunk.GetState())
	}

	// Vérifier qu'il y a un mélange de types de blocs
	blockTypes := make(map[BlockType]int)
	for x := 0; x < chunk.Size.Width; x++ {
		for y := 0; y < chunk.Size.Height; y++ {
			for z := 0; z < chunk.Size.Depth; z++ {
				block, _ := chunk.GetBlock(x, y, z)
				blockTypes[block.Type]++
			}
		}
	}

	// Devrait avoir de l'air (creusé) et des blocs solides
	if blockTypes[BlockTypeAir] == 0 {
		t.Error("Expected some air blocks after terrain sculpting")
	}

	if blockTypes[BlockTypeStone] == 0 {
		t.Error("Expected some stone blocks")
	}

	// Devrait avoir de l'herbe sur les surfaces
	if blockTypes[BlockTypeGrass] == 0 {
		t.Error("Expected some grass blocks on surfaces")
	}
}

// TestChunkGeneratorErrorCases teste les cas d'erreur
func TestChunkGeneratorErrorCases(t *testing.T) {
	config := GenerationConfig{
		Base:        5.0,
		Amplitude:   3.0,
		NoiseScale:  10.0,
		Octaves:     3,
		Persistence: 0.4,
		Lacunarity:  1.8,
	}

	generator := NewChunkGenerator(42, config)

	// Test avec chunk nil
	err := generator.GenerateChunk(nil)
	if err == nil {
		t.Error("Expected error when chunk is nil")
	}

	// Test avec chunk dans mauvais état
	registry := createMockRegistry()
	atlas := createMockAtlas()
	planet := &Planet{
		Size:      mgl32.Vec3{3, 3, 3},
		ChunkSize: ChunkSize{Width: 16, Height: 32, Depth: 16},
	}

	chunk := NewChunk(mgl32.Vec3{0, 0, 0}, ChunkSize{16, 32, 16}, planet, registry, atlas)
	chunk.SetState(ChunkStateReady) // Mauvais état initial

	err = generator.GenerateChunk(chunk)
	if err == nil {
		t.Error("Expected error when chunk is not in initialized state")
	}
}

// TestChunkGeneratorDifferentConfigs teste différentes configurations
func TestChunkGeneratorDifferentConfigs(t *testing.T) {
	configs := []GenerationConfig{
		{
			Base:        1.0,
			Amplitude:   1.0,
			NoiseScale:  5.0,
			Octaves:     2,
			Persistence: 0.3,
			Lacunarity:  1.5,
		},
		{
			Base:        15.0,
			Amplitude:   8.0,
			NoiseScale:  25.0,
			Octaves:     6,
			Persistence: 0.7,
			Lacunarity:  2.5,
		},
	}

	registry := createMockRegistry()
	atlas := createMockAtlas()

	planet := &Planet{
		Size:      mgl32.Vec3{3, 3, 3},
		ChunkSize: ChunkSize{Width: 16, Height: 32, Depth: 16},
		Seed:      42,
	}

	for i, config := range configs {
		generator := NewChunkGenerator(int64(i+1), config)
		chunk := NewChunk(mgl32.Vec3{0, 0, 0}, ChunkSize{16, 32, 16}, planet, registry, atlas)
		chunk.BoundaryFaces = []WorldFace{WorldFaceTop}

		err := generator.GenerateChunk(chunk)
		if err != nil {
			t.Errorf("GenerateChunk failed for config %d: %v", i, err)
		}

		if chunk.GetState() != ChunkStateGenerated {
			t.Errorf("Expected ChunkStateGenerated for config %d, got %v", i, chunk.GetState())
		}
	}
}

// TestChunkGeneratorTerrainSculpting teste le sculptage du terrain
func TestChunkGeneratorTerrainSculpting(t *testing.T) {
	config := GenerationConfig{
		Base:        5.0,
		Amplitude:   3.0,
		NoiseScale:  10.0,
		Octaves:     3,
		Persistence: 0.4,
		Lacunarity:  1.8,
	}

	generator := NewChunkGenerator(42, config)
	registry := createMockRegistry()
	atlas := createMockAtlas()

	planet := &Planet{
		Size:      mgl32.Vec3{3, 3, 3},
		ChunkSize: ChunkSize{Width: 16, Height: 32, Depth: 16},
		Seed:      42,
	}

	// Test pour chaque face
	faces := []WorldFace{
		WorldFaceTop,
		WorldFaceBottom,
		WorldFaceLeft,
		WorldFaceRight,
		WorldFaceFront,
		WorldFaceBack,
	}

	for _, face := range faces {
		chunk := NewChunk(mgl32.Vec3{0, 0, 0}, ChunkSize{16, 32, 16}, planet, registry, atlas)
		chunk.BoundaryFaces = []WorldFace{face}

		err := generator.GenerateChunk(chunk)
		if err != nil {
			t.Errorf("GenerateChunk failed for face %v: %v", face, err)
		}

		// Vérifier que le terrain a été sculpté (il devrait y avoir de l'air)
		hasAir := false
		for x := 0; x < chunk.Size.Width; x++ {
			for y := 0; y < chunk.Size.Height; y++ {
				for z := 0; z < chunk.Size.Depth; z++ {
					block, _ := chunk.GetBlock(x, y, z)
					if block.Type == BlockTypeAir {
						hasAir = true
						break
					}
				}
				if hasAir {
					break
				}
			}
			if hasAir {
				break
			}
		}

		if !hasAir {
			t.Errorf("Expected terrain sculpting (air blocks) for face %v", face)
		}
	}
}

// TestChunkGeneratorGrassPlacement teste le placement de l'herbe
func TestChunkGeneratorGrassPlacement(t *testing.T) {
	config := GenerationConfig{
		Base:        3.0,
		Amplitude:   2.0,
		NoiseScale:  8.0,
		Octaves:     2,
		Persistence: 0.3,
		Lacunarity:  1.5,
	}

	generator := NewChunkGenerator(123, config)
	registry := createMockRegistry()
	atlas := createMockAtlas()

	planet := &Planet{
		Size:      mgl32.Vec3{3, 3, 3},
		ChunkSize: ChunkSize{Width: 16, Height: 32, Depth: 16},
		Seed:      123,
	}

	chunk := NewChunk(mgl32.Vec3{0, 0, 0}, ChunkSize{16, 32, 16}, planet, registry, atlas)
	chunk.BoundaryFaces = []WorldFace{WorldFaceTop}

	err := generator.GenerateChunk(chunk)
	if err != nil {
		t.Errorf("GenerateChunk failed: %v", err)
	}

	// Vérifier qu'il y a de l'herbe
	hasGrass := false
	hasDirt := false
	for x := 0; x < chunk.Size.Width; x++ {
		for y := 0; y < chunk.Size.Height; y++ {
			for z := 0; z < chunk.Size.Depth; z++ {
				block, _ := chunk.GetBlock(x, y, z)
				if block.Type == BlockTypeGrass {
					hasGrass = true
				}
				if block.Type == BlockTypeDirt {
					hasDirt = true
				}
			}
		}
	}

	if !hasGrass {
		t.Error("Expected grass blocks on surface")
	}

	if !hasDirt {
		t.Error("Expected dirt blocks below grass")
	}
}

// TestChunkGeneratorConsistency teste la cohérence des résultats
func TestChunkGeneratorConsistency(t *testing.T) {
	config := GenerationConfig{
		Base:        5.0,
		Amplitude:   3.0,
		NoiseScale:  10.0,
		Octaves:     3,
		Persistence: 0.4,
		Lacunarity:  1.8,
	}

	seed := int64(42)
	registry := createMockRegistry()
	atlas := createMockAtlas()

	planet := &Planet{
		Size:      mgl32.Vec3{3, 3, 3},
		ChunkSize: ChunkSize{Width: 16, Height: 32, Depth: 16},
		Seed:      seed,
	}

	// Générer le même chunk deux fois avec la même seed
	generator1 := NewChunkGenerator(seed, config)
	chunk1 := NewChunk(mgl32.Vec3{0, 0, 0}, ChunkSize{16, 32, 16}, planet, registry, atlas)
	chunk1.BoundaryFaces = []WorldFace{WorldFaceTop}

	generator2 := NewChunkGenerator(seed, config)
	chunk2 := NewChunk(mgl32.Vec3{0, 0, 0}, ChunkSize{16, 32, 16}, planet, registry, atlas)
	chunk2.BoundaryFaces = []WorldFace{WorldFaceTop}

	err1 := generator1.GenerateChunk(chunk1)
	if err1 != nil {
		t.Errorf("GenerateChunk failed for chunk1: %v", err1)
	}

	err2 := generator2.GenerateChunk(chunk2)
	if err2 != nil {
		t.Errorf("GenerateChunk failed for chunk2: %v", err2)
	}

	// Vérifier que les résultats sont identiques
	for x := 0; x < chunk1.Size.Width; x++ {
		for y := 0; y < chunk1.Size.Height; y++ {
			for z := 0; z < chunk1.Size.Depth; z++ {
				block1, _ := chunk1.GetBlock(x, y, z)
				block2, _ := chunk2.GetBlock(x, y, z)

				if block1.Type != block2.Type {
					t.Errorf("Block mismatch at (%d, %d, %d): chunk1=%v, chunk2=%v",
						x, y, z, block1.Type, block2.Type)
				}
			}
		}
	}
}
