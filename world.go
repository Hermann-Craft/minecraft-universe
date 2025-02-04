package main

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/go-gl/mathgl/mgl32"
)

type World struct {
	IsInit         bool
	CenterPosition mgl32.Vec3
	Projection     mgl32.Mat4
	StartTime      time.Time

	Camera *Camera
	Sun    *Sun
	Moon   *Moon        // Ajout de la lune
	Chunks [][][]*Chunk // Tableau 3D de chunks

	Game *Game
}

func (world *World) Init(game *Game) {
	world.StartTime = time.Now()

	world.CenterPosition = mgl32.Vec3{0, 0, 0}
	world.Projection = mgl32.Perspective(mgl32.DegToRad(45.0), 800.0/600.0, 0.1, 1000.0)

	world.Sun = &Sun{}
	world.Sun.Init()

	world.Moon = &Moon{}
	world.Moon.Init()

	// Définir le rayon (en nombre de chunks) dans chaque direction
	// Définir le rayon (en nombre de chunks) dans chaque direction
	chunkRadius := 5
	// Taille totale dans chaque dimension = 2*chunkRadius + 1
	size := 2*chunkRadius + 1

	// Allocation du tableau 3D de chunks
	world.Chunks = make([][][]*Chunk, size)
	for x := 0; x < size; x++ {
		world.Chunks[x] = make([][]*Chunk, size)
		for y := 0; y < size; y++ {
			world.Chunks[x][y] = make([]*Chunk, size)
		}
	}

	seed := time.Now().UnixNano()
	for cx := -chunkRadius; cx <= chunkRadius; cx++ {
		for cy := -chunkRadius; cy <= chunkRadius; cy++ {
			for cz := -chunkRadius; cz <= chunkRadius; cz++ {
				globalPosCentered := mgl32.Vec3{
					float32(cx * 16),
					float32(cy * 16),
					float32(cz * 16),
				}

				localPosCentered := mgl32.Vec3{
					float32(cx),
					float32(cy),
					float32(cz),
				}

				// Here normalize the cx, cy, cz
				globalPosBottomLeft := mgl32.Vec3{
					float32((cx + chunkRadius) * 16),
					float32((cy + chunkRadius) * 16),
					float32((cz + chunkRadius) * 16),
				}

				localPosBottomLeft := mgl32.Vec3{
					float32(cx + chunkRadius),
					float32(cy + chunkRadius),
					float32(cz + chunkRadius),
				}

				// Déterminer quelles faces de la frontière s'appliquent.
				// Par exemple, si cx == chunkRadius, alors la face Right s'applique, etc.
				boundaryFaces := []WorldFace{}
				if cx <= -chunkRadius {
					boundaryFaces = append(boundaryFaces, WorldFaceLeft)
				} else if cx >= chunkRadius {
					boundaryFaces = append(boundaryFaces, WorldFaceRight)
				}
				// Pour la dimension Y
				if cy <= -chunkRadius {
					boundaryFaces = append(boundaryFaces, WorldFaceBottom)
				} else if cy >= chunkRadius {
					boundaryFaces = append(boundaryFaces, WorldFaceTop)
				}
				// Pour la dimension Z
				if cz <= -chunkRadius {
					boundaryFaces = append(boundaryFaces, WorldFaceBack)
				} else if cz >= chunkRadius {
					boundaryFaces = append(boundaryFaces, WorldFaceFront)
				}

				// Créer le chunk en passant le booléen isBoundary et le slice des faces.
				newChunk := NewChunk(globalPosCentered, localPosCentered, globalPosBottomLeft, localPosBottomLeft, seed, boundaryFaces, world)
				newChunk.Init(world)

				// Placement dans le tableau 3D (décalage de +chunkRadius pour obtenir les indices corrects)
				world.Chunks[cx+chunkRadius][cy+chunkRadius][cz+chunkRadius] = newChunk
			}
		}
	}

	for x := 0; x < len(world.Chunks); x++ {
		for y := 0; y < len(world.Chunks[x]); y++ {
			for z := 0; z < len(world.Chunks[x][y]); z++ {
				chunk := world.Chunks[x][y][z]
				if chunk != nil {
					chunk.GenerateMesh()
				}
			}
		}
	}

	spawnFace := WorldFaceFront
	spawnPosition := world.GetSpawnPosition(spawnFace)

	world.Camera = NewCamera(spawnPosition)
	world.Camera.Init(world, spawnFace)

	world.Game = game
	world.Game.window.SetCursorPosCallback(world.Camera.HandlerCursorPosCallback)

	world.IsInit = true
}

func (world *World) GetChunkAt(pos mgl32.Vec3) *Chunk {
	// Calcul des coordonnées de chunk (entiers) à partir de la position mondiale.
	// On utilise math.Floor pour gérer correctement les positions négatives.
	cx := int(math.Floor(float64(pos.X()) / float64(ChunkSize)))
	cy := int(math.Floor(float64(pos.Y()) / float64(ChunkHeight)))
	cz := int(math.Floor(float64(pos.Z()) / float64(ChunkSize)))

	// Lors de l'initialisation, les chunks ont été placés dans world.Chunks
	// avec des indices décalés de +chunkRadius pour centrer le cube.
	// On suppose que la taille du tableau dans chaque dimension est size = 2*chunkRadius + 1.
	size := len(world.Chunks)
	chunkRadius := (size - 1) / 2

	// Calcul des indices dans le tableau.
	ix := cx + chunkRadius
	iy := cy + chunkRadius
	iz := cz + chunkRadius

	// Vérifier que les indices sont dans les limites du tableau.
	if ix < 0 || ix >= size ||
		iy < 0 || iy >= size ||
		iz < 0 || iz >= size {
		return nil
	}

	return world.Chunks[ix][iy][iz]
}

