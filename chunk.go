package main

import (
	"fmt"
	"log"
	"sync"

	"math"

	perlin "github.com/aquilax/go-perlin"
	"github.com/go-gl/gl/v4.6-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
	goecs "github.com/oneforx/go-ecs"
)

//////////////////////////////////////////////////////////
// SHADERS : Scène principale (Phong avec lumière locale et lumière du soleil)
//////////////////////////////////////////////////////////

const vertexShaderSource = `
#version 460 core
layout (location = 0) in vec3 aPos;
layout (location = 1) in vec3 aNormal;
layout (location = 2) in vec2 aTexCoord;

out vec3 FragPos;
out vec3 Normal;
out vec2 TexCoord;

uniform mat4 model;
uniform mat4 view;
uniform mat4 projection;

void main() {
	FragPos = vec3(model * vec4(aPos, 1.0));
	Normal = mat3(transpose(inverse(model))) * aNormal;
	TexCoord = aTexCoord;
	gl_Position = projection * view * model * vec4(aPos, 1.0);
}
` + "\x00"

const fragmentShaderSource = `
#version 460 core
out vec4 FragColor;

in vec3 FragPos;
in vec3 Normal;
in vec2 TexCoord;

uniform vec3 lightPos;
uniform vec3 lightColor;
uniform vec3 viewPos;
uniform float ambient;
uniform float specularStrength;
uniform float shininess;

void main() {
	// Ambient
	vec3 ambient = ambient * lightColor;

	// Diffuse
	vec3 norm = normalize(Normal);
	vec3 lightDir = normalize(lightPos - FragPos);
	float diff = max(dot(norm, lightDir), 0.0);
	vec3 diffuse = diff * lightColor;

	// Specular
	vec3 viewDir = normalize(viewPos - FragPos);
	vec3 reflectDir = reflect(-lightDir, norm);
	float spec = pow(max(dot(viewDir, reflectDir), 0.0), shininess);
	vec3 specular = specularStrength * spec * lightColor;

	vec3 result = (ambient + diffuse + specular) * vec3(0.8, 0.8, 0.8); // Couleur de base grise
	FragColor = vec4(result, 1.0);
}
` + "\x00"

const (
	ChunkSize   = 16 // Taille horizontale (X et Z)
	ChunkHeight = 32 // Hauteur (Y) pour test rapide
	BlockSize   = 1.0
)

// Niveaux de détail pour les chunks
const (
	LOD_HIGH   = 0 // Chunk complet
	LOD_MEDIUM = 1 // Chunk simplifié (moins de faces)
	LOD_LOW    = 2 // Chunk très simplifié (cube simple)
)

// Chunk représente un morceau de terrain composé d'un tableau 3D de blocs.
// Pour simplifier, un bloc est soit présent (true) soit absent (false).
type Chunk struct {
	*goecs.Identifier
	Position *mgl32.Vec3
	Rotation *mgl32.Quat
	Planet   *Planet // Référence à la planète parente

	// OpenGL
	vao             uint32
	vbo             uint32
	shader          *Shader
	texture         uint32
	isInit          bool
	isMeshGenerated bool

	// Technicals
	ebo uint32

	// Shader
	ShaderProgram uint32

	// Blocks
	Blocks [ChunkSize][ChunkHeight][ChunkSize]Block

	// Maillage généré : tableau de vertex (chaque vertex : position (3), texCoord (2) et normal (3))
	Vertices []float32

	ID uint32

	// LOD
	CurrentLOD int
	LODMeshes  map[int][]float32 // Maillages pour chaque niveau de détail
	LODVAOs    map[int]uint32    // VAOs pour chaque niveau de détail
	LODVBOs    map[int]uint32    // VBOs pour chaque niveau de détail

	// Cache pour le LOD
	lastLODUpdate        float64
	lodUpdateInterval    float64
	lastFrustumCheck     float64
	frustumCheckInterval float64
	isInFrustum          bool

	// Système de cache et génération asynchrone
	meshGenerationInProgress bool
	meshGenerationDone       chan bool
	meshCache                map[int][]float32 // Cache des maillages générés
	meshCacheMutex           sync.Mutex
	lastMeshGeneration       float64
	meshGenerationInterval   float64

	// Texture
	TextureAtlas *TextureAtlas
}

