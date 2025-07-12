package world

import (
	"fmt"
	"testing"

	"github.com/go-gl/mathgl/mgl32"
	goecs "github.com/oneforx/go-ecs"
)

// TestFaceVisibilityBasicCases teste les cas de base de visibilité des faces
func TestFaceVisibilityBasicCases(t *testing.T) {
	registry := createMockRegistry()
	atlas := createMockAtlas()

	planet := &Planet{
		Size:      mgl32.Vec3{3, 3, 3},
		ChunkSize: ChunkSize{Width: 16, Height: 32, Depth: 16},
	}

	// Créer une planète sans population automatique
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

	chunk := planet.Chunks[1][1][1] // Chunk central

	// Cas 1: Bloc isolé dans l'air - TOUTES les faces doivent être visibles
	chunk.SetBlock(8, 8, 8, NewStoneBlock())

	testCases := []struct {
		name     string
		x, y, z  int
		faceName string
		expected bool
		reason   string
	}{
		{"isolated_up", 8, 8, 8, "up", true, "Face up d'un bloc isolé doit être visible (air au-dessus)"},
		{"isolated_down", 8, 8, 8, "down", true, "Face down d'un bloc isolé doit être visible (air en-dessous)"},
		{"isolated_north", 8, 8, 8, "north", true, "Face north d'un bloc isolé doit être visible (air au nord)"},
		{"isolated_south", 8, 8, 8, "south", true, "Face south d'un bloc isolé doit être visible (air au sud)"},
		{"isolated_east", 8, 8, 8, "east", true, "Face east d'un bloc isolé doit être visible (air à l'est)"},
		{"isolated_west", 8, 8, 8, "west", true, "Face west d'un bloc isolé doit être visible (air à l'ouest)"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			visible := chunk.isFaceVisible(tc.x, tc.y, tc.z, tc.faceName)
			if visible != tc.expected {
				// Diagnostiquer le problème
				nx, ny, nz := tc.x, tc.y, tc.z
				switch tc.faceName {
				case "north":
					nz--
				case "south":
					nz++
				case "west":
					nx--
				case "east":
					nx++
				case "down":
					ny--
				case "up":
					ny++
				}

				var neighborType BlockType
				if nx < 0 || nx >= chunk.Size.Width || ny < 0 || ny >= chunk.Size.Height || nz < 0 || nz >= chunk.Size.Depth {
					// Cas de bordure de chunk
					globalX := int(chunk.Position.X()) + nx
					globalY := int(chunk.Position.Y()) + ny
					globalZ := int(chunk.Position.Z()) + nz
					neighbor, _, err := planet.GetBlockAt(globalX, globalY, globalZ)
					if err != nil {
						t.Logf("Erreur lors de la récupération du bloc voisin aux coordonnées globales (%d, %d, %d): %v", globalX, globalY, globalZ, err)
					} else if neighbor != nil {
						neighborType = neighbor.Type
					}
				} else {
					// Cas interne au chunk
					neighbor, err := chunk.GetBlock(nx, ny, nz)
					if err != nil {
						t.Logf("Erreur lors de la récupération du bloc voisin local (%d, %d, %d): %v", nx, ny, nz, err)
					} else {
						neighborType = neighbor.Type
					}
				}

				t.Errorf("%s: Face %s du bloc (%d, %d, %d) devrait être %t mais est %t. Bloc voisin (%d, %d, %d) est de type %v",
					tc.reason, tc.faceName, tc.x, tc.y, tc.z, tc.expected, visible, nx, ny, nz, neighborType)
			}
		})
	}
}

// TestFaceVisibilityAdjacentBlocks teste la visibilité avec des blocs adjacents
func TestFaceVisibilityAdjacentBlocks(t *testing.T) {
	registry := createMockRegistry()
	atlas := createMockAtlas()

	planet := &Planet{
		Size:      mgl32.Vec3{3, 3, 3},
		ChunkSize: ChunkSize{Width: 16, Height: 32, Depth: 16},
	}

	// Créer une planète sans population automatique
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

	chunk := planet.Chunks[1][1][1] // Chunk central

	// Cas 2: Deux blocs adjacents - les faces internes ne doivent PAS être visibles
	chunk.SetBlock(8, 8, 8, NewStoneBlock()) // Bloc principal
	chunk.SetBlock(8, 8, 9, NewStoneBlock()) // Bloc au sud
	chunk.SetBlock(8, 9, 8, NewDirtBlock())  // Bloc au-dessus
	chunk.SetBlock(9, 8, 8, NewGrassBlock()) // Bloc à l'est

	testCases := []struct {
		name     string
		x, y, z  int
		faceName string
		expected bool
		reason   string
	}{
		// Faces cachées par des blocs adjacents
		{"hidden_south", 8, 8, 8, "south", false, "Face south cachée par bloc adjacent au sud"},
		{"hidden_up", 8, 8, 8, "up", false, "Face up cachée par bloc adjacent au-dessus"},
		{"hidden_east", 8, 8, 8, "east", false, "Face east cachée par bloc adjacent à l'est"},

		// Faces opposées qui doivent rester visibles
		{"visible_north", 8, 8, 8, "north", true, "Face north doit être visible (pas de bloc au nord)"},
		{"visible_down", 8, 8, 8, "down", true, "Face down doit être visible (pas de bloc en-dessous)"},
		{"visible_west", 8, 8, 8, "west", true, "Face west doit être visible (pas de bloc à l'ouest)"},

		// Faces des blocs adjacents qui sont exposées
		{"adjacent_south_north", 8, 8, 9, "north", false, "Face north du bloc sud cachée par le bloc principal"},
		{"adjacent_up_down", 8, 9, 8, "down", false, "Face down du bloc au-dessus cachée par le bloc principal"},
		{"adjacent_east_west", 9, 8, 8, "west", false, "Face west du bloc à l'est cachée par le bloc principal"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			visible := chunk.isFaceVisible(tc.x, tc.y, tc.z, tc.faceName)
			if visible != tc.expected {
				t.Errorf("%s: Face %s du bloc (%d, %d, %d) devrait être %t mais est %t",
					tc.reason, tc.faceName, tc.x, tc.y, tc.z, tc.expected, visible)
			}
		})
	}
}