var cubeFaceNormals = [...]struct {
	face   WorldFace
	normal mgl32.Vec3
}{
	{WorldFaceTop, mgl32.Vec3{0, 1, 0}},
	{WorldFaceBottom, mgl32.Vec3{0, -1, 0}},
	{WorldFaceRight, mgl32.Vec3{1, 0, 0}},
	{WorldFaceLeft, mgl32.Vec3{-1, 0, 0}},
	{WorldFaceFront, mgl32.Vec3{0, 0, 1}},
	{WorldFaceBack, mgl32.Vec3{0, 0, -1}},
}

func (world *World) Update(deltaTime float32) {
	if !world.IsInit {
		return
	}

	world.Camera.Update(world.Game.window, deltaTime)

	world.Sun.Update(world)
	world.Moon.Update(world)
}

func (world *World) Render() {
	// Render all chunk if world.IsInit
	if !world.IsInit {
		return
	}

	world.Sun.Render(world)
	world.Moon.Render(world)

	for x := 0; x < len(world.Chunks); x++ {
		for y := 0; y < len(world.Chunks[x]); y++ {
			for z := 0; z < len(world.Chunks[x][y]); z++ {
				chunk := world.Chunks[x][y][z]
				if chunk != nil {
					chunk.Render(world)
				}
			}
		}
	}

	newFace := world.Camera.GetDynamicCurrentWorldFaceByPosition(float32(len(world.Chunks) * 16))
	playerPosStr := fmt.Sprintf("Player Position: X=%.2f Y=%.2f Z=%.2f Face=%s",
		world.Camera.Position.X(),
		world.Camera.Position.Y(),
		world.Camera.Position.Z(),
		newFace.ToString())

	world.Game.textRenderer.RenderText(playerPosStr, 10, 20, 1.0)

	// Chaîne pour l'orientation du joueur : ici on affiche yaw, pitch et le vecteur Front
	lookFace := world.Camera.GetLookFace()
	playerOrientationStr := fmt.Sprintf("Player Orientation: Yaw=%.2f Pitch=%.2f Front=(%.2f, %.2f, %.2f) Look Face: %s",
		world.Camera.Yaw,
		world.Camera.Pitch,
		world.Camera.Front.X(),
		world.Camera.Front.Y(),
		world.Camera.Front.Z(),
		lookFace.ToString())

	world.Game.textRenderer.RenderText(playerOrientationStr, 10, 50, 1.0)

	world.Game.textRenderer.RenderText(fmt.Sprintf("Camera.FitWorld=%v", world.Camera.FitWorld), 10, 70, 1.0)
}

func (world *World) Clean() {
	if !world.IsInit {
		return
	}

	world.Sun.Clean()
	world.Moon.Clean()

	for x := 0; x < len(world.Chunks); x++ {
		for y := 0; y < len(world.Chunks[x]); y++ {
			for z := 0; z < len(world.Chunks[x][y]); z++ {
				chunk := world.Chunks[x][y][z]
				if chunk != nil {
					chunk.Clean()
				}
			}
		}
	}
}

const (
	WorldFaceTop WorldFace = iota
	WorldFaceBottom
	WorldFaceLeft
	WorldFaceRight
	WorldFaceFront
	WorldFaceBack
)

type WorldFace int

func (world *World) GetRandomFace() WorldFace {
	return WorldFace(rand.Intn(5))
}

func (worldFace WorldFace) ToString() string {
	switch worldFace {
	case WorldFaceTop:
		return "Top"
	case WorldFaceBottom:
		return "Bottom"
	case WorldFaceLeft:
		return "Left"
	case WorldFaceRight:
		return "Right"
	case WorldFaceFront:
		return "Front"
	case WorldFaceBack:
		return "Back"
	default:
		panic("Unknow worldface")
	}
}

func (world *World) GetSpawnPosition(spawnFace WorldFace) (spawnPosition mgl32.Vec3) {
	switch spawnFace {
	case WorldFaceTop:
		spawnPosition = mgl32.Vec3{0, 6 * 16, 0}
	case WorldFaceBottom:
		spawnPosition = mgl32.Vec3{0, -6 * 16, 0}
	case WorldFaceLeft:
		spawnPosition = mgl32.Vec3{-6 * 16, 0, 0}
	case WorldFaceRight:
		spawnPosition = mgl32.Vec3{6 * 16, 0, 0}
	case WorldFaceFront:
		spawnPosition = mgl32.Vec3{0, 0, 6 * 16}
	case WorldFaceBack:
		spawnPosition = mgl32.Vec3{0, 0, -6 * 16}
	default:
		spawnPosition = mgl32.Vec3{0, 6 * 16, 0}
	}
	return spawnPosition
}