// Directions servant à vérifier les voisins pour chaque face
var directions = []struct {
	name       string
	dx, dy, dz int
}{
	{"front", 0, 0, 1},   // face avant (+Z)
	{"back", 0, 0, -1},   // face arrière (-Z)
	{"right", 1, 0, 0},   // face droite (+X)
	{"left", -1, 0, 0},   // face gauche (-X)
	{"top", 0, 1, 0},     // face du haut (+Y)
	{"bottom", 0, -1, 0}, // face du bas (-Y)
}

// Chaque face est définie par 6 vertices (deux triangles).
// Pour chaque vertex, on stocke : position (x, y, z), coordonnée de texture (u, v) et normale (nx, ny, nz).
// Les positions sont définies dans l'espace local d'un bloc (cube d'unités) situé en (0,0,0) à (1,1,1).
var faceVertices = map[string][]float32{
	"front": {
		0, 0, 1, 0, 0, 0, 0, 1,
		1, 0, 1, 1, 0, 0, 0, 1,
		1, 1, 1, 1, 1, 0, 0, 1,

		1, 1, 1, 1, 1, 0, 0, 1,
		0, 1, 1, 0, 1, 0, 0, 1,
		0, 0, 1, 0, 0, 0, 0, 1,
	},
	"back": {
		1, 0, 0, 0, 0, 0, 0, -1,
		0, 0, 0, 1, 0, 0, 0, -1,
		0, 1, 0, 1, 1, 0, 0, -1,

		0, 1, 0, 1, 1, 0, 0, -1,
		1, 1, 0, 0, 1, 0, 0, -1,
		1, 0, 0, 0, 0, 0, 0, -1,
	},
	"right": {
		1, 0, 1, 0, 0, 1, 0, 0,
		1, 0, 0, 1, 0, 1, 0, 0,
		1, 1, 0, 1, 1, 1, 0, 0,

		1, 1, 0, 1, 1, 1, 0, 0,
		1, 1, 1, 0, 1, 1, 0, 0,
		1, 0, 1, 0, 0, 1, 0, 0,
	},
	"left": {
		0, 0, 0, 0, 0, -1, 0, 0,
		0, 0, 1, 1, 0, -1, 0, 0,
		0, 1, 1, 1, 1, -1, 0, 0,

		0, 1, 1, 1, 1, -1, 0, 0,
		0, 1, 0, 0, 1, -1, 0, 0,
		0, 0, 0, 0, 0, -1, 0, 0,
	},
	"top": {
		0, 1, 1, 0, 0, 0, 1, 0,
		1, 1, 1, 1, 0, 0, 1, 0,
		1, 1, 0, 1, 1, 0, 1, 0,

		1, 1, 0, 1, 1, 0, 1, 0,
		0, 1, 0, 0, 1, 0, 1, 0,
		0, 1, 1, 0, 0, 0, 1, 0,
	},
	"bottom": {
		0, 0, 0, 0, 0, 0, -1, 0,
		1, 0, 0, 1, 0, 0, -1, 0,
		1, 0, 1, 1, 1, 0, -1, 0,

		1, 0, 1, 1, 1, 0, -1, 0,
		0, 0, 1, 0, 1, 0, -1, 0,
		0, 0, 0, 0, 0, 0, -1, 0,
	},
}