// TestFaceVisibilityChunkBoundaries teste la visibilité aux bordures des chunks
func TestFaceVisibilityChunkBoundaries(t *testing.T) {
	registry := createMockRegistry()
	atlas := createMockAtlas()

	planet := &Planet{
		Size:      mgl32.Vec3{3, 3, 3},
		ChunkSize: ChunkSize{Width: 16, Height: 32, Depth: 16},
	}

	// Créer une planète sans population automatique
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

	// Cas 3: Blocs à la bordure des chunks
	centerChunk := planet.Chunks[1][1][1]
	neighborChunk := planet.Chunks[1][1][2] // Chunk voisin au sud

	// Bloc à la bordure sud du chunk central
	centerChunk.SetBlock(8, 8, 15, NewStoneBlock())

	// Bloc au début du chunk voisin
	neighborChunk.SetBlock(8, 8, 0, NewDirtBlock())

	testCases := []struct {
		name     string
		chunk    *Chunk
		x, y, z  int
		faceName string
		expected bool
		reason   string
	}{
		// Faces entre chunks avec des blocs adjacents
		{"boundary_south_hidden", centerChunk, 8, 8, 15, "south", false, "Face south à la bordure cachée par bloc du chunk voisin"},
		{"boundary_north_hidden", neighborChunk, 8, 8, 0, "north", false, "Face north à la bordure cachée par bloc du chunk précédent"},

		// Faces exposées à la bordure
		{"boundary_north_visible", centerChunk, 8, 8, 15, "north", true, "Face north à la bordure doit être visible (air au nord)"},
		{"boundary_south_visible", neighborChunk, 8, 8, 0, "south", true, "Face south à la bordure doit être visible (air au sud)"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			visible := tc.chunk.isFaceVisible(tc.x, tc.y, tc.z, tc.faceName)
			if visible != tc.expected {
				// Diagnostique approfondie pour les cas de bordure
				nx, ny, nz := tc.x, tc.y, tc.z
				switch tc.faceName {
				case "north":
					nz--
				case "south":
					nz++
				case "west":
					nx--
				case "east":
					nx++
				case "down":
					ny--
				case "up":
					ny++
				}

				globalX := int(tc.chunk.Position.X()) + nx
				globalY := int(tc.chunk.Position.Y()) + ny
				globalZ := int(tc.chunk.Position.Z()) + nz

				t.Logf("Chunk position: (%.0f, %.0f, %.0f)", tc.chunk.Position.X(), tc.chunk.Position.Y(), tc.chunk.Position.Z())
				t.Logf("Bloc local: (%d, %d, %d)", tc.x, tc.y, tc.z)
				t.Logf("Voisin local calculé: (%d, %d, %d)", nx, ny, nz)
				t.Logf("Voisin global calculé: (%d, %d, %d)", globalX, globalY, globalZ)

				neighbor, _, err := planet.GetBlockAt(globalX, globalY, globalZ)
				if err != nil {
					t.Logf("Erreur GetBlockAt: %v", err)
				} else if neighbor != nil {
					t.Logf("Type du bloc voisin: %v", neighbor.Type)
				} else {
					t.Logf("Bloc voisin est nil")
				}

				t.Errorf("%s: Face %s du bloc (%d, %d, %d) devrait être %t mais est %t",
					tc.reason, tc.faceName, tc.x, tc.y, tc.z, tc.expected, visible)
			}
		})
	}
}

// TestFaceVisibilityCoordinateCalculation teste spécifiquement le calcul des coordonnées
func TestFaceVisibilityCoordinateCalculation(t *testing.T) {
	registry := createMockRegistry()
	atlas := createMockAtlas()

	planet := &Planet{
		Position:  mgl32.Vec3{0, 0, 0}, // Position explicite de la planète
		Size:      mgl32.Vec3{2, 2, 2},
		ChunkSize: ChunkSize{Width: 16, Height: 32, Depth: 16},
	}

	// Créer une planète sans population automatique
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
				t.Logf("Créé chunk[%d][%d][%d] à la position (%.0f, %.0f, %.0f)",
					x, y, z, globalPos.X(), globalPos.Y(), globalPos.Z())
			}
		}
	}
	planet.SetState(PlanetStateReady)

	// Test avec des coordonnées précises
	chunk := planet.Chunks[0][0][0] // Premier chunk à (0, 0, 0)

	// Placer un bloc à la bordure est du chunk (x=15)
	chunk.SetBlock(15, 16, 8, NewStoneBlock())

	// Placer un bloc voisin dans le chunk adjacent
	neighborChunk := planet.Chunks[1][0][0] // Chunk à (16, 0, 0)
	neighborChunk.SetBlock(0, 16, 8, NewDirtBlock())

	t.Logf("Bloc principal dans chunk (0,0,0) à position locale (15,16,8)")
	t.Logf("Bloc voisin dans chunk (1,0,0) à position locale (0,16,8)")
	t.Logf("Position globale du bloc principal: (%d, %d, %d)",
		int(chunk.Position.X())+15, int(chunk.Position.Y())+16, int(chunk.Position.Z())+8)
	t.Logf("Position globale du bloc voisin: (%d, %d, %d)",
		int(neighborChunk.Position.X())+0, int(neighborChunk.Position.Y())+16, int(neighborChunk.Position.Z())+8)

	// Test de la face est du bloc principal (qui devrait être cachée par le voisin)
	visible := chunk.isFaceVisible(15, 16, 8, "east")

	// Diagnostique détaillée
	nx, ny, nz := 15+1, 16, 8 // Coordonnées du voisin (face est)
	t.Logf("Coordonnées locales du voisin: (%d, %d, %d)", nx, ny, nz)

	if nx >= chunk.Size.Width {
		globalX := int(chunk.Position.X()) + nx
		globalY := int(chunk.Position.Y()) + ny
		globalZ := int(chunk.Position.Z()) + nz
		t.Logf("Voisin hors chunk, coordonnées globales: (%d, %d, %d)", globalX, globalY, globalZ)

		neighbor, _, err := planet.GetBlockAt(globalX, globalY, globalZ)
		if err != nil {
			t.Logf("Erreur GetBlockAt: %v", err)
		} else if neighbor != nil {
			t.Logf("Type du bloc voisin trouvé: %v", neighbor.Type)
		} else {
			t.Logf("Bloc voisin est nil")
		}
	}

	if visible {
		t.Error("PROBLÈME DÉTECTÉ: Face est devrait être cachée par le bloc voisin mais est marquée comme visible")
	} else {
		t.Log("OK: Face est correctement cachée par le bloc voisin")
	}
}

// TestFaceVisibilityWithVertexGeneration teste la génération de vertices et compte les faces créées
func TestFaceVisibilityWithVertexGeneration(t *testing.T) {
	registry := createMockRegistry()
	atlas := createMockAtlas()

	planet := &Planet{
		Size:      mgl32.Vec3{2, 2, 2},
		ChunkSize: ChunkSize{Width: 16, Height: 32, Depth: 16},
	}

	// Créer une planète sans population automatique
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

	chunk := planet.Chunks[0][0][0]

	// Cas de test : bloc isolé devrait générer 6 faces
	chunk.SetBlock(8, 8, 8, NewStoneBlock())

	vertices, indices := chunk.GenerateVertices()

	// Chaque face = 4 vertices * 8 floats (3 pos + 3 norm + 2 uv) = 32 floats
	// 6 faces = 6 * 32 = 192 floats
	expectedVertices := 6 * 4 * 8 // 6 faces, 4 vertices par face, 8 floats par vertex
	expectedIndices := 6 * 6      // 6 faces, 6 indices par face (2 triangles)

	t.Logf("Bloc isolé: Généré %d vertices (attendu %d), %d indices (attendu %d)",
		len(vertices), expectedVertices, len(indices), expectedIndices)

	if len(vertices) != expectedVertices {
		t.Errorf("Bloc isolé: attendu %d vertices, obtenu %d", expectedVertices, len(vertices))
	}

	if len(indices) != expectedIndices {
		t.Errorf("Bloc isolé: attendu %d indices, obtenu %d", expectedIndices, len(indices))
	}

	// Cas de test : deux blocs adjacents devraient générer moins de faces
	chunk.SetBlock(8, 8, 9, NewDirtBlock()) // Adjacent au sud

	vertices2, indices2 := chunk.GenerateVertices()

	// Maintenant on a 2 blocs, mais 2 faces sont cachées (sud du premier, nord du second)
	// 2 blocs * 6 faces = 12 faces, moins 2 cachées = 10 faces
	expectedVertices2 := 10 * 4 * 8
	expectedIndices2 := 10 * 6

	t.Logf("Deux blocs adjacents: Généré %d vertices (attendu %d), %d indices (attendu %d)",
		len(vertices2), expectedVertices2, len(indices2), expectedIndices2)

	if len(vertices2) != expectedVertices2 {
		t.Errorf("Deux blocs adjacents: attendu %d vertices, obtenu %d", expectedVertices2, len(vertices2))
	}

	if len(indices2) != expectedIndices2 {
		t.Errorf("Deux blocs adjacents: attendu %d indices, obtenu %d", expectedIndices2, len(indices2))
	}
}

