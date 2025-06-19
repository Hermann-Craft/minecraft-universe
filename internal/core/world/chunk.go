package world

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/aquilax/go-perlin"
	"github.com/go-gl/gl/v4.6-core/gl"
	"github.com/go-gl/mathgl/mgl32"
	goecs "github.com/oneforx/go-ecs"

	"github.com/hermann-craft/unicube/internal/core/camera"
	"github.com/hermann-craft/unicube/internal/core/geom"
	"github.com/hermann-craft/unicube/internal/render"
)

// ChunkState représente l'état d'un chunk (principe StateLogic)
type ChunkState int

const (
	ChunkStateUninitialized ChunkState = iota
	ChunkStateInitialized
	ChunkStateGenerating
	ChunkStateGenerated
	ChunkStateMeshBuilding
	ChunkStateReady
	ChunkStateError
)

// Chunk représente un morceau de terrain avec état explicite
type Chunk struct {
	// Identité
	ID            goecs.Identifier
	Position      mgl32.Vec3
	Rotation      mgl32.Quat
	State         ChunkState
	mu            sync.RWMutex
	Blocks        [16][32][16]Block
	Size          ChunkSize
	BoundaryFaces []WorldFace
	Planet        *Planet
	BlockRegistry *BlockRegistry
	TextureAtlas  render.TextureAtlas
	LastUpdate    time.Time
	Error         error
	vao           uint32
	vbo           uint32
	ebo           uint32
	Vertices      []float32
	IndexCount    int32
}

// ChunkSize définit les dimensions d'un chunk
type ChunkSize struct {
	Width  int
	Height int
	Depth  int
}

// NewChunk crée un nouveau chunk
func NewChunk(position mgl32.Vec3, size ChunkSize, planet *Planet, registry *BlockRegistry, atlas render.TextureAtlas) *Chunk {
	return &Chunk{
		ID:            goecs.Identifier{Namespace: "core", Path: "chunk"},
		Position:      position,
		Rotation:      mgl32.Quat{W: 1, V: mgl32.Vec3{0, 0, 0}},
		State:         ChunkStateInitialized,
		Size:          size,
		BoundaryFaces: []WorldFace{},
		Planet:        planet,
		BlockRegistry: registry,
		TextureAtlas:  atlas,
	}
}

// SetState change l'état du chunk de manière thread-safe
func (c *Chunk) SetState(state ChunkState) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.State = state
	c.LastUpdate = time.Now()
}

// GetState retourne l'état actuel du chunk
func (c *Chunk) GetState() ChunkState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.State
}

// IsReady vérifie si le chunk est prêt pour le rendu
func (c *Chunk) IsReady() bool {
	return c.GetState() == ChunkStateReady
}

// IsGenerating vérifie si le chunk est en cours de génération
func (c *Chunk) IsGenerating() bool {
	state := c.GetState()
	return state == ChunkStateGenerating || state == ChunkStateMeshBuilding
}

// SetError définit une erreur sur le chunk
func (c *Chunk) SetError(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Error = err
	c.State = ChunkStateError
}

// GetError retourne l'erreur du chunk
func (c *Chunk) GetError() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Error
}

// GetBlock retourne un bloc aux coordonnées données
func (c *Chunk) GetBlock(x, y, z int) (Block, error) {
	if x < 0 || x >= c.Size.Width ||
		y < 0 || y >= c.Size.Height ||
		z < 0 || z >= c.Size.Depth {
		return Block{}, fmt.Errorf("block coordinates out of bounds: (%d, %d, %d)", x, y, z)
	}
	return c.Blocks[x][y][z], nil
}

// SetBlock définit un bloc aux coordonnées données
func (c *Chunk) SetBlock(x, y, z int, block Block) error {
	if x < 0 || x >= c.Size.Width ||
		y < 0 || y >= c.Size.Height ||
		z < 0 || z >= c.Size.Depth {
		return fmt.Errorf("block coordinates out of bounds: (%d, %d, %d)", x, y, z)
	}
	c.Blocks[x][y][z] = block
	c.SetState(ChunkStateInitialized) // Mark for remeshing
	return nil
}