func fractalNoise2D(p *perlin.Perlin, x, z float64, octaves int, persistence, lacunarity float64) float64 {
	total := 0.0
	frequency := 1.0
	amplitude := 1.0
	maxValue := 0.0 // Pour normaliser le résultat.
	for i := 0; i < octaves; i++ {
		total += p.Noise2D(x*frequency, z*frequency) * amplitude
		maxValue += amplitude
		amplitude *= persistence
		frequency *= lacunarity
	}
	// Noise initial est dans [-1, 1]. On le normalise en [0, 1].
	noise := total / maxValue
	normalized := (noise + 1) * 0.5

	// Appliquer une transformation exponentielle
	// Une puissance supérieure à 1 (ici 2.0) tend à aplatir les hautes valeurs, réduisant ainsi les pics.
	flat := math.Pow(normalized, 2.0)

	// Appliquer une fonction smootherstep pour lisser encore plus la courbe
	// smootherstep(t) = t^3 * (t * (t * 6 - 15) + 10)
	t := flat
	smoother := t * t * t * (t*(t*6-15) + 10)

	// Remapper le résultat en [-1, 1]
	return smoother*2 - 1
}

// NewChunk crée un nouveau chunk à la position spécifiée.
// Retourne une erreur si l'initialisation échoue.
func NewChunk(globalPosCentered mgl32.Vec3, seed int64) (*Chunk, error) {
	chunk := &Chunk{
		Identifier: &goecs.Identifier{Namespace: "core", Path: "chunk"},
		Position:   &globalPosCentered,
		Rotation:   &mgl32.Quat{W: 1, V: mgl32.Vec3{0, 0, 0}},
	}

	// Charger le shader
	shader, err := LoadShader("default")
	if err != nil {
		return nil, fmt.Errorf("failed to create chunk: %w", err)
	}
	chunk.shader = shader

	// Générer les blocs
	chunk.GenerateBlocks()

	// Ne pas générer le mesh ici, il sera généré par les workers
	chunk.isInit = true
	return chunk, nil
}

func simpleThreshold(p *perlin.Perlin, x, y, z, base, amplitude, scale float64) int {
	// Décalage pour que x, y, z soient dans [0, scale] si initialement ils sont dans [-scale/2, scale/2]

	noiseVal := p.Noise3D(x/scale, y/scale, z/scale)
	// Normalisation en [0, 1]
	return int(math.Round(base + noiseVal*amplitude))
}

// compileShader compile un shader à partir de son code source
func compileShader(source string, shaderType uint32) (uint32, error) {
	shader := gl.CreateShader(shaderType)
	if shader == 0 {
		return 0, fmt.Errorf("failed to create shader")
	}

	// Convertir le code source en C string
	csource, free := gl.Strs(source)
	defer free()

	// Définir le code source du shader
	gl.ShaderSource(shader, 1, csource, nil)
	gl.CompileShader(shader)

	// Vérifier les erreurs de compilation
	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)
		log := make([]byte, logLength)
		gl.GetShaderInfoLog(shader, logLength, nil, &log[0])
		gl.DeleteShader(shader)
		return 0, fmt.Errorf("failed to compile shader: %v", string(log))
	}

	return shader, nil
}

func (c *Chunk) Init(identifier goecs.Identifier, position mgl32.Vec3, rotation mgl32.Quat) error {
	c.Identifier = &identifier
	c.Position = &position
	c.Rotation = &rotation

	// Charger le shader seulement s'il n'est pas déjà chargé
	if c.shader == nil {
		var err error
		c.shader, err = LoadShader("default")
		if err != nil {
			log.Printf("Failed to load shader: %v", err)
			return fmt.Errorf("failed to load shader: %v", err)
		}
	}

	// Charger la texture
	gl.GenTextures(1, &c.texture)
	gl.BindTexture(gl.TEXTURE_2D, c.texture)

	// Paramètres de la texture
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR_MIPMAP_LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)

	// Pour l'instant, on utilise une texture de test (carré blanc)
	pixels := []uint8{
		255, 255, 255, 255,
		255, 255, 255, 255,
		255, 255, 255, 255,
		255, 255, 255, 255,
	}
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, 2, 2, 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(pixels))
	gl.GenerateMipmap(gl.TEXTURE_2D)

	// Générer les blocs
	c.GenerateBlocks()

	// Générer le mesh
	c.GenerateMesh()

	// Initialiser les buffers OpenGL
	c.setupMesh()

	c.isInit = true
	return nil
}