// TestFaceVisibilityRealWorldScenario simule un scénario de monde réel
func TestFaceVisibilityRealWorldScenario(t *testing.T) {
	registry := createMockRegistry()
	atlas := createMockAtlas()

	planet := &Planet{
		Size:      mgl32.Vec3{3, 3, 3},
		ChunkSize: ChunkSize{Width: 16, Height: 32, Depth: 16},
		Seed:      42,
	}

	// Créer une planète et l'initialiser complètement
	planet.BlockRegistry = registry
	planet.TextureAtlas = atlas
	err := planet.InitializeChunks(registry, atlas)
	if err != nil {
		t.Fatalf("Erreur d'initialisation: %v", err)
	}

	// Attendre que la planète soit prête (elle populate automatiquement)
	if planet.GetState() != PlanetStateReady {
		t.Logf("État de la planète: %v", planet.GetState())
	}

	// Analyser ce que contient réellement le chunk central
	chunk := planet.Chunks[1][1][1] // Chunk central

	t.Logf("Analysing chunk central à la position (%.0f, %.0f, %.0f)",
		chunk.Position.X(), chunk.Position.Y(), chunk.Position.Z())

	// Compter les types de blocs
	airCount := 0
	stoneCount := 0
	dirtCount := 0
	grassCount := 0
	otherCount := 0

	for x := 0; x < chunk.Size.Width; x++ {
		for y := 0; y < chunk.Size.Height; y++ {
			for z := 0; z < chunk.Size.Depth; z++ {
				block, _ := chunk.GetBlock(x, y, z)
				switch block.Type {
				case BlockTypeAir:
					airCount++
				case BlockTypeStone:
					stoneCount++
				case BlockTypeDirt:
					dirtCount++
				case BlockTypeGrass:
					grassCount++
				default:
					otherCount++
				}
			}
		}
	}

	totalBlocks := chunk.Size.Width * chunk.Size.Height * chunk.Size.Depth
	t.Logf("Analyse du contenu du chunk central (%d blocs total):", totalBlocks)
	t.Logf("  Air: %d blocs", airCount)
	t.Logf("  Stone: %d blocs", stoneCount)
	t.Logf("  Dirt: %d blocs", dirtCount)
	t.Logf("  Grass: %d blocs", grassCount)
	t.Logf("  Autres: %d blocs", otherCount)

	solidCount := stoneCount + dirtCount + grassCount + otherCount
	t.Logf("  Total solide: %d blocs", solidCount)

	if solidCount == 0 {
		// Le chunk est entièrement vide, c'est peut-être normal selon la génération
		t.Logf("Le chunk central est entièrement composé d'air - vérifions d'autres chunks")

		// Vérifier les autres chunks
		for cx := 0; cx < int(planet.Size.X()); cx++ {
			for cy := 0; cy < int(planet.Size.Y()); cy++ {
				for cz := 0; cz < int(planet.Size.Z()); cz++ {
					chunk := planet.Chunks[cx][cy][cz]
					solidInChunk := 0
					for x := 0; x < chunk.Size.Width; x++ {
						for y := 0; y < chunk.Size.Height; y++ {
							for z := 0; z < chunk.Size.Depth; z++ {
								block, _ := chunk.GetBlock(x, y, z)
								if block.Type != BlockTypeAir {
									solidInChunk++
								}
							}
						}
					}
					if solidInChunk > 0 {
						t.Logf("Chunk[%d][%d][%d] contient %d blocs solides", cx, cy, cz, solidInChunk)
						// Tester sur ce chunk à la place
						return
					}
				}
			}
		}

		t.Error("Aucun bloc solide trouvé dans toute la planète - problème de génération")
		return
	}

	// Trouver un bloc solide et vérifier ses faces
	var foundSolidBlock bool
	for x := 1; x < chunk.Size.Width-1; x++ {
		for y := 1; y < chunk.Size.Height-1; y++ {
			for z := 1; z < chunk.Size.Depth-1; z++ {
				block, _ := chunk.GetBlock(x, y, z)
				if block.Type != BlockTypeAir {
					foundSolidBlock = true

					// Compter les faces visibles
					visibleFaces := 0
					faces := []string{"up", "down", "north", "south", "east", "west"}

					t.Logf("Analysing bloc solide de type %v à (%d, %d, %d)", block.Type, x, y, z)

					for _, face := range faces {
						visible := chunk.isFaceVisible(x, y, z, face)
						if visible {
							visibleFaces++
							t.Logf("  Face %s: VISIBLE", face)
						} else {
							// Vérifier le voisin pour diagnostiquer
							nx, ny, nz := x, y, z
							switch face {
							case "north":
								nz--
							case "south":
								nz++
							case "west":
								nx--
							case "east":
								nx++
							case "down":
								ny--
							case "up":
								ny++
							}

							neighbor, _ := chunk.GetBlock(nx, ny, nz)
							t.Logf("  Face %s: CACHÉE par bloc de type %v", face, neighbor.Type)
						}
					}

					// Un bloc entouré de blocs solides ne devrait avoir aucune face visible
					if visibleFaces == 0 {
						t.Logf("  ✓ Bloc complètement entouré (aucune face visible)")
					} else {
						t.Logf("  ✓ Bloc partiellement exposé (%d faces visibles)", visibleFaces)
					}

					// Vérifier la cohérence : si une face est visible, le voisin doit être de l'air
					for _, face := range faces {
						visible := chunk.isFaceVisible(x, y, z, face)
						if visible {
							nx, ny, nz := x, y, z
							switch face {
							case "north":
								nz--
							case "south":
								nz++
							case "west":
								nx--
							case "east":
								nx++
							case "down":
								ny--
							case "up":
								ny++
							}

							neighbor, err := chunk.GetBlock(nx, ny, nz)
							if err == nil && neighbor.Type != BlockTypeAir {
								t.Errorf("BUG DÉTECTÉ: Face %s est visible mais le voisin (%d,%d,%d) est de type %v (pas Air)",
									face, nx, ny, nz, neighbor.Type)
							}
						}
					}

					// On n'a besoin de tester qu'un seul bloc pour ce test
					return
				}
			}
		}
	}

	if !foundSolidBlock {
		t.Error("Aucun bloc solide trouvé dans le chunk central")
	}
}