// GetPosition retourne la position du chunk
func (c *Chunk) GetPosition() mgl32.Vec3 {
	return c.Position
}

// GetBoundingBox retourne la boîte englobante du chunk
func (c *Chunk) GetBoundingBox() geom.BoundingBox {
	return geom.NewBoundingBox(
		c.Position,
		c.Position.Add(mgl32.Vec3{
			float32(c.Size.Width),
			float32(c.Size.Height),
			float32(c.Size.Depth),
		}),
	)
}

// IsEdgeChunk vérifie si le chunk est sur le bord de la planète
func (c *Chunk) IsEdgeChunk() bool {
	if c.Planet == nil {
		return false
	}

	chunkX := int(c.Position.X()) / c.Size.Width
	chunkY := int(c.Position.Y()) / c.Size.Height
	chunkZ := int(c.Position.Z()) / c.Size.Depth

	sizeX := int(c.Planet.Size.X())
	sizeY := int(c.Planet.Size.Y())
	sizeZ := int(c.Planet.Size.Z())

	return chunkX == 0 || chunkX == sizeX-1 ||
		chunkY == 0 || chunkY == sizeY-1 ||
		chunkZ == 0 || chunkZ == sizeZ-1
}

// GenerateBlocks génère les blocs du chunk avec du bruit de Perlin
func (c *Chunk) GenerateBlocks() error {
	if c.GetState() != ChunkStateInitialized {
		return fmt.Errorf("chunk must be in initialized state to generate blocks")
	}

	c.SetState(ChunkStateGenerating)

	// Créer un générateur de bruit avec la seed de la planète
	noise := perlin.NewPerlin(2.0, 2.0, 3, int64(c.Planet.Seed))

	// Générer les blocs
	for x := 0; x < c.Size.Width; x++ {
		for y := 0; y < c.Size.Height; y++ {
			for z := 0; z < c.Size.Depth; z++ {
				// Position globale du bloc
				globalX := c.Position.X() + float32(x)
				globalY := c.Position.Y() + float32(y)
				globalZ := c.Position.Z() + float32(z)

				// Générer la hauteur avec du bruit de Perlin
				height := c.generateHeight(noise, globalX, globalZ)

				// Déterminer le type de bloc basé sur la hauteur
				var blockType BlockType
				if globalY < height-2 {
					blockType = BlockTypeStone
				} else if globalY < height {
					blockType = BlockTypeGrass
				} else if globalY == height {
					blockType = BlockTypeGrass
				} else {
					blockType = BlockTypeAir
				}

				c.Blocks[x][y][z] = Block{Type: blockType}
			}
		}
	}

	return nil
}

// generateHeight génère la hauteur du terrain à une position donnée
func (c *Chunk) generateHeight(noise *perlin.Perlin, x, z float32) float32 {
	// Échelle du bruit
	scale := 0.01
	// Amplitude de la hauteur
	amplitude := 10.0

	// Générer la hauteur avec du bruit de Perlin
	height := float32(noise.Noise2D(float64(x)*scale, float64(z)*scale))
	height = height*float32(amplitude) + 16 // Centrer autour de 16

	return height
}

// Render rend le chunk avec OpenGL
func (c *Chunk) Render(currentTime float64, renderer *render.Renderer, camera *camera.Camera, projection mgl32.Mat4) {
	if c.GetState() != ChunkStateReady {
		return
	}
	if c.vao == 0 || c.IndexCount == 0 {
		return
	}

	shader, ok := renderer.GetShader("basic")
	if !ok {
		log.Println("Shader 'basic' non trouvé")
		return
	}
	shader.Activate()

	c.TextureAtlas.Bind(0)
	shader.SetInt("textureAtlas", 0)

	view := camera.GetViewMatrix()
	model := mgl32.Translate3D(c.Position.X(), c.Position.Y(), c.Position.Z())

	shader.SetMat4("model", model)
	shader.SetMat4("view", view)
	shader.SetMat4("projection", projection)

	// Set lighting uniforms
	shader.SetVec3("lightPos", mgl32.Vec3{32, 100, 32})
	shader.SetVec3("lightColor", mgl32.Vec3{1.0, 1.0, 1.0})
	shader.SetVec3("viewPos", camera.GetPosition())
	shader.SetFloat("ambient", 0.1)
	shader.SetFloat("specularStrength", 0.5)
	shader.SetFloat("shininess", 32.0)

	gl.BindVertexArray(c.vao)
	gl.DrawElements(gl.TRIANGLES, c.IndexCount, gl.UNSIGNED_INT, nil)
	gl.BindVertexArray(0)
}