func (c *Chunk) isEdgeChunk() bool {
	if c.Planet == nil {
		return false // Par défaut, pas d'info
	}
	chunkX := int(c.Position.X()) / ChunkSize
	chunkY := int(c.Position.Y()) / ChunkSize
	chunkZ := int(c.Position.Z()) / ChunkSize
	sizeX := int(c.Planet.Size.X())
	sizeY := int(c.Planet.Size.Y())
	sizeZ := int(c.Planet.Size.Z())
	return chunkX == 0 || chunkX == sizeX-1 ||
		chunkY == 0 || chunkY == sizeY-1 ||
		chunkZ == 0 || chunkZ == sizeZ-1
}

func (c *Chunk) GenerateBlocks() {
	if c.isEdgeChunk() {
		// Relief avec noise sur les chunks d'extrémité
		p := perlin.NewPerlin(2, 2, 3, 42)
		for x := 0; x < ChunkSize; x++ {
			for z := 0; z < ChunkSize; z++ {
				globalX := x + int(c.Position.X())
				globalZ := z + int(c.Position.Z())
				height := int((fractalNoise2D(p, float64(globalX), float64(globalZ), 4, 0.5, 2.0) + 1) * float64(ChunkHeight*3/4))
				if height < 1 {
					height = 1
				}
				for y := 0; y < ChunkHeight; y++ {
					globalY := y + int(c.Position.Y())
					if globalY < height {
						if globalY == height-1 {
							c.Blocks[x][y][z] = Block{Type: BlockTypeGrass}
						} else if globalY > height-5 {
							c.Blocks[x][y][z] = Block{Type: BlockTypeDirt}
						} else {
							c.Blocks[x][y][z] = Block{Type: BlockTypeStone}
						}
					} else {
						c.Blocks[x][y][z] = Block{Type: BlockTypeAir}
					}
				}
			}
		}
	} else {
		// Chunk interne : plein de blocs solides
		for x := 0; x < ChunkSize; x++ {
			for z := 0; z < ChunkSize; z++ {
				for y := 0; y < ChunkHeight; y++ {
					c.Blocks[x][y][z] = Block{Type: BlockTypeStone}
				}
			}
		}
	}
}

// GenerateVertices génère les vertices du chunk
func (c *Chunk) GenerateVertices() {
	if c.isMeshGenerated {
		return
	}

	log.Println("Starting mesh generation...")
	c.Vertices = make([]float32, 0)

	for x := 0; x < ChunkSize; x++ {
		for y := 0; y < ChunkSize; y++ {
			for z := 0; z < ChunkSize; z++ {
				block := c.Blocks[x][y][z]
				if block.Type == BlockTypeAir {
					continue
				}
				for _, dir := range directions {
					if c.isFaceVisible(x, y, z, dir.dx, dir.dy, dir.dz) {
						faceVerts := faceVertices[dir.name]
						if len(faceVerts) == 0 {
							continue
						}
						textureName := GetBlockFaceTextureName(block.Type, dir.name)
						u1, v1, u2, v2 := c.TextureAtlas.GetTextureCoords(textureName)
						if u1 == 0 && v1 == 0 && u2 == 1 && v2 == 1 && textureName != "grass_block_top.png" {
							log.Printf("[WARN] Texture '%s' not found in atlas for block type %d face %s", textureName, block.Type, dir.name)
						}
						for i := 0; i < len(faceVerts); i += 8 {
							vx := faceVerts[i+0]
							vy := faceVerts[i+1]
							vz := faceVerts[i+2]
							tu := faceVerts[i+3]
							uv := faceVerts[i+4]
							nx := faceVerts[i+5]
							ny := faceVerts[i+6]
							nz := faceVerts[i+7]
							// Interpolation UV (tu, uv sont 0 ou 1)
							u := u1 + (u2-u1)*tu
							v := v1 + (v2-v1)*uv
							c.Vertices = append(c.Vertices,
								float32(x)+vx,
								float32(y)+vy,
								float32(z)+vz,
								nx, ny, nz,
								u, v,
							)
						}
					}
				}
			}
		}
	}
	c.isMeshGenerated = true
	log.Printf("Mesh generation complete. Generated %d vertices", len(c.Vertices))
}