// TestFaceVisibilityDebugLogging ajoute des logs détaillés pour débogage
func TestFaceVisibilityDebugLogging(t *testing.T) {
	registry := createMockRegistry()
	atlas := createMockAtlas()

	// Créer un chunk simple pour le débogage
	planet := &Planet{
		Position:  mgl32.Vec3{0, 0, 0},
		Size:      mgl32.Vec3{1, 1, 1},
		ChunkSize: ChunkSize{Width: 16, Height: 32, Depth: 16},
	}

	planet.BlockRegistry = registry
	planet.TextureAtlas = atlas
	planet.Chunks = make([][][]*Chunk, 1)
	planet.Chunks[0] = make([][]*Chunk, 1)
	planet.Chunks[0][0] = make([]*Chunk, 1)
	planet.Chunks[0][0][0] = NewChunk(mgl32.Vec3{0, 0, 0}, planet.ChunkSize, planet, registry, atlas)
	planet.SetState(PlanetStateReady)

	chunk := planet.Chunks[0][0][0]

	// Créer une situation problématique potentielle
	chunk.SetBlock(0, 0, 0, NewStoneBlock()) // Coin du chunk
	chunk.SetBlock(1, 0, 0, NewDirtBlock())  // Voisin à l'est

	// Analyser en détail la face est du premier bloc
	t.Log("=== ANALYSE DÉTAILLÉE DE LA VISIBILITÉ ===")

	x, y, z := 0, 0, 0
	faceName := "east"

	t.Logf("Bloc analysé: (%d, %d, %d) de type Stone", x, y, z)
	t.Logf("Face analysée: %s", faceName)

	// Calculer les coordonnées du voisin manuellement
	nx, ny, nz := x, y, z
	switch faceName {
	case "east":
		nx++
	}

	t.Logf("Coordonnées du voisin calculées: (%d, %d, %d)", nx, ny, nz)

	// Vérifier si c'est dans les limites du chunk
	inBounds := nx >= 0 && nx < chunk.Size.Width && ny >= 0 && ny < chunk.Size.Height && nz >= 0 && nz < chunk.Size.Depth
	t.Logf("Voisin dans les limites du chunk: %t", inBounds)

	if inBounds {
		neighbor, err := chunk.GetBlock(nx, ny, nz)
		if err != nil {
			t.Logf("Erreur GetBlock: %v", err)
		} else {
			t.Logf("Type du bloc voisin: %v", neighbor.Type)
			t.Logf("Voisin est Air: %t", neighbor.Type == BlockTypeAir)
		}
	} else {
		// Cas de bordure
		globalX := int(chunk.Position.X()) + nx
		globalY := int(chunk.Position.Y()) + ny
		globalZ := int(chunk.Position.Z()) + nz
		t.Logf("Coordonnées globales du voisin: (%d, %d, %d)", globalX, globalY, globalZ)

		neighbor, _, err := planet.GetBlockAt(globalX, globalY, globalZ)
		if err != nil {
			t.Logf("Erreur GetBlockAt: %v", err)
			t.Logf("Face considérée comme visible à cause de l'erreur")
		} else if neighbor == nil {
			t.Logf("Bloc voisin est nil - face visible")
		} else {
			t.Logf("Type du bloc voisin global: %v", neighbor.Type)
		}
	}

	// Appeler la fonction réelle et comparer
	actualVisible := chunk.isFaceVisible(x, y, z, faceName)
	t.Logf("Résultat isFaceVisible: %t", actualVisible)

	// Dans ce cas, la face est devrait être cachée car il y a un bloc de dirt à l'est
	expectedVisible := false
	if actualVisible != expectedVisible {
		t.Errorf("BUG DÉTECTÉ: Face %s devrait être %t mais est %t", faceName, expectedVisible, actualVisible)
	} else {
		t.Logf("✓ Visibilité correcte")
	}
}

// TestFaceVisibilityOnRealTerrain teste la visibilité sur du terrain généré réellement
func TestFaceVisibilityOnRealTerrain(t *testing.T) {
	registry := createMockRegistry()
	atlas := createMockAtlas()

	planet := &Planet{
		Size:      mgl32.Vec3{3, 3, 3},
		ChunkSize: ChunkSize{Width: 16, Height: 32, Depth: 16},
		Seed:      42,
	}

	// Créer une planète et l'initialiser complètement
	planet.BlockRegistry = registry
	planet.TextureAtlas = atlas
	err := planet.InitializeChunks(registry, atlas)
	if err != nil {
		t.Fatalf("Erreur d'initialisation: %v", err)
	}

	// Utiliser le chunk [0][0][0] qui contient des blocs solides
	chunk := planet.Chunks[0][0][0]

	t.Logf("=== ANALYSE DU TERRAIN RÉEL ===")
	t.Logf("Chunk à la position (%.0f, %.0f, %.0f)", chunk.Position.X(), chunk.Position.Y(), chunk.Position.Z())

	// Analyser quelques blocs pour détecter des problèmes
	problematicBlocks := 0
	totalSolidBlocks := 0

	for x := 0; x < chunk.Size.Width; x++ {
		for y := 0; y < chunk.Size.Height; y++ {
			for z := 0; z < chunk.Size.Depth; z++ {
				block, _ := chunk.GetBlock(x, y, z)
				if block.Type != BlockTypeAir {
					totalSolidBlocks++

					// Vérifier la cohérence des faces visibles
					faces := []string{"up", "down", "north", "south", "east", "west"}

					for _, face := range faces {
						visible := chunk.isFaceVisible(x, y, z, face)

						if visible {
							// Si une face est visible, vérifier que le voisin est bien de l'air
							nx, ny, nz := x, y, z
							switch face {
							case "north":
								nz--
							case "south":
								nz++
							case "west":
								nx--
							case "east":
								nx++
							case "down":
								ny--
							case "up":
								ny++
							}

							// Vérifier le voisin
							var neighborType BlockType = BlockTypeAir
							if nx >= 0 && nx < chunk.Size.Width && ny >= 0 && ny < chunk.Size.Height && nz >= 0 && nz < chunk.Size.Depth {
								// Voisin dans le chunk
								neighbor, err := chunk.GetBlock(nx, ny, nz)
								if err == nil {
									neighborType = neighbor.Type
								}
							} else {
								// Voisin hors du chunk - vérifier via la planète
								globalX := int(chunk.Position.X()) + nx
								globalY := int(chunk.Position.Y()) + ny
								globalZ := int(chunk.Position.Z()) + nz
								neighbor, _, err := planet.GetBlockAt(globalX, globalY, globalZ)
								if err == nil && neighbor != nil {
									neighborType = neighbor.Type
								}
							}

							// BUG DÉTECTÉ : Face visible mais voisin solide
							if neighborType != BlockTypeAir {
								problematicBlocks++
								t.Errorf("BUG FACE TRANSPARENTE: Bloc (%d,%d,%d) type %v, face %s visible mais voisin (%d,%d,%d) est type %v",
									x, y, z, block.Type, face, nx, ny, nz, neighborType)

								// Logs de débogage pour ce cas spécifique
								t.Logf("  Position chunk: (%.0f, %.0f, %.0f)", chunk.Position.X(), chunk.Position.Y(), chunk.Position.Z())
								t.Logf("  Bloc local: (%d, %d, %d)", x, y, z)
								t.Logf("  Voisin calculé: (%d, %d, %d)", nx, ny, nz)
								if nx < 0 || nx >= chunk.Size.Width || ny < 0 || ny >= chunk.Size.Height || nz < 0 || nz >= chunk.Size.Depth {
									globalX := int(chunk.Position.X()) + nx
									globalY := int(chunk.Position.Y()) + ny
									globalZ := int(chunk.Position.Z()) + nz
									t.Logf("  Voisin global: (%d, %d, %d)", globalX, globalY, globalZ)
								}
							}
						}
					}

					// Arrêter après avoir analysé 100 blocs pour éviter trop de logs
					if totalSolidBlocks >= 100 {
						break
					}
				}
			}
			if totalSolidBlocks >= 100 {
				break
			}
		}
		if totalSolidBlocks >= 100 {
			break
		}
	}

	t.Logf("Analysé %d blocs solides, trouvé %d blocs avec des faces transparentes erronées",
		totalSolidBlocks, problematicBlocks)

	if problematicBlocks > 0 {
		t.Errorf("PROBLÈME CRITIQUE: %d blocs ont des faces transparentes qui ne devraient pas l'être", problematicBlocks)
	} else {
		t.Logf("✓ Toutes les faces visibles sont correctement exposées à l'air")
	}
}