var faceVertices = map[string][]float32{
	// Corrected vertices with consistent CCW winding for all faces
	"north": {
		1, 0, 0, 0, 0, -1,
		0, 0, 0, 0, 0, -1,
		0, 1, 0, 0, 0, -1,
		1, 1, 0, 0, 0, -1,
	},
	"south": {
		0, 0, 1, 0, 0, 1,
		1, 0, 1, 0, 0, 1,
		1, 1, 1, 0, 0, 1,
		0, 1, 1, 0, 0, 1,
	},
	"west": {
		0, 0, 0, -1, 0, 0,
		0, 0, 1, -1, 0, 0,
		0, 1, 1, -1, 0, 0,
		0, 1, 0, -1, 0, 0,
	},
	"east": {
		1, 0, 1, 1, 0, 0,
		1, 0, 0, 1, 0, 0,
		1, 1, 0, 1, 0, 0,
		1, 1, 1, 1, 0, 0,
	},
	"down": {
		0, 0, 0, 0, -1, 0,
		1, 0, 0, 0, -1, 0,
		1, 0, 1, 0, -1, 0,
		0, 0, 1, 0, -1, 0,
	},
	"up": {
		0, 1, 1, 0, 1, 0,
		1, 1, 1, 0, 1, 0,
		1, 1, 0, 0, 1, 0,
		0, 1, 0, 0, 1, 0,
	},
}

// GenerateVertices génère les vertices du chunk
func (c *Chunk) GenerateVertices() ([]float32, []uint32) {
	c.SetState(ChunkStateGenerating)
	c.Vertices = make([]float32, 0, 1024)
	var indices []uint32

	for x := 0; x < c.Size.Width; x++ {
		for y := 0; y < c.Size.Height; y++ {
			for z := 0; z < c.Size.Depth; z++ {
				block, _ := c.GetBlock(x, y, z)
				if block.Type == BlockTypeAir {
					continue
				}

				model, ok := c.BlockRegistry.GetModelForBlock(block.Type)
				if !ok {
					// This can happen for blocks that have no model, which is fine.
					continue
				}

				for _, element := range model.Elements {
					for faceName, face := range element.Faces {
						if !c.isFaceVisible(x, y, z, faceName) {
							continue
						}

						// Calculate base index before adding vertices
						baseIndex := uint32(len(c.Vertices) / 8)

						// In the new system, face.Texture is the final, resolved texture path.
						texturePath := face.Texture

						// Clean the path to match what the atlas expects (e.g., "stone.png")
						cleanPath := strings.TrimPrefix(texturePath, "minecraft:block/")
						cleanPath = strings.TrimPrefix(cleanPath, "block/")
						if !strings.HasSuffix(cleanPath, ".png") {
							cleanPath += ".png"
						}

						// Get the texture's bounding box in the atlas (e.g., u: 0.125, v: 0.25, width: 0.125, height: 0.125)
						atlasU0, atlasV0, atlasU1, atlasV1 := c.TextureAtlas.GetTextureCoords(cleanPath)
						atlasW := atlasU1 - atlasU0
						atlasH := atlasV1 - atlasV0

						// Get the UV data from the model face, defaulting to the full texture [0, 0, 16, 16]
						faceUV := face.UV
						if faceUV == nil || len(faceUV) != 4 {
							faceUV = []float32{0, 0, 16, 16}
						}

						// Convert pixel coordinates (0-16) to texture coordinates (0-1)
						u_start := faceUV[0] / 16.0
						v_start := faceUV[1] / 16.0
						u_end := faceUV[2] / 16.0
						v_end := faceUV[3] / 16.0

						// Calculate the final UVs by mapping the face's UVs into the texture's area within the atlas
						final_u0 := atlasU0 + u_start*atlasW
						final_v0 := atlasV0 + v_start*atlasH
						final_u1 := atlasU0 + u_end*atlasW
						final_v1 := atlasV0 + v_end*atlasH

						verts, ok := faceVertices[faceName]
						if !ok {
							continue
						}

						// Map these final atlas coordinates to the quad's vertices
						uvMap := []float32{final_u0, final_v1, final_u1, final_v1, final_u1, final_v0, final_u0, final_v0}
						for i := 0; i < 4; i++ {
							c.Vertices = append(c.Vertices, verts[i*6+0]+float32(x), verts[i*6+1]+float32(y), verts[i*6+2]+float32(z))
							c.Vertices = append(c.Vertices, verts[i*6+3], verts[i*6+4], verts[i*6+5])
							c.Vertices = append(c.Vertices, uvMap[i*2], uvMap[i*2+1])
						}

						// Add indices for the quad
						indices = append(indices,
							baseIndex, baseIndex+1, baseIndex+2,
							baseIndex, baseIndex+2, baseIndex+3,
						)
					}
				}
			}
		}
	}
	c.SetState(ChunkStateGenerated)
	return c.Vertices, indices
}