// GenerateMesh génère le mesh OpenGL du chunk
func (c *Chunk) GenerateMesh() {
	if len(c.Vertices) == 0 {
		log.Println("No vertices to setup mesh")
		return
	}

	// Vérifier que nous avons un contexte OpenGL valide
	if glfw.GetCurrentContext() == nil {
		log.Println("No valid OpenGL context")
		return
	}

	// Supprimer les anciens buffers s'ils existent
	if c.vao != 0 {
		gl.DeleteVertexArrays(1, &c.vao)
		c.vao = 0
	}
	if c.vbo != 0 {
		gl.DeleteBuffers(1, &c.vbo)
		c.vbo = 0
	}

	// Créer et configurer le VAO
	var vao uint32
	gl.GenVertexArrays(1, &vao)
	if vao == 0 {
		log.Println("Failed to generate VAO")
		return
	}
	c.vao = vao
	gl.BindVertexArray(c.vao)
	log.Println("Created VAO:", c.vao)

	// Créer et configurer le VBO
	var vbo uint32
	gl.GenBuffers(1, &vbo)
	if vbo == 0 {
		log.Println("Failed to generate VBO")
		gl.DeleteVertexArrays(1, &c.vao)
		c.vao = 0
		return
	}
	c.vbo = vbo
	gl.BindBuffer(gl.ARRAY_BUFFER, c.vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(c.Vertices)*4, gl.Ptr(c.Vertices), gl.STATIC_DRAW)
	log.Println("Created VBO:", c.vbo)

	// Configurer les attributs de vertex
	// Position
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 8*4, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)
	// Normale
	gl.VertexAttribPointer(2, 3, gl.FLOAT, false, 8*4, gl.PtrOffset(3*4))
	gl.EnableVertexAttribArray(2)
	// UV
	gl.VertexAttribPointer(1, 2, gl.FLOAT, false, 8*4, gl.PtrOffset(6*4))
	gl.EnableVertexAttribArray(1)

	// Désactiver le VAO
	gl.BindVertexArray(0)

	// Vérifier les erreurs OpenGL
	if err := gl.GetError(); err != gl.NO_ERROR {
		log.Printf("OpenGL error in setupMesh: %v", err)
		// Nettoyer en cas d'erreur
		if c.vao != 0 {
			gl.DeleteVertexArrays(1, &c.vao)
			c.vao = 0
		}
		if c.vbo != 0 {
			gl.DeleteBuffers(1, &c.vbo)
			c.vbo = 0
		}
	}
}