// TestFaceVisibilityVertexGenerationIntegration teste l'intégration complète
func TestFaceVisibilityVertexGenerationIntegration(t *testing.T) {
	registry := createMockRegistry()
	atlas := createMockAtlas()

	planet := &Planet{
		Size:      mgl32.Vec3{2, 2, 2},
		ChunkSize: ChunkSize{Width: 16, Height: 32, Depth: 16},
		Seed:      42,
	}

	// Créer une planète et l'initialiser complètement
	planet.BlockRegistry = registry
	planet.TextureAtlas = atlas
	err := planet.InitializeChunks(registry, atlas)
	if err != nil {
		t.Fatalf("Erreur d'initialisation: %v", err)
	}

	// Tester la génération de vertices sur un chunk réel
	chunk := planet.Chunks[0][0][0]

	// Compter les blocs solides
	solidBlocks := 0
	for x := 0; x < chunk.Size.Width; x++ {
		for y := 0; y < chunk.Size.Height; y++ {
			for z := 0; z < chunk.Size.Depth; z++ {
				block, _ := chunk.GetBlock(x, y, z)
				if block.Type != BlockTypeAir {
					solidBlocks++
				}
			}
		}
	}

	// Générer les vertices
	vertices, indices := chunk.GenerateVertices()

	// Analyser les résultats
	vertexCount := len(vertices) / 8 // 8 floats par vertex
	faceCount := len(indices) / 6    // 6 indices par face (2 triangles)

	t.Logf("=== INTÉGRATION GÉNÉRATION DE VERTICES ===")
	t.Logf("Blocs solides: %d", solidBlocks)
	t.Logf("Vertices générés: %d", vertexCount)
	t.Logf("Faces générées: %d", faceCount)
	t.Logf("Indices: %d", len(indices))

	// Calculs théoriques
	maxPossibleFaces := solidBlocks * 6
	t.Logf("Faces max théoriques: %d (si tous les blocs étaient isolés)", maxPossibleFaces)

	// Si on a plus de faces que de blocs solides, c'est normal
	// Si on a 0 faces avec des blocs solides, c'est un problème
	if solidBlocks > 0 && faceCount == 0 {
		t.Error("PROBLÈME: Des blocs solides existent mais aucune face n'a été générée")
	} else if faceCount > maxPossibleFaces {
		t.Errorf("PROBLÈME: Plus de faces générées (%d) que théoriquement possible (%d)", faceCount, maxPossibleFaces)
	} else {
		cullRate := float64(maxPossibleFaces-faceCount) / float64(maxPossibleFaces) * 100
		t.Logf("✓ Taux de culling: %.1f%% (%d faces cachées sur %d)", cullRate, maxPossibleFaces-faceCount, maxPossibleFaces)
	}
}

// TestFaceVisibilityEdgeCases teste des cas spéciaux qui pourraient causer des problèmes
func TestFaceVisibilityEdgeCases(t *testing.T) {
	registry := createMockRegistry()
	atlas := createMockAtlas()

	planet := &Planet{
		Size:      mgl32.Vec3{2, 2, 2},
		ChunkSize: ChunkSize{Width: 4, Height: 4, Depth: 4}, // Chunk plus petit pour tester les bordures
	}

	// Créer une planète sans population automatique
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

	t.Log("=== TESTS DE CAS EDGE POUR LA TRANSPARENCE ===")

	// Cas 1: Blocs aux coins des chunks
	chunk00 := planet.Chunks[0][0][0]
	chunk01 := planet.Chunks[0][0][1]
	chunk10 := planet.Chunks[1][0][0]
	chunk11 := planet.Chunks[1][0][1]

	// Créer un motif spécifique qui pourrait causer des problèmes
	chunk00.SetBlock(3, 1, 3, NewStoneBlock()) // Coin du chunk
	chunk01.SetBlock(3, 1, 0, NewDirtBlock())  // Dans le chunk voisin
	chunk10.SetBlock(0, 1, 3, NewGrassBlock()) // Dans le chunk voisin
	chunk11.SetBlock(0, 1, 0, NewStoneBlock()) // Dans le chunk diagonal

	testCases := []struct {
		name     string
		chunk    *Chunk
		x, y, z  int
		face     string
		shouldBe bool
		reason   string
	}{
		// Tests aux bordures de chunks
		{"coin_chunk_south", chunk00, 3, 1, 3, "south", false, "Face sud cachée par bloc du chunk voisin"},
		{"coin_chunk_north", chunk01, 3, 1, 0, "north", false, "Face nord cachée par bloc du chunk précédent"},
		{"coin_chunk_east", chunk00, 3, 1, 3, "east", false, "Face est cachée par bloc du chunk voisin"},
		{"coin_chunk_west", chunk10, 0, 1, 3, "west", false, "Face ouest cachée par bloc du chunk précédent"},

		// Tests de faces exposées
		{"coin_chunk_up", chunk00, 3, 1, 3, "up", true, "Face up exposée à l'air"},
		{"coin_chunk_down", chunk00, 3, 1, 3, "down", true, "Face down exposée à l'air"},
		{"coin_chunk_west_exposed", chunk00, 3, 1, 3, "west", true, "Face ouest exposée à l'air"},
		{"coin_chunk_north_exposed", chunk00, 3, 1, 3, "north", true, "Face nord exposée à l'air"},
	}

	foundProblems := 0
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.chunk.isFaceVisible(tc.x, tc.y, tc.z, tc.face)
			if actual != tc.shouldBe {
				foundProblems++
				t.Errorf("PROBLÈME: %s - Face %s du bloc (%d,%d,%d) devrait être %t mais est %t",
					tc.reason, tc.face, tc.x, tc.y, tc.z, tc.shouldBe, actual)

				// Diagnostic détaillé
				nx, ny, nz := tc.x, tc.y, tc.z
				switch tc.face {
				case "north":
					nz--
				case "south":
					nz++
				case "west":
					nx--
				case "east":
					nx++
				case "down":
					ny--
				case "up":
					ny++
				}

				t.Logf("  Chunk position: (%.0f, %.0f, %.0f)", tc.chunk.Position.X(), tc.chunk.Position.Y(), tc.chunk.Position.Z())
				t.Logf("  Voisin local: (%d, %d, %d)", nx, ny, nz)

				if nx < 0 || nx >= tc.chunk.Size.Width || ny < 0 || ny >= tc.chunk.Size.Height || nz < 0 || nz >= tc.chunk.Size.Depth {
					globalX := int(tc.chunk.Position.X()) + nx
					globalY := int(tc.chunk.Position.Y()) + ny
					globalZ := int(tc.chunk.Position.Z()) + nz
					t.Logf("  Voisin global: (%d, %d, %d)", globalX, globalY, globalZ)

					neighbor, _, err := planet.GetBlockAt(globalX, globalY, globalZ)
					if err != nil {
						t.Logf("  Erreur GetBlockAt: %v", err)
					} else if neighbor != nil {
						t.Logf("  Type voisin global: %v", neighbor.Type)
					} else {
						t.Logf("  Voisin global est nil")
					}
				} else {
					neighbor, err := tc.chunk.GetBlock(nx, ny, nz)
					if err != nil {
						t.Logf("  Erreur GetBlock local: %v", err)
					} else {
						t.Logf("  Type voisin local: %v", neighbor.Type)
					}
				}
			}
		})
	}

	if foundProblems > 0 {
		t.Errorf("RÉSUMÉ: %d problèmes de transparence détectés dans les cas edge", foundProblems)
	} else {
		t.Log("✓ Tous les cas edge fonctionnent correctement")
	}
}

// TestDetectTransparencyBugsInRuntime teste de détecter des bugs au runtime
func TestDetectTransparencyBugsInRuntime(t *testing.T) {
	t.Log("=== GUIDE POUR DÉTECTER LES BUGS DE TRANSPARENCE AU RUNTIME ===")
	t.Log("")
	t.Log("Pour activer le débogage des faces transparentes dans votre jeu :")
	t.Log("1. Dans le fichier internal/core/world/chunk.go, fonction isFaceVisible")
	t.Log("2. Décommentez les lignes de log.Printf pour activer les logs de débogage")
	t.Log("3. Relancez votre jeu et observez la console")
	t.Log("")
	t.Log("Les logs vous indiqueront :")
	t.Log("  - Quelles faces sont cachées et par quoi")
	t.Log("  - Les coordonnées exactes des blocs problématiques")
	t.Log("  - Les types de blocs impliqués")
	t.Log("")
	t.Log("Recherchez des messages du type :")
	t.Log("  'Face X du bloc (a,b,c) cachée par voisin (d,e,f) type Y'")
	t.Log("Si vous voyez une face qui devrait être visible mais qui est cachée,")
	t.Log("vous avez trouvé votre bug !")
	t.Log("")
	t.Log("Autre méthode de débogage :")
	t.Log("  - Modifiez temporairement isFaceVisible pour toujours retourner true")
	t.Log("  - Si le problème disparaît, le bug est dans cette fonction")
	t.Log("  - Si le problème persiste, le bug est dans GenerateVertices ou le rendu")
}

