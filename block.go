package main

import (
	"github.com/go-gl/gl/v4.6-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

// BlockType représente le type de bloc
type BlockType int

const (
	BlockTypeAir BlockType = iota
	BlockTypeDirt
	BlockTypeGrass
	BlockTypeStone
	BlockTypeSand
	BlockTypeWater
	BlockTypeWood
	BlockTypeLeaves
)

// Block représente un bloc dans le monde
type Block struct {
	Type     BlockType
	Position mgl32.Vec3
	Scale    mgl32.Vec3

	// Coordonnées de texture pour chaque face
	TextureCoords struct {
		Top    [2]float32
		Bottom [2]float32
		Left   [2]float32
		Right  [2]float32
		Front  [2]float32
		Back   [2]float32
	}

	// Technicals
	vao *VertexArray
	vbo *VertexBuffer
	ebo *ElementsBuffer
}

var (
	// Vertices d'un cube unitaire centré à l'origine
	cubeVertices = []float32{
		// Positions
		-0.5, -0.5, -0.5,
		0.5, -0.5, -0.5,
		0.5, 0.5, -0.5,
		-0.5, 0.5, -0.5,
		-0.5, -0.5, 0.5,
		0.5, -0.5, 0.5,
		0.5, 0.5, 0.5,
		-0.5, 0.5, 0.5,
	}

	// Indices pour former les faces du cube
	cubeIndices = []uint32{
		0, 1, 2, 2, 3, 0, // Face avant
		1, 5, 6, 6, 2, 1, // Face droite
		5, 4, 7, 7, 6, 5, // Face arrière
		4, 0, 3, 3, 7, 4, // Face gauche
		3, 2, 6, 6, 7, 3, // Face haut
		4, 5, 1, 1, 0, 4, // Face bas
	}

	// Mapping des textures par type de bloc et par face
	BlockTextureMap = map[BlockType]map[string]string{
		BlockTypeGrass: {
			"top":    "grass_block_top.png",
			"bottom": "dirt.png",
			"front":  "grass_block_top.png",
			"back":   "grass_block_top.png",
			"left":   "grass_block_top.png",
			"right":  "grass_block_top.png",
		},
		BlockTypeDirt: {
			"top":    "dirt.png",
			"bottom": "dirt.png",
			"front":  "dirt.png",
			"back":   "dirt.png",
			"left":   "dirt.png",
			"right":  "dirt.png",
		},
		BlockTypeStone: {
			"top":    "blackstone.png",
			"bottom": "blackstone.png",
			"front":  "blackstone.png",
			"back":   "blackstone.png",
			"left":   "blackstone.png",
			"right":  "blackstone.png",
		},
	}
)

var highlightVAO, highlightVBO, highlightEBO uint32

var highlightCubeVertices = []float32{
	0, 0, 0,
	1, 0, 0,
	1, 1, 0,
	0, 1, 0,
	0, 0, 1,
	1, 0, 1,
	1, 1, 1,
	0, 1, 1,
}

var highlightCubeIndices = []uint32{
	0, 1, 1, 2, 2, 3, 3, 0, // bas
	4, 5, 5, 6, 6, 7, 7, 4, // haut
	0, 4, 1, 5, 2, 6, 3, 7, // verticales
}

func InitHighlightCube() {
	gl.GenVertexArrays(1, &highlightVAO)
	gl.GenBuffers(1, &highlightVBO)
	gl.GenBuffers(1, &highlightEBO)

	gl.BindVertexArray(highlightVAO)
	gl.BindBuffer(gl.ARRAY_BUFFER, highlightVBO)
	gl.BufferData(gl.ARRAY_BUFFER, len(highlightCubeVertices)*4, gl.Ptr(highlightCubeVertices), gl.STATIC_DRAW)

	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, highlightEBO)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, len(highlightCubeIndices)*4, gl.Ptr(highlightCubeIndices), gl.STATIC_DRAW)

	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 3*4, gl.PtrOffset(0))

	gl.BindVertexArray(0)
}

func DrawBlockHighlight(x, y, z float32, shaderProgram uint32) {
	gl.UseProgram(shaderProgram)
	colorLoc := gl.GetUniformLocation(shaderProgram, gl.Str("highlightColor\x00"))
	modelLoc := gl.GetUniformLocation(shaderProgram, gl.Str("model\x00"))

	gl.PolygonMode(gl.FRONT_AND_BACK, gl.LINE)
	gl.LineWidth(1.0)
	for i, scale := range []float32{1.12, 1.07, 1.00} {
		var color [3]float32
		switch i {
		case 0:
			color = [3]float32{1, 1, 0.3} // halo large, pâle
		case 1:
			color = [3]float32{1, 1, 0.6} // halo moyen
		case 2:
			color = [3]float32{1, 1, 0} // contour vif
		}
		if colorLoc != -1 {
			gl.Uniform3f(colorLoc, color[0], color[1], color[2])
		}
		model := mgl32.Translate3D(x, y, z).Mul4(mgl32.Scale3D(scale, scale, scale)).Mul4(mgl32.Translate3D((1-scale)/2, (1-scale)/2, (1-scale)/2))
		gl.UniformMatrix4fv(modelLoc, 1, false, &model[0])
		gl.BindVertexArray(highlightVAO)
		gl.DrawElements(gl.LINES, int32(len(highlightCubeIndices)), gl.UNSIGNED_INT, nil)
		gl.BindVertexArray(0)
	}
	gl.PolygonMode(gl.FRONT_AND_BACK, gl.FILL)
	gl.LineWidth(1.0)
}

