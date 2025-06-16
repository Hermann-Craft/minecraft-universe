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

	// Faces auxquelles ce chunk appartient (pour la génération cubique)
	BoundaryFaces []WorldFace
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
		0, 1, 0, 0, 0, 0, 1, 0,
		0, 1, 1, 0, 1, 0, 1, 0,
		1, 1, 1, 1, 1, 0, 1, 0,
		1, 1, 1, 1, 1, 0, 1, 0,
		1, 1, 0, 1, 0, 0, 1, 0,
		0, 1, 0, 0, 0, 0, 1, 0,
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
func NewChunk(globalPosCentered mgl32.Vec3, seed int64, planet *Planet) (*Chunk, error) {
	chunk := &Chunk{
		Identifier: &goecs.Identifier{Namespace: "core", Path: "chunk"},
		Position:   &globalPosCentered,
		Rotation:   &mgl32.Quat{W: 1, V: mgl32.Vec3{0, 0, 0}},
		Planet:     planet,
	}

	// Charger le shader
	shader, err := LoadShader("default")
	if err != nil {
		return nil, fmt.Errorf("failed to create chunk: %w", err)
	}
	chunk.shader = shader

	// Ne pas générer les blocs ici, ils seront générés après l'assignation des BoundaryFaces
	// chunk.GenerateBlocks()

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
	// Paramètres de génération
	planetSize := c.Planet.Size

	// Paramètres de base pour le bruit Perlin
	alpha, beta, nOctaves := 2.0, 2.0, int32(4)
	p := perlin.NewPerlin(alpha, beta, nOctaves, 42) // seed fixe pour tests

	// Taille totale de la planète en blocs
	totalSizeX := float64(planetSize.X()) * float64(ChunkSize)
	totalSizeY := float64(planetSize.Y()) * float64(ChunkSize)
	totalSizeZ := float64(planetSize.Z()) * float64(ChunkSize)

	// Utiliser la plus petite dimension pour éviter des valeurs trop grandes
	scaleFactor := math.Min(totalSizeX, math.Min(totalSizeY, totalSizeZ))

	// Paramètres ajustés pour un meilleur relief
	base := 10.0       // Distance de base depuis le bord (en blocs)
	amplitude := 5.0   // Amplitude du relief (en blocs)
	noiseScale := 20.0 // Échelle du noise

	// Log pour debug
	if c.Position.X() == 0 && c.Position.Y() == 0 && c.Position.Z() == 0 {
		log.Printf("Génération chunk (0,0,0): scaleFactor=%f, base=%f, amplitude=%f", scaleFactor, base, amplitude)
		log.Printf("BoundaryFaces: %v", c.BoundaryFaces)
	}

	// D'abord, initialiser tous les blocs comme solides
	for x := 0; x < ChunkSize; x++ {
		for y := 0; y < ChunkHeight; y++ {
			for z := 0; z < ChunkSize; z++ {
				c.Blocks[x][y][z] = Block{Type: BlockTypeStone}
			}
		}
	}

	// Si le chunk n'a pas de faces de bord, le laisser plein
	if len(c.BoundaryFaces) == 0 {
		return
	}

	blocksCarved := 0

	// Pour les chunks en boundary, sculpter le terrain
	for x := 0; x < ChunkSize; x++ {
		for y := 0; y < ChunkHeight; y++ {
			for z := 0; z < ChunkSize; z++ {
				// Position globale du bloc
				globalX := float64(x) + float64(c.Position.X())
				globalY := float64(y) + float64(c.Position.Y())
				globalZ := float64(z) + float64(c.Position.Z())

				shouldCarve := false

				for _, face := range c.BoundaryFaces {
					switch face {
					case WorldFaceTop:
						// Face supérieure : on garde les blocs près du sommet
						distFromTop := totalSizeY - globalY
						noiseValue := p.Noise3D(globalX/noiseScale, globalY/noiseScale, globalZ/noiseScale)
						threshold := base + noiseValue*amplitude
						if distFromTop > threshold {
							shouldCarve = true
						}
					case WorldFaceBottom:
						// Face inférieure : on creuse depuis le bas
						distFromBottom := globalY
						noiseValue := p.Noise3D(globalX/noiseScale, globalY/noiseScale, globalZ/noiseScale)
						threshold := base + noiseValue*amplitude
						if distFromBottom < threshold {
							shouldCarve = true
						}
					case WorldFaceRight:
						// Face droite : on creuse depuis la droite
						distFromRight := totalSizeX - globalX
						noiseValue := p.Noise3D(globalX/noiseScale, globalY/noiseScale, globalZ/noiseScale)
						threshold := base + noiseValue*amplitude
						if distFromRight < threshold {
							shouldCarve = true
						}
					case WorldFaceLeft:
						// Face gauche : on creuse depuis la gauche
						distFromLeft := globalX
						noiseValue := p.Noise3D(globalX/noiseScale, globalY/noiseScale, globalZ/noiseScale)
						threshold := base + noiseValue*amplitude
						if distFromLeft < threshold {
							shouldCarve = true
						}
					case WorldFaceFront:
						// Face avant : on creuse depuis l'avant
						distFromFront := totalSizeZ - globalZ
						noiseValue := p.Noise3D(globalX/noiseScale, globalY/noiseScale, globalZ/noiseScale)
						threshold := base + noiseValue*amplitude
						if distFromFront < threshold {
							shouldCarve = true
						}
					case WorldFaceBack:
						// Face arrière : on creuse depuis l'arrière
						distFromBack := globalZ
						noiseValue := p.Noise3D(globalX/noiseScale, globalY/noiseScale, globalZ/noiseScale)
						threshold := base + noiseValue*amplitude
						if distFromBack < threshold {
							shouldCarve = true
						}
					}

					if shouldCarve {
						break
					}
				}

				// Si on doit sculpter (creuser), mettre de l'air
				if shouldCarve {
					c.Blocks[x][y][z] = Block{Type: BlockTypeAir}
					blocksCarved++
				}
			}
		}
	}

	// Log pour debug
	if c.Position.X() == 0 && c.Position.Y() == 0 && c.Position.Z() == 0 {
		log.Printf("Chunk (0,0,0): %d blocs sculptés sur %d", blocksCarved, ChunkSize*ChunkHeight*ChunkSize)
	}

	// Post-traitement : ajouter de l'herbe sur toutes les faces extérieures
	grassAdded := 0
	for x := 0; x < ChunkSize; x++ {
		for y := 0; y < ChunkHeight; y++ {
			for z := 0; z < ChunkSize; z++ {
				if c.Blocks[x][y][z].Type == BlockTypeAir {
					continue
				}
				// Pour chaque direction, vérifier si on est en bord de planète et exposé à l'air
				for _, dir := range directions {
					nx, ny, nz := x+dir.dx, y+dir.dy, z+dir.dz
					globalX := int(c.Position.X()) + x
					globalY := int(c.Position.Y()) + y
					globalZ := int(c.Position.Z()) + z
					isEdge := false
					switch dir.name {
					case "left":
						isEdge = (globalX == 0)
					case "right":
						isEdge = (globalX == int(c.Planet.Size.X())*ChunkSize-1)
					case "bottom":
						isEdge = (globalY == 0)
					case "top":
						isEdge = (globalY == int(c.Planet.Size.Y())*ChunkSize-1)
					case "back":
						isEdge = (globalZ == 0)
					case "front":
						isEdge = (globalZ == int(c.Planet.Size.Z())*ChunkSize-1)
					}
					// Si on est en bord ET exposé à l'air
					if isEdge &&
						nx >= 0 && nx < ChunkSize &&
						ny >= 0 && ny < ChunkHeight &&
						nz >= 0 && nz < ChunkSize &&
						c.Blocks[nx][ny][nz].Type == BlockTypeAir {
						c.Blocks[x][y][z] = Block{Type: BlockTypeGrass}
						grassAdded++
						// Ajouter de la dirt "sous" l'herbe (vers l'intérieur de la planète)
						ix, iy, iz := x-dir.dx, y-dir.dy, z-dir.dz
						if ix >= 0 && ix < ChunkSize && iy >= 0 && iy < ChunkHeight && iz >= 0 && iz < ChunkSize {
							if c.Blocks[ix][iy][iz].Type == BlockTypeStone {
								c.Blocks[ix][iy][iz] = Block{Type: BlockTypeDirt}
							}
						}
						break // On ne traite qu'une face par bloc
					}
				}
			}
		}
	}

	if c.Position.X() == 0 && c.Position.Y() == 0 && c.Position.Z() == 0 {
		log.Printf("Chunk (0,0,0): %d blocs d'herbe ajoutés", grassAdded)
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
		for y := 0; y < ChunkHeight; y++ {
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
	// Calculer la position du bloc adjacent
	adjacentX := x + dx
	adjacentY := y + dy
	adjacentZ := z + dz

	// Si le bloc adjacent est dans le même chunk
	if adjacentX >= 0 && adjacentX < ChunkSize &&
		adjacentY >= 0 && adjacentY < ChunkHeight &&
		adjacentZ >= 0 && adjacentZ < ChunkSize {
		// Vérifier si le bloc adjacent est de l'air
		return c.Blocks[adjacentX][adjacentY][adjacentZ].Type == BlockTypeAir
	}

	// Si le bloc adjacent est dans un autre chunk
	// Calculer la position du chunk actuel dans la grille de la planète
	chunkX := int(c.Position.X()) / ChunkSize
	chunkY := int(c.Position.Y()) / ChunkSize
	chunkZ := int(c.Position.Z()) / ChunkSize

	// Calculer quel chunk contient le bloc adjacent
	neighborChunkX := chunkX
	neighborChunkY := chunkY
	neighborChunkZ := chunkZ

	// Position du bloc dans le chunk voisin
	neighborBlockX := adjacentX
	neighborBlockY := adjacentY
	neighborBlockZ := adjacentZ

	// Ajuster les coordonnées du chunk et du bloc si on traverse une frontière
	if adjacentX < 0 {
		neighborChunkX--
		neighborBlockX = ChunkSize - 1
	} else if adjacentX >= ChunkSize {
		neighborChunkX++
		neighborBlockX = 0
	}

	if adjacentY < 0 {
		neighborChunkY--
		neighborBlockY = ChunkHeight - 1
	} else if adjacentY >= ChunkHeight {
		neighborChunkY++
		neighborBlockY = 0
	}

	if adjacentZ < 0 {
		neighborChunkZ--
		neighborBlockZ = ChunkSize - 1
	} else if adjacentZ >= ChunkSize {
		neighborChunkZ++
		neighborBlockZ = 0
	}

	// Vérifier si le chunk voisin existe dans la planète
	if neighborChunkX >= 0 && neighborChunkX < int(c.Planet.Size.X()) &&
		neighborChunkY >= 0 && neighborChunkY < int(c.Planet.Size.Y()) &&
		neighborChunkZ >= 0 && neighborChunkZ < int(c.Planet.Size.Z()) {

		neighborChunk := c.Planet.Chunks[neighborChunkX][neighborChunkY][neighborChunkZ]
		if neighborChunk != nil {
			// Vérifier si le bloc dans le chunk voisin est de l'air
			return neighborChunk.Blocks[neighborBlockX][neighborBlockY][neighborBlockZ].Type == BlockTypeAir
		}
		// Si le chunk voisin n'est pas encore chargé, on rend la face
		return true
	}

	// Si on est en dehors des limites de la planète
	// On est sur une face extérieure de la planète, donc on rend la face
	return true
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
		switch face {
		case "top":
			return "grass_block_top.png"
		case "bottom":
			return "dirt.png"
		default:
			return "grass_block_side.png"
		}
	case BlockTypeDirt:
		return "dirt.png"
	case BlockTypeStone:
		return "stone.png"
	default:
		return "grass_block_top.png"
	}
}

// RaycastBlock effectue un raycast depuis une position et une direction, et retourne le premier bloc solide touché
// Retourne : chunk, coordonnées locales du bloc (bx, by, bz), et hit (bool)
func RaycastBlock(origin, direction mgl32.Vec3, planet *Planet, maxDistance float32) (*Chunk, int, int, int, bool) {
	step := float32(0.1) // précision du raycast
	dir := direction.Normalize()
	for t := float32(0); t < maxDistance; t += step {
		pos := origin.Add(dir.Mul(t))
		// Convertir la position globale en indices de chunk et de bloc
		chunkX := int(pos.X()) / ChunkSize
		chunkY := int(pos.Y()) / ChunkSize
		chunkZ := int(pos.Z()) / ChunkSize
		bx := int(pos.X()) % ChunkSize
		by := int(pos.Y()) % ChunkHeight
		bz := int(pos.Z()) % ChunkSize
		if bx < 0 {
			bx += ChunkSize
		}
		if by < 0 {
			by += ChunkHeight
		}
		if bz < 0 {
			bz += ChunkSize
		}
		// Vérifier les bornes
		if chunkX < 0 || chunkY < 0 || chunkZ < 0 ||
			chunkX >= int(planet.Size.X()) || chunkY >= int(planet.Size.Y()) || chunkZ >= int(planet.Size.Z()) {
			continue
		}
		chunk := planet.Chunks[chunkX][chunkY][chunkZ]
		if chunk == nil {
			continue
		}
		block := chunk.Blocks[bx][by][bz]
		if block.Type != BlockTypeAir {
			return chunk, bx, by, bz, true
		}
	}
	return nil, 0, 0, 0, false
}