func (c *Chunk) setupMesh() {
	if len(c.Vertices) == 0 {
		log.Println("No vertices to setup mesh")
		return
	}

	// Vérifier que nous avons un contexte OpenGL valide
	if glfw.GetCurrentContext() == nil {
		log.Println("No valid OpenGL context")
		return
	}

	// Supprimer les anciens buffers s'ils existent
	if c.vao != 0 {
		gl.DeleteVertexArrays(1, &c.vao)
		c.vao = 0
	}
	if c.vbo != 0 {
		gl.DeleteBuffers(1, &c.vbo)
		c.vbo = 0
	}

	// Créer et configurer le VAO
	var vao uint32
	gl.GenVertexArrays(1, &vao)
	if vao == 0 {
		log.Println("Failed to generate VAO")
		return
	}
	c.vao = vao
	gl.BindVertexArray(c.vao)
	log.Println("Created VAO:", c.vao)

	// Créer et configurer le VBO
	var vbo uint32
	gl.GenBuffers(1, &vbo)
	if vbo == 0 {
		log.Println("Failed to generate VBO")
		gl.DeleteVertexArrays(1, &c.vao)
		c.vao = 0
		return
	}
	c.vbo = vbo
	gl.BindBuffer(gl.ARRAY_BUFFER, c.vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(c.Vertices)*4, gl.Ptr(c.Vertices), gl.STATIC_DRAW)
	log.Println("Created VBO:", c.vbo)

	// Configurer les attributs de vertex
	// Position
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 8*4, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)
	// Normale
	gl.VertexAttribPointer(2, 3, gl.FLOAT, false, 8*4, gl.PtrOffset(3*4))
	gl.EnableVertexAttribArray(2)
	// UV
	gl.VertexAttribPointer(1, 2, gl.FLOAT, false, 8*4, gl.PtrOffset(6*4))
	gl.EnableVertexAttribArray(1)

	// Désactiver le VAO
	gl.BindVertexArray(0)

	// Vérifier les erreurs OpenGL
	if err := gl.GetError(); err != gl.NO_ERROR {
		log.Printf("OpenGL error in setupMesh: %v", err)
		// Nettoyer en cas d'erreur
		if c.vao != 0 {
			gl.DeleteVertexArrays(1, &c.vao)
			c.vao = 0
		}
		if c.vbo != 0 {
			gl.DeleteBuffers(1, &c.vbo)
			c.vbo = 0
		}
	}
}