// GetTextureCoords retourne les coordonnées de texture pour un type de bloc donné
func GetTextureCoords(blockType BlockType) (top, bottom, left, right, front, back [2]float32) {
	switch blockType {
	case BlockTypeDirt:
		return [2]float32{0, 0}, [2]float32{0, 0}, [2]float32{0, 0}, [2]float32{0, 0}, [2]float32{0, 0}, [2]float32{0, 0}
	case BlockTypeGrass:
		return [2]float32{1, 0}, [2]float32{0, 0}, [2]float32{2, 0}, [2]float32{2, 0}, [2]float32{2, 0}, [2]float32{2, 0}
	case BlockTypeStone:
		return [2]float32{3, 0}, [2]float32{3, 0}, [2]float32{3, 0}, [2]float32{3, 0}, [2]float32{3, 0}, [2]float32{3, 0}
	case BlockTypeSand:
		return [2]float32{4, 0}, [2]float32{4, 0}, [2]float32{4, 0}, [2]float32{4, 0}, [2]float32{4, 0}, [2]float32{4, 0}
	case BlockTypeWater:
		return [2]float32{5, 0}, [2]float32{5, 0}, [2]float32{5, 0}, [2]float32{5, 0}, [2]float32{5, 0}, [2]float32{5, 0}
	case BlockTypeWood:
		return [2]float32{6, 0}, [2]float32{6, 0}, [2]float32{7, 0}, [2]float32{7, 0}, [2]float32{7, 0}, [2]float32{7, 0}
	case BlockTypeLeaves:
		return [2]float32{8, 0}, [2]float32{8, 0}, [2]float32{8, 0}, [2]float32{8, 0}, [2]float32{8, 0}, [2]float32{8, 0}
	default:
		return [2]float32{0, 0}, [2]float32{0, 0}, [2]float32{0, 0}, [2]float32{0, 0}, [2]float32{0, 0}, [2]float32{0, 0}
	}
}

func NewBlock(blockType BlockType, position mgl32.Vec3, scale mgl32.Vec3) *Block {
	block := &Block{
		Type:     blockType,
		Position: position,
		Scale:    scale,
	}

	// Obtenir les coordonnées de texture pour ce type de bloc
	block.TextureCoords.Top, block.TextureCoords.Bottom, block.TextureCoords.Left,
		block.TextureCoords.Right, block.TextureCoords.Front, block.TextureCoords.Back = GetTextureCoords(blockType)

	// Initialisation des buffers
	block.vao = NewVertexArray()
	block.vbo = NewVertexBuffer(cubeVertices)
	block.ebo = NewElementsBuffer(cubeIndices)

	// Configuration des attributs
	block.vao.Bind()
	block.vao.LinkAttrib(block.vbo, 0, 3, gl.FLOAT, 3*4, 0)
	block.vao.Unbind()

	return block
}

func (block *Block) Render() {
	if block.Type == BlockTypeAir {
		return
	}

	// Activer le shader (à implémenter)
	// shader.Activate()

	// Calculer la matrice de transformation
	model := mgl32.Ident4()
	model = model.Mul4(mgl32.Translate3D(block.Position.X(), block.Position.Y(), block.Position.Z()))
	model = model.Mul4(mgl32.Scale3D(block.Scale.X(), block.Scale.Y(), block.Scale.Z()))

	// Définir la matrice de transformation dans le shader
	// shader.SetMat4("model", model)

	// Rendu du cube
	block.vao.Bind()
	gl.DrawElements(gl.TRIANGLES, int32(len(cubeIndices)), gl.UNSIGNED_INT, nil)
	block.vao.Unbind()
}

func (block *Block) Delete() {
	block.vao.Delete()
	block.vbo.Delete()
	block.ebo.Delete()
}

// IsTransparent retourne true si le bloc est transparent
func (block *Block) IsTransparent() bool {
	return block.Type == BlockTypeAir || block.Type == BlockTypeWater || block.Type == BlockTypeLeaves
}

// IsSolid retourne true si le bloc est solide
func (block *Block) IsSolid() bool {
	return block.Type != BlockTypeAir
}

// Fonction utilitaire pour obtenir le nom de la texture pour un bloc et une face
func GetBlockFaceTextureName(t BlockType, face string) string {
	if faces, ok := BlockTextureMap[t]; ok {
		if name, ok := faces[face]; ok {
			return name
		}
	}
	return "grass_block_top.png" // fallback explicite
}