// isFaceVisible vérifie si une face est visible
func (c *Chunk) isFaceVisible(x, y, z int, faceName string) bool {
	nx, ny, nz := x, y, z
	switch faceName {
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

	if nx < 0 || nx >= c.Size.Width || ny < 0 || ny >= c.Size.Height || nz < 0 || nz >= c.Size.Depth {
		neighbor, _, err := c.Planet.GetBlockAt(int(c.Position.X())+nx, int(c.Position.Y())+ny, int(c.Position.Z())+nz)
		return err != nil || neighbor == nil || neighbor.Type == BlockTypeAir
	}

	neighbor, _ := c.GetBlock(nx, ny, nz)
	return neighbor.Type == BlockTypeAir
}

// GenerateMesh génère le mesh OpenGL du chunk
func (c *Chunk) GenerateMesh(vertices []float32, indices []uint32) {
	if c.GetState() != ChunkStateGenerated {
		return // Ne pas reconstruire si les vertices n'ont pas été regénérés
	}
	c.SetState(ChunkStateMeshBuilding)

	// Toujours nettoyer les anciens tampons pour éviter les fuites de mémoire
	if c.vao != 0 {
		gl.DeleteVertexArrays(1, &c.vao)
		gl.DeleteBuffers(1, &c.vbo)
		gl.DeleteBuffers(1, &c.ebo)
		c.vao, c.vbo, c.ebo = 0, 0, 0
	}

	// Si après la reconstruction, il n'y a plus de vertices, le chunk est vide.
	// C'est un état valide, il ne faut juste rien dessiner.
	if len(vertices) == 0 {
		c.IndexCount = 0
		c.SetState(ChunkStateReady)
		return
	}

	// Création et configuration du VAO, VBO, EBO
	gl.GenVertexArrays(1, &c.vao)
	gl.GenBuffers(1, &c.vbo)
	gl.GenBuffers(1, &c.ebo)

	gl.BindVertexArray(c.vao)

	gl.BindBuffer(gl.ARRAY_BUFFER, c.vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*4, gl.Ptr(vertices), gl.STATIC_DRAW)

	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, c.ebo)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, len(indices)*4, gl.Ptr(indices), gl.STATIC_DRAW)

	// Position attribute
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 8*4, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)
	// Normal attribute
	gl.VertexAttribPointer(1, 3, gl.FLOAT, false, 8*4, gl.PtrOffset(3*4))
	gl.EnableVertexAttribArray(1)
	// UV attribute
	gl.VertexAttribPointer(2, 2, gl.FLOAT, false, 8*4, gl.PtrOffset(6*4))
	gl.EnableVertexAttribArray(2)

	gl.BindVertexArray(0)

	c.IndexCount = int32(len(indices))
	c.SetState(ChunkStateReady)
}