func (c *Chunk) Render(currentTime float64) {
	if !c.isInit || !c.isMeshGenerated || len(c.Vertices) == 0 || c.vao == 0 || c.vbo == 0 {
		return
	}

	if glfw.GetCurrentContext() == nil {
		log.Println("No valid OpenGL context in chunk render")
		return
	}

	if c.shader == nil {
		log.Println("Invalid shader in chunk render")
		return
	}

	c.shader.Activate()

	// Toujours binder l'atlas
	if c.TextureAtlas != nil {
		c.TextureAtlas.Bind()
	}
	gl.Uniform1i(gl.GetUniformLocation(c.shader.ID, gl.Str("ourTexture\x00")), 0)

	// Calculer la matrice de transformation
	model := mgl32.Ident4()
	if c.Position != nil {
		model = model.Mul4(mgl32.Translate3D(c.Position.X(), c.Position.Y(), c.Position.Z()))
	}
	if c.Rotation != nil {
		model = model.Mul4(c.Rotation.Mat4())
	}

	// Obtenir la vue et la projection de la caméra
	view := CameraInstance.GetViewMatrix()
	projection := mgl32.Perspective(mgl32.DegToRad(CameraInstance.Zoom), float32(800)/float32(600), 0.1, 1000.0)

	// Définir les uniformes
	modelLoc := gl.GetUniformLocation(c.shader.ID, gl.Str("model\x00"))
	viewLoc := gl.GetUniformLocation(c.shader.ID, gl.Str("view\x00"))
	projLoc := gl.GetUniformLocation(c.shader.ID, gl.Str("projection\x00"))
	lightPosLoc := gl.GetUniformLocation(c.shader.ID, gl.Str("lightPos\x00"))
	lightColorLoc := gl.GetUniformLocation(c.shader.ID, gl.Str("lightColor\x00"))
	viewPosLoc := gl.GetUniformLocation(c.shader.ID, gl.Str("viewPos\x00"))
	ambientLoc := gl.GetUniformLocation(c.shader.ID, gl.Str("ambient\x00"))
	specularStrengthLoc := gl.GetUniformLocation(c.shader.ID, gl.Str("specularStrength\x00"))
	shininessLoc := gl.GetUniformLocation(c.shader.ID, gl.Str("shininess\x00"))

	// Vérifier que tous les uniformes sont valides
	if modelLoc == -1 || viewLoc == -1 || projLoc == -1 || lightPosLoc == -1 ||
		lightColorLoc == -1 || viewPosLoc == -1 || ambientLoc == -1 ||
		specularStrengthLoc == -1 || shininessLoc == -1 {
		log.Println("Invalid uniform locations in chunk render")
		return
	}

	// Définir les valeurs des uniformes
	gl.UniformMatrix4fv(modelLoc, 1, false, &model[0])
	gl.UniformMatrix4fv(viewLoc, 1, false, &view[0])
	gl.UniformMatrix4fv(projLoc, 1, false, &projection[0])

	// Paramètres d'éclairage (ajustés pour une meilleure visibilité)
	lightPos := mgl32.Vec3{10.0, 10.0, 10.0}
	lightColor := mgl32.Vec3{1.0, 1.0, 1.0}
	viewPos := CameraInstance.Position

	gl.Uniform3f(lightPosLoc, lightPos.X(), lightPos.Y(), lightPos.Z())
	gl.Uniform3f(lightColorLoc, lightColor.X(), lightColor.Y(), lightColor.Z())
	gl.Uniform3f(viewPosLoc, viewPos.X(), viewPos.Y(), viewPos.Z())
	gl.Uniform1f(ambientLoc, 0.5) // augmenté (était 0.2)
	gl.Uniform1f(specularStrengthLoc, 0.5)
	gl.Uniform1f(shininessLoc, 32.0)

	// Rendu du mesh
	gl.BindVertexArray(c.vao)
	gl.DrawArrays(gl.TRIANGLES, 0, int32(len(c.Vertices)/8))
	gl.BindVertexArray(0)

	// Vérifier les erreurs OpenGL
	if err := gl.GetError(); err != gl.NO_ERROR {
		log.Printf("OpenGL error in chunk render: %v", err)
		// En cas d'erreur, marquer le mesh comme non généré pour le régénérer
		c.isMeshGenerated = false
	}
}

// Cleanup nettoie les ressources du chunk
func (c *Chunk) Cleanup() {
	if c.vao != 0 {
		gl.DeleteVertexArrays(1, &c.vao)
		c.vao = 0
	}
	if c.vbo != 0 {
		gl.DeleteBuffers(1, &c.vbo)
		c.vbo = 0
	}
	c.Vertices = nil
}

func (c *Chunk) isFaceVisible(x, y, z, dx, dy, dz int) bool {
	nx, ny, nz := x+dx, y+dy, z+dz
	if nx < 0 || nx >= ChunkSize || ny < 0 || ny >= ChunkHeight || nz < 0 || nz >= ChunkSize {
		return true // bord du chunk
	}
	return c.Blocks[nx][ny][nz].Type == BlockTypeAir
}

// GetTextureUV retourne les coordonnées UV pour un type de bloc et une face, selon l'atlas
func (c *Chunk) GetTextureUV(t BlockType, face string) (float32, float32, float32, float32) {
	if c.TextureAtlas == nil {
		return 0, 0, 1, 1
	}
	textureName := getTextureNameForBlockFace(t, face)
	return c.TextureAtlas.GetTextureCoords(textureName)
}

// getTextureNameForBlockFace retourne le nom de la texture pour un bloc et une face
func getTextureNameForBlockFace(t BlockType, face string) string {
	switch t {
	case BlockTypeGrass:
		if face == "top" {
			return "grass_block_top.png"
		}
		return "grass_block_top.png" // à adapter pour d'autres faces
	case BlockTypeDirt:
		return "dirt.png"
	case BlockTypeStone:
		return "stone.png"
	default:
		return "grass_block_top.png"
	}
}