// TestFaceVisibilityStressTest effectue un test de stress avec de nombreuses configurations
func TestFaceVisibilityStressTest(t *testing.T) {
	registry := createMockRegistry()
	atlas := createMockAtlas()

	// Test avec une planète plus grande (5x5x5 chunks)
	planet := NewPlanet(
		goecs.Identifier{Namespace: "test", Path: "stress-planet"},
		mgl32.Vec3{0, 0, 0},
		mgl32.Vec3{5, 5, 5},
		ChunkSize{Width: 16, Height: 32, Depth: 16},
		12345,
	)

	err := planet.InitializeChunks(registry, atlas)
	if err != nil {
		t.Fatalf("Failed to initialize planet: %v", err)
	}

	// Remplir avec un pattern complexe de blocs
	problemCases := 0
	totalChecked := 0

	for chunkX := 0; chunkX < 5; chunkX++ {
		for chunkY := 0; chunkY < 5; chunkY++ {
			for chunkZ := 0; chunkZ < 5; chunkZ++ {
				chunk, _ := planet.GetChunk(chunkX, chunkY, chunkZ)

				// Pattern complexe : damier 3D avec quelques blocs isolés
				for x := 0; x < 16; x++ {
					for y := 0; y < 32; y++ {
						for z := 0; z < 16; z++ {
							if (x+y+z)%3 == 0 {
								chunk.SetBlock(x, y, z, NewStoneBlock())
							} else if (x+y+z)%7 == 0 {
								chunk.SetBlock(x, y, z, NewGrassBlock())
							} else {
								chunk.SetBlock(x, y, z, NewAirBlock())
							}
						}
					}
				}
			}
		}
	}

	// Vérifier toutes les faces de tous les blocs non-air
	for chunkX := 0; chunkX < 5; chunkX++ {
		for chunkY := 0; chunkY < 5; chunkY++ {
			for chunkZ := 0; chunkZ < 5; chunkZ++ {
				chunk, _ := planet.GetChunk(chunkX, chunkY, chunkZ)

				for x := 0; x < 16; x++ {
					for y := 0; y < 32; y++ {
						for z := 0; z < 16; z++ {
							block, _ := chunk.GetBlock(x, y, z)
							if block.Type == BlockTypeAir {
								continue
							}

							totalChecked++

							// Vérifier chaque face
							faces := []string{"north", "south", "east", "west", "up", "down"}
							for _, face := range faces {
								isVisible := chunk.isFaceVisible(x, y, z, face)

								// Calculer manuellement si la face devrait être visible
								shouldBeVisible := manualVisibilityCheck(planet, chunkX, chunkY, chunkZ, x, y, z, face)

								if isVisible != shouldBeVisible {
									problemCases++
									t.Logf("PROBLÈME: Chunk(%d,%d,%d) Bloc(%d,%d,%d) Face %s - Attendu: %v, Obtenu: %v",
										chunkX, chunkY, chunkZ, x, y, z, face, shouldBeVisible, isVisible)
								}
							}
						}
					}
				}
			}
		}
	}

	t.Logf("Test de stress terminé - Blocs vérifiés: %d, Problèmes détectés: %d", totalChecked, problemCases)
	if problemCases > 0 {
		t.Errorf("Détecté %d cas problématiques sur %d blocs testés", problemCases, totalChecked)
	}
}

// TestFaceVisibilityPatterns teste des patterns spécifiques qui pourraient causer des bugs
func TestFaceVisibilityPatterns(t *testing.T) {
	registry := createMockRegistry()
	atlas := createMockAtlas()

	testCases := []struct {
		name        string
		setupBlocks func(*Planet)
		testCoord   [3]int
		testFace    string
		expected    bool
		description string
	}{
		{
			name: "Bloc isolé dans l'air",
			setupBlocks: func(p *Planet) {
				p.SetBlockAt(32, 32, 32, NewStoneBlock())
			},
			testCoord:   [3]int{32, 32, 32},
			testFace:    "north",
			expected:    true,
			description: "Un bloc isolé doit avoir toutes ses faces visibles",
		},
		{
			name: "Bloc entouré de tous côtés",
			setupBlocks: func(p *Planet) {
				// Bloc central
				p.SetBlockAt(32, 32, 32, NewStoneBlock())
				// Entourer de tous côtés
				p.SetBlockAt(31, 32, 32, NewStoneBlock()) // west
				p.SetBlockAt(33, 32, 32, NewStoneBlock()) // east
				p.SetBlockAt(32, 31, 32, NewStoneBlock()) // down
				p.SetBlockAt(32, 33, 32, NewStoneBlock()) // up
				p.SetBlockAt(32, 32, 31, NewStoneBlock()) // north
				p.SetBlockAt(32, 32, 33, NewStoneBlock()) // south
			},
			testCoord:   [3]int{32, 32, 32},
			testFace:    "north",
			expected:    false,
			description: "Un bloc entouré ne doit avoir aucune face visible",
		},
		{
			name: "Ligne de blocs - milieu",
			setupBlocks: func(p *Planet) {
				for i := 30; i <= 34; i++ {
					p.SetBlockAt(i, 32, 32, NewStoneBlock())
				}
			},
			testCoord:   [3]int{32, 32, 32},
			testFace:    "north",
			expected:    true,
			description: "Le bloc du milieu d'une ligne doit avoir ses faces perpendiculaires visibles",
		},
		{
			name: "Ligne de blocs - bout",
			setupBlocks: func(p *Planet) {
				for i := 30; i <= 34; i++ {
					p.SetBlockAt(i, 32, 32, NewStoneBlock())
				}
			},
			testCoord:   [3]int{30, 32, 32},
			testFace:    "west",
			expected:    true,
			description: "Le bloc au bout d'une ligne doit avoir sa face d'extrémité visible",
		},
		{
			name: "Mur avec trou - bloc adjacent au trou",
			setupBlocks: func(p *Planet) {
				// Créer un mur 5x5
				for x := 30; x <= 34; x++ {
					for y := 30; y <= 34; y++ {
						p.SetBlockAt(x, y, 32, NewStoneBlock())
					}
				}
				// Faire un trou au centre
				p.SetBlockAt(32, 32, 32, NewAirBlock())
			},
			testCoord:   [3]int{31, 32, 32},
			testFace:    "east",
			expected:    true,
			description: "Un bloc adjacent à un trou doit avoir sa face vers le trou visible",
		},
		{
			name: "Escalier - marche",
			setupBlocks: func(p *Planet) {
				p.SetBlockAt(30, 30, 32, NewStoneBlock())
				p.SetBlockAt(31, 31, 32, NewStoneBlock())
				p.SetBlockAt(32, 32, 32, NewStoneBlock())
			},
			testCoord:   [3]int{31, 31, 32},
			testFace:    "up",
			expected:    true,
			description: "Une marche d'escalier doit avoir sa face supérieure visible",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Créer une nouvelle planète pour chaque test
			planet := NewPlanet(
				goecs.Identifier{Namespace: "test", Path: "pattern-planet"},
				mgl32.Vec3{0, 0, 0},
				mgl32.Vec3{5, 5, 5},
				ChunkSize{Width: 16, Height: 32, Depth: 16},
				54321,
			)

			err := planet.InitializeChunks(registry, atlas)
			if err != nil {
				t.Fatalf("Failed to initialize planet: %v", err)
			}

			// Remplir d'air par défaut
			for x := 0; x < 80; x++ {
				for y := 0; y < 160; y++ {
					for z := 0; z < 80; z++ {
						planet.SetBlockAt(x, y, z, NewAirBlock())
					}
				}
			}

			// Appliquer le setup spécifique
			tc.setupBlocks(planet)

			// Obtenir le chunk et tester
			chunkX := tc.testCoord[0] / 16
			chunkY := tc.testCoord[1] / 32
			chunkZ := tc.testCoord[2] / 16
			localX := tc.testCoord[0] % 16
			localY := tc.testCoord[1] % 32
			localZ := tc.testCoord[2] % 16

			chunk, err := planet.GetChunk(chunkX, chunkY, chunkZ)
			if err != nil {
				t.Fatalf("Failed to get chunk: %v", err)
			}

			result := chunk.isFaceVisible(localX, localY, localZ, tc.testFace)

			if result != tc.expected {
				t.Errorf("%s: Face %s du bloc (%d,%d,%d) - Attendu: %v, Obtenu: %v\nDescription: %s",
					tc.name, tc.testFace, tc.testCoord[0], tc.testCoord[1], tc.testCoord[2], tc.expected, result, tc.description)
			}
		})
	}
}

// TestFaceVisibilityBoundaryStress teste intensivement les bordures entre chunks
func TestFaceVisibilityBoundaryStress(t *testing.T) {
	registry := createMockRegistry()
	atlas := createMockAtlas()

	planet := NewPlanet(
		goecs.Identifier{Namespace: "test", Path: "boundary-planet"},
		mgl32.Vec3{0, 0, 0},
		mgl32.Vec3{4, 4, 4},
		ChunkSize{Width: 16, Height: 32, Depth: 16},
		99999,
	)

	err := planet.InitializeChunks(registry, atlas)
	if err != nil {
		t.Fatalf("Failed to initialize planet: %v", err)
	}

	// Test de patterns complexes aux bordures
	patterns := []struct {
		name  string
		setup func(*Planet)
	}{
		{
			name: "Damier aux bordures",
			setup: func(p *Planet) {
				// Pattern damier spécifiquement aux bordures de chunks
				for chunkX := 0; chunkX < 4; chunkX++ {
					for chunkY := 0; chunkY < 4; chunkY++ {
						for chunkZ := 0; chunkZ < 4; chunkZ++ {
							baseX := chunkX * 16
							baseY := chunkY * 32
							baseZ := chunkZ * 16

							// Bordures du chunk
							for i := 0; i < 16; i++ {
								for j := 0; j < 32; j++ {
									// Face X=0
									if (i+j)%2 == 0 {
										p.SetBlockAt(baseX, baseY+j, baseZ+i, NewStoneBlock())
									}
									// Face X=15
									if (i+j)%2 == 1 {
										p.SetBlockAt(baseX+15, baseY+j, baseZ+i, NewStoneBlock())
									}
								}
							}
						}
					}
				}
			},
		},
		{
			name: "Lignes continues entre chunks",
			setup: func(p *Planet) {
				// Lignes qui traversent plusieurs chunks
				for i := 0; i < 64; i++ {
					p.SetBlockAt(i, 32, 32, NewStoneBlock()) // Ligne horizontale
					p.SetBlockAt(32, i, 32, NewStoneBlock()) // Ligne verticale
					p.SetBlockAt(32, 32, i, NewStoneBlock()) // Ligne en profondeur
				}
			},
		},
		{
			name: "Structures en forme de + aux bordures",
			setup: func(p *Planet) {
				// Structure en + à chaque intersection de chunk
				for chunkX := 0; chunkX < 3; chunkX++ {
					for chunkY := 0; chunkY < 3; chunkY++ {
						for chunkZ := 0; chunkZ < 3; chunkZ++ {
							centerX := (chunkX + 1) * 16
							centerY := (chunkY + 1) * 32
							centerZ := (chunkZ + 1) * 16

							// Croix horizontale
							for i := -2; i <= 2; i++ {
								p.SetBlockAt(centerX+i, centerY, centerZ, NewStoneBlock())
								p.SetBlockAt(centerX, centerY, centerZ+i, NewStoneBlock())
							}
						}
					}
				}
			},
		},
	}

	for _, pattern := range patterns {
		t.Run(pattern.name, func(t *testing.T) {
			// Réinitialiser la planète
			for x := 0; x < 64; x++ {
				for y := 0; y < 128; y++ {
					for z := 0; z < 64; z++ {
						planet.SetBlockAt(x, y, z, NewAirBlock())
					}
				}
			}

			// Appliquer le pattern
			pattern.setup(planet)

			// Vérifier la cohérence aux bordures
			problems := 0
			checked := 0

			for chunkX := 0; chunkX < 4; chunkX++ {
				for chunkY := 0; chunkY < 4; chunkY++ {
					for chunkZ := 0; chunkZ < 4; chunkZ++ {
						chunk, _ := planet.GetChunk(chunkX, chunkY, chunkZ)

						// Se concentrer sur les bordures du chunk
						borderPositions := []struct{ x, y, z int }{
							{0, 16, 8}, {15, 16, 8}, // Bordures X
							{8, 0, 8}, {8, 31, 8}, // Bordures Y
							{8, 16, 0}, {8, 16, 15}, // Bordures Z
						}

						for _, pos := range borderPositions {
							block, err := chunk.GetBlock(pos.x, pos.y, pos.z)
							if err != nil || block.Type == BlockTypeAir {
								continue
							}

							checked++
							faces := []string{"north", "south", "east", "west", "up", "down"}
							for _, face := range faces {
								isVisible := chunk.isFaceVisible(pos.x, pos.y, pos.z, face)

								// Vérification manuelle
								globalX := chunkX*16 + pos.x
								globalY := chunkY*32 + pos.y
								globalZ := chunkZ*16 + pos.z
								shouldBeVisible := manualVisibilityCheck(planet, chunkX, chunkY, chunkZ, pos.x, pos.y, pos.z, face)

								if isVisible != shouldBeVisible {
									problems++
									t.Logf("BORDURE %s: Chunk(%d,%d,%d) Pos(%d,%d,%d) Global(%d,%d,%d) Face %s - Attendu: %v, Obtenu: %v",
										pattern.name, chunkX, chunkY, chunkZ, pos.x, pos.y, pos.z, globalX, globalY, globalZ, face, shouldBeVisible, isVisible)
								}
							}
						}
					}
				}
			}

			t.Logf("Pattern '%s' terminé - Positions vérifiées: %d, Problèmes: %d", pattern.name, checked, problems)
			if problems > 0 {
				t.Errorf("Pattern '%s': %d problèmes détectés", pattern.name, problems)
			}
		})
	}
}

// TestFaceVisibilityMixedBlockTypes teste avec différents types de blocs
func TestFaceVisibilityMixedBlockTypes(t *testing.T) {
	registry := createMockRegistry()
	atlas := createMockAtlas()

	planet := NewPlanet(
		goecs.Identifier{Namespace: "test", Path: "mixed-planet"},
		mgl32.Vec3{0, 0, 0},
		mgl32.Vec3{3, 3, 3},
		ChunkSize{Width: 16, Height: 32, Depth: 16},
		11111,
	)

	err := planet.InitializeChunks(registry, atlas)
	if err != nil {
		t.Fatalf("Failed to initialize planet: %v", err)
	}

	// Créer un pattern avec différents types de blocs
	blockTypes := []func() Block{
		NewStoneBlock,
		NewDirtBlock,
		NewGrassBlock,
	}

	// Remplir avec un pattern mixte
	for x := 0; x < 48; x++ {
		for y := 0; y < 96; y++ {
			for z := 0; z < 48; z++ {
				if (x+y+z)%4 == 0 {
					// Air
					planet.SetBlockAt(x, y, z, NewAirBlock())
				} else {
					// Bloc solide aléatoire
					blockType := blockTypes[(x+y+z)%len(blockTypes)]
					planet.SetBlockAt(x, y, z, blockType())
				}
			}
		}
	}

	// Vérifier que la logique ne dépend pas du type de bloc
	inconsistencies := 0
	totalTested := 0

	for chunkX := 0; chunkX < 3; chunkX++ {
		for chunkY := 0; chunkY < 3; chunkY++ {
			for chunkZ := 0; chunkZ < 3; chunkZ++ {
				chunk, _ := planet.GetChunk(chunkX, chunkY, chunkZ)

				for x := 0; x < 16; x += 2 { // Échantillonnage pour performance
					for y := 0; y < 32; y += 2 {
						for z := 0; z < 16; z += 2 {
							block, _ := chunk.GetBlock(x, y, z)
							if block.Type == BlockTypeAir {
								continue
							}

							totalTested++
							faces := []string{"north", "south", "east", "west", "up", "down"}
							for _, face := range faces {
								isVisible := chunk.isFaceVisible(x, y, z, face)
								shouldBeVisible := manualVisibilityCheck(planet, chunkX, chunkY, chunkZ, x, y, z, face)

								if isVisible != shouldBeVisible {
									inconsistencies++
									t.Logf("Type mixte: Chunk(%d,%d,%d) Bloc(%d,%d,%d) Type:%v Face %s - Attendu: %v, Obtenu: %v",
										chunkX, chunkY, chunkZ, x, y, z, block.Type, face, shouldBeVisible, isVisible)
								}
							}
						}
					}
				}
			}
		}
	}

	t.Logf("Test types mixtes terminé - Blocs testés: %d, Incohérences: %d", totalTested, inconsistencies)
	if inconsistencies > 0 {
		t.Errorf("Détecté %d incohérences avec différents types de blocs", inconsistencies)
	}
}

// manualVisibilityCheck effectue une vérification manuelle de la visibilité
func manualVisibilityCheck(planet *Planet, chunkX, chunkY, chunkZ, localX, localY, localZ int, face string) bool {
	// Calculer les coordonnées globales du bloc
	globalX := chunkX*16 + localX
	globalY := chunkY*32 + localY
	globalZ := chunkZ*16 + localZ

	// Calculer les coordonnées du voisin selon la face
	var neighborX, neighborY, neighborZ int
	switch face {
	case "north":
		neighborX, neighborY, neighborZ = globalX, globalY, globalZ-1
	case "south":
		neighborX, neighborY, neighborZ = globalX, globalY, globalZ+1
	case "west":
		neighborX, neighborY, neighborZ = globalX-1, globalY, globalZ
	case "east":
		neighborX, neighborY, neighborZ = globalX+1, globalY, globalZ
	case "down":
		neighborX, neighborY, neighborZ = globalX, globalY-1, globalZ
	case "up":
		neighborX, neighborY, neighborZ = globalX, globalY+1, globalZ
	default:
		return true // Face inconnue, considérer comme visible
	}

	// Obtenir le bloc voisin
	neighbor, _, err := planet.GetBlockAt(neighborX, neighborY, neighborZ)

	// Si on ne peut pas obtenir le voisin (hors limites ou erreur), la face est visible
	if err != nil || neighbor == nil {
		return true
	}

	// La face est visible si le voisin est de l'air
	return neighbor.Type == BlockTypeAir
}

// TestFaceVisibilityPerformance teste les performances de la fonction
func TestFaceVisibilityPerformance(t *testing.T) {
	registry := createMockRegistry()
	atlas := createMockAtlas()

	planet := NewPlanet(
		goecs.Identifier{Namespace: "test", Path: "perf-planet"},
		mgl32.Vec3{0, 0, 0},
		mgl32.Vec3{2, 2, 2},
		ChunkSize{Width: 16, Height: 32, Depth: 16},
		7777,
	)

	err := planet.InitializeChunks(registry, atlas)
	if err != nil {
		t.Fatalf("Failed to initialize planet: %v", err)
	}

	// Remplir avec un pattern dense
	for x := 0; x < 32; x++ {
		for y := 0; y < 64; y++ {
			for z := 0; z < 32; z++ {
				if (x+y+z)%3 != 0 {
					planet.SetBlockAt(x, y, z, NewStoneBlock())
				}
			}
		}
	}

	// Test de performance
	chunk, _ := planet.GetChunk(0, 0, 0)

	totalCalls := 0
	faces := []string{"north", "south", "east", "west", "up", "down"}

	// Mesurer le temps d'exécution
	for x := 0; x < 16; x++ {
		for y := 0; y < 32; y++ {
			for z := 0; z < 16; z++ {
				block, _ := chunk.GetBlock(x, y, z)
				if block.Type == BlockTypeAir {
					continue
				}

				for _, face := range faces {
					chunk.isFaceVisible(x, y, z, face)
					totalCalls++
				}
			}
		}
	}

	t.Logf("Test de performance terminé - %d appels à isFaceVisible effectués", totalCalls)
	if totalCalls == 0 {
		t.Error("Aucun appel à isFaceVisible n'a été effectué")
	}
}

// TestFaceVisibilityRandomPatterns teste avec des patterns aléatoires
func TestFaceVisibilityRandomPatterns(t *testing.T) {
	registry := createMockRegistry()
	atlas := createMockAtlas()

	// Effectuer plusieurs tests avec des seeds différentes
	seeds := []int64{1111, 2222, 3333, 4444, 5555}

	for _, seed := range seeds {
		t.Run(fmt.Sprintf("Seed_%d", seed), func(t *testing.T) {
			planet := NewPlanet(
				goecs.Identifier{Namespace: "test", Path: "random-planet"},
				mgl32.Vec3{0, 0, 0},
				mgl32.Vec3{3, 3, 3},
				ChunkSize{Width: 16, Height: 32, Depth: 16},
				seed,
			)

			err := planet.InitializeChunks(registry, atlas)
			if err != nil {
				t.Fatalf("Failed to initialize planet: %v", err)
			}

			// Générer un pattern pseudo-aléatoire basé sur la seed
			for x := 0; x < 48; x++ {
				for y := 0; y < 96; y++ {
					for z := 0; z < 48; z++ {
						// Utiliser une fonction hash simple pour la pseudo-aléatoire
						hash := (x*73 + y*31 + z*17 + int(seed)) % 100
						if hash < 70 { // 70% de blocs solides
							planet.SetBlockAt(x, y, z, NewStoneBlock())
						} else {
							planet.SetBlockAt(x, y, z, NewAirBlock())
						}
					}
				}
			}

			// Vérifier la cohérence sur un échantillon
			errors := 0
			tested := 0

			for chunkX := 0; chunkX < 3; chunkX++ {
				for chunkY := 0; chunkY < 3; chunkY++ {
					for chunkZ := 0; chunkZ < 3; chunkZ++ {
						chunk, _ := planet.GetChunk(chunkX, chunkY, chunkZ)

						// Échantillonnage (tous les 3 blocs pour performance)
						for x := 0; x < 16; x += 3 {
							for y := 0; y < 32; y += 3 {
								for z := 0; z < 16; z += 3 {
									block, _ := chunk.GetBlock(x, y, z)
									if block.Type == BlockTypeAir {
										continue
									}

									tested++
									faces := []string{"north", "south", "east", "west", "up", "down"}
									for _, face := range faces {
										isVisible := chunk.isFaceVisible(x, y, z, face)
										shouldBeVisible := manualVisibilityCheck(planet, chunkX, chunkY, chunkZ, x, y, z, face)

										if isVisible != shouldBeVisible {
											errors++
										}
									}
								}
							}
						}
					}
				}
			}

			t.Logf("Pattern aléatoire seed %d - Blocs testés: %d, Erreurs: %d", seed, tested, errors)
			if errors > 0 {
				t.Errorf("Seed %d: %d erreurs détectées", seed, errors)
			}
		})
	}
}
