package main

import (
	"log"
	"math"

	perlin "github.com/aquilax/go-perlin"
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

//////////////////////////////////////////////////////////
// SHADERS : Scène principale (Phong avec lumière locale et lumière du soleil)
//////////////////////////////////////////////////////////

const vertexShaderSource = `
#version 410 core
layout (location = 0) in vec3 aPos;
layout (location = 1) in vec2 aTexCoord;
layout (location = 2) in vec3 aNormal;
out vec2 TexCoord;
out vec3 FragPos;
out vec3 Normal;
uniform mat4 model;
uniform mat4 view;
uniform mat4 projection;
void main()
{
    vec4 worldPos = model * vec4(aPos, 1.0);
    FragPos = worldPos.xyz;
    Normal = mat3(model) * aNormal;
    TexCoord = aTexCoord;
    gl_Position = projection * view * worldPos;
}
` + "\x00"

const fragmentShaderSource = `
#version 410 core
in vec2 TexCoord;
in vec3 FragPos;
in vec3 Normal;
out vec4 FragColor;

// Uniformes pour la lumière locale (point light)
uniform sampler2D ourTexture;
uniform vec3 lightPos;
uniform vec3 lightColor;
uniform float ambient;
uniform float specularStrength;
uniform float shininess;

// Uniformes pour la lumière du soleil (directionnelle)
uniform vec3 sunPos;
uniform vec3 sunColor;
uniform float sunAmbient;

// Uniformes pour la lumière de la lune (directionnelle)
uniform vec3 moonPos;
uniform vec3 moonColor;
uniform float moonAmbient;

// Uniforme pour la caméra
uniform vec3 viewPos;

void main()
{
    // Éclairage local
    vec3 ambientTerm = ambient * lightColor;
    vec3 norm = normalize(Normal);
    vec3 lightDir = normalize(lightPos - FragPos);
    float diff = max(dot(norm, lightDir), 0.0);
    vec3 diffuse = diff * lightColor;
    vec3 viewDir = normalize(viewPos - FragPos);
    vec3 reflectDir = reflect(-lightDir, norm);
    float spec = pow(max(dot(viewDir, reflectDir), 0.0), shininess);
    vec3 specular = specularStrength * spec * lightColor;
    float distance = length(lightPos - FragPos);
    float attenuation = 1.0 - smoothstep(2.0, 5.0, distance);
    vec3 localLight = ambientTerm + attenuation * (diffuse + specular);
    
    // Lumière du soleil (directionnelle)
    vec3 sunDir = normalize(sunPos - vec3(0.0));
    float sunDiff = max(dot(norm, sunDir), 0.0);
    vec3 sunDiffuse = sunDiff * sunColor;
    vec3 sunAmbientTerm = sunAmbient * sunColor;
    vec3 sunReflect = reflect(-sunDir, norm);
    float sunSpec = pow(max(dot(viewDir, sunReflect), 0.0), shininess);
    vec3 sunSpecular = specularStrength * sunSpec * sunColor;
    vec3 sunLight = sunAmbientTerm + sunDiffuse + sunSpecular;
    
    // Lumière de la lune (directionnelle)
    vec3 moonDir = normalize(moonPos - vec3(0.0));
    float moonDiff = max(dot(norm, moonDir), 0.0);
    vec3 moonDiffuse = moonDiff * moonColor;
    vec3 moonAmbientTerm = moonAmbient * moonColor;
    vec3 moonReflect = reflect(-moonDir, norm);
    float moonSpec = pow(max(dot(viewDir, moonReflect), 0.0), shininess);
    vec3 moonSpecular = specularStrength * moonSpec * moonColor;
    vec3 moonLight = moonAmbientTerm + moonDiffuse + moonSpecular;
    
    // Somme des contributions de la lumière locale, du soleil et de la lune
	float sunLightFactor = mix(0, 1, sunDiff);
	sunLight *= sunLightFactor;

	float moonLightFactor = mix(0, 1, moonDiff);
	moonLight *= moonLightFactor;
	
    vec3 finalLight = sunLight + moonLight;
    
    finalLight = max(finalLight, vec3(0.9));

    vec4 texColor = texture(ourTexture, TexCoord);
    FragColor = vec4(finalLight, 1.0) * texColor;
}
` + "\x00"

const (
	ChunkSize   = 16 // Taille horizontale (X et Z)
	ChunkHeight = 16 // Hauteur (Y)
)

// Chunk représente un morceau de terrain composé d'un tableau 3D de blocs.
// Pour simplifier, un bloc est soit présent (true) soit absent (false).
type Chunk struct {
	// Position du chunk dans le monde (en nombre de chunks)
	GlobalPosCentered   mgl32.Vec3
	LocalPosCentered    mgl32.Vec3
	GlobalPosBottomLeft mgl32.Vec3
	LocalPosBottomLeft  mgl32.Vec3
	// Tableau 3D indiquant la présence d'un bloc
	Blocks [ChunkSize][ChunkHeight][ChunkSize]bool

	// Maillage généré : tableau de vertex (chaque vertex : position (3), texCoord (2) et normal (3))
	Vertices []float32

	TextureID     uint32
	ShaderProgram uint32
	World         *World

	// Identifiants OpenGL pour le maillage du chunk
	vao, vbo uint32
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

func NewChunk(globalPosCentered, localPosCentered, globalPosBottomLeft, localPosBottomLeft mgl32.Vec3, seed int64, boundaryFaces []WorldFace, world *World) *Chunk {
	chunk := &Chunk{
		GlobalPosCentered:   globalPosCentered,
		LocalPosCentered:    localPosCentered,
		GlobalPosBottomLeft: globalPosBottomLeft,
		LocalPosBottomLeft:  localPosBottomLeft,
		World:               world,
	}
	// Paramètres de base pour le bruit Perlin
	alpha, beta, nOctaves := 2.0, 2.0, int32(4)
	p := perlin.NewPerlin(alpha, beta, nOctaves, seed)
	activeBlocks := 0

	// Paramètres pour notre "simple noise" inspiré de Minecraft.
	// Pour les faces top/bottom, on travaille sur ChunkHeight,
	// et pour les faces latérales, sur ChunkSize.
	var (
		scaleFactor = float64(ChunkSize * len(world.Chunks)) // Échelle pour les coordonnées
	)
	// Définir les bases et amplitudes en fonction de la dimension
	base := scaleFactor
	amplitude := 20.0

	// Si le chunk n'est pas en boundary, on génère un terrain plat (ou autre)

	// Pour les chunks en boundary (y compris les coins/sommets),
	// on itère sur chaque bloc et on teste pour chaque face concernée.
	for x := 0; x < ChunkSize; x++ {
		for y := 0; y < ChunkHeight; y++ {
			for z := 0; z < ChunkSize; z++ {
				fillBlock := true

				globalX := float64(x) + float64(chunk.GlobalPosBottomLeft.X())
				globalY := float64(y) + float64(chunk.GlobalPosBottomLeft.Y())
				globalZ := float64(z) + float64(chunk.GlobalPosBottomLeft.Z())

				for _, face := range boundaryFaces {
					switch face {
					case WorldFaceTop:
						// Utilise le plan (X,Z) et compare Y.
						threshold := simpleThreshold(p, globalX, globalY, globalZ, base, amplitude, scaleFactor)
						// Si y (la coordonnée locale) est supérieur au seuil, pas de bloc.
						if float64(globalY) >= float64(threshold) {
							fillBlock = false
						}
					case WorldFaceBottom:
						// Utilise le plan (X,Z) et compare la distance depuis le haut.
						threshold := simpleThreshold(p, globalX, globalY, globalZ, base, amplitude, scaleFactor)
						// Ici, on considère la distance depuis le haut (ChunkHeight-1 - y)
						if float64(scaleFactor-globalY) >= float64(threshold) {
							fillBlock = false
						}
					case WorldFaceRight:
						// Face droite : plan (Y,Z) et compare X.
						threshold := simpleThreshold(p, globalX, globalY, globalZ, base, amplitude, scaleFactor)
						// Pour Right, on considère x (depuis le bord gauche)
						if float64(globalX) >= float64(threshold) {
							fillBlock = false
						}
					case WorldFaceLeft:
						// Face gauche : plan (Y,Z) et compare la distance depuis le bord droit.
						threshold := simpleThreshold(p, globalX, globalY, globalZ, base, amplitude, scaleFactor)
						if float64(scaleFactor-globalX) >= float64(threshold) {
							fillBlock = false
						}
					case WorldFaceFront:
						// Face avant : plan (X,Y) et compare Z.
						threshold := simpleThreshold(p, globalX, globalY, globalZ, base, amplitude, scaleFactor)
						if float64(globalZ) >= float64(threshold) {
							fillBlock = false
						}
					case WorldFaceBack:
						// Face arrière : plan (X,Y) et compare la distance depuis le bord avant.
						threshold := simpleThreshold(p, globalX, globalY, globalZ, base, amplitude, scaleFactor)
						if float64(scaleFactor-globalZ) >= float64(threshold) {
							fillBlock = false
						}
					}
					if !fillBlock {
						break
					}
				}
				if fillBlock {
					chunk.Blocks[x][y][z] = true
					activeBlocks++
				}
			}
		}
	}

	return chunk
}

func simpleThreshold(p *perlin.Perlin, x, y, z, base, amplitude, scale float64) int {
	// Décalage pour que x, y, z soient dans [0, scale] si initialement ils sont dans [-scale/2, scale/2]

	noiseVal := p.Noise3D(x/scale, y/scale, z/scale)
	// Normalisation en [0, 1]
	return int(math.Round(base + noiseVal*amplitude))
}

// GenerateMesh parcourt tous les blocs du chunk et construit le maillage
// en n'ajoutant que les faces pour lesquelles il n'existe pas de bloc voisin.
func (c *Chunk) GenerateMesh() {
	var vertices []float32
	for x := 0; x < ChunkSize; x++ {
		for y := 0; y < ChunkHeight; y++ {
			for z := 0; z < ChunkSize; z++ {
				if !c.Blocks[x][y][z] {
					continue
				}
				// Position mondiale du bloc courant
				blockPos := mgl32.Vec3{
					c.GlobalPosCentered.X() + float32(x),
					c.GlobalPosCentered.Y() + float32(y),
					c.GlobalPosCentered.Z() + float32(z),
				}
				// Pour chaque face (direction) on détermine si elle est visible
				for _, dir := range directions {
					// Coordonnées du bloc voisin dans le chunk local
					nx, ny, nz := x+dir.dx, y+dir.dy, z+dir.dz
					// Si le voisin est dans le même chunk, on teste simplement
					if nx >= 0 && nx < ChunkSize &&
						ny >= 0 && ny < ChunkHeight &&
						nz >= 0 && nz < ChunkSize {
						if c.Blocks[nx][ny][nz] {
							continue // Le voisin est présent, on n'ajoute pas cette face.
						}
					} else {
						// Le bloc voisin se trouve dans un autre chunk.
						if worldNeighborBlock(c.World, c, nx, ny, nz) {
							continue // Le voisin existe dans le chunk adjacent, on ne dessine pas la face.
						}
					}

					// Si on arrive ici, la face doit être affichée.
					face, ok := faceVertices[dir.name]
					if !ok {
						continue
					}
					// Pour chaque vertex de la face, décale par blockPos.
					for i := 0; i < len(face); i += 8 {
						vx := face[i+0] + blockPos.X()
						vy := face[i+1] + blockPos.Y()
						vz := face[i+2] + blockPos.Z()
						u := face[i+3]
						v := face[i+4]
						nxF := face[i+5]
						nyF := face[i+6]
						nzF := face[i+7]
						vertices = append(vertices, vx, vy, vz, u, v, nxF, nyF, nzF)
					}
				}
			}
		}
	}
	c.Vertices = vertices
	// log.Printf("Chunk %v: %d vertices générés", c.Pos, len(c.Vertices))
	c.setupMesh()
}

func worldNeighborBlock(world *World, c *Chunk, nx, ny, nz int) bool {
	// Calculer le décalage en chunk (dx, dy, dz)
	dx, dy, dz := 0, 0, 0

	if nx < 0 {
		dx = -1
		nx += ChunkSize
	} else if nx >= ChunkSize {
		dx = 1
		nx -= ChunkSize
	}

	if ny < 0 {
		dy = -1
		ny += ChunkHeight
	} else if ny >= ChunkHeight {
		dy = 1
		ny -= ChunkHeight
	}

	if nz < 0 {
		dz = -1
		nz += ChunkSize
	} else if nz >= ChunkSize {
		dz = 1
		nz -= ChunkSize
	}

	// Déterminer la position du chunk voisin.
	// On suppose que c.Pos est en unités de blocs et que les chunks ont une taille fixe.
	// Par exemple, si c.Pos = (cx*ChunkSize, cy*ChunkHeight, cz*ChunkSize)
	// alors la position du chunk voisin est :
	neighborPos := mgl32.Vec3{
		c.GlobalPosCentered.X() + float32(dx*ChunkSize),
		c.GlobalPosCentered.Y() + float32(dy*ChunkHeight),
		c.GlobalPosCentered.Z() + float32(dz*ChunkSize),
	}

	// Convertir la position du chunk voisin en indices dans le tableau 3D world.Chunks.
	// On suppose qu'il y a un offset (par exemple, chunkRadius) appliqué lors de l'initialisation.
	// Vous devrez adapter ce calcul selon votre implémentation.
	// Ici, on utilise une fonction helper getChunkFromWorld.
	neighborChunk := world.GetChunkAt(neighborPos)
	if neighborChunk == nil {
		// Si le chunk voisin n'existe pas, on considère le bloc comme absent.
		return false
	}
	return neighborChunk.Blocks[nx][ny][nz]
}

// setupMesh crée le VAO et VBO pour le maillage.
func (c *Chunk) setupMesh() {
	if c.vao != 0 {
		gl.DeleteVertexArrays(1, &c.vao)
	}
	if c.vbo != 0 {
		gl.DeleteBuffers(1, &c.vbo)
	}
	if len(c.Vertices) == 0 {
		return
	}
	gl.GenVertexArrays(1, &c.vao)
	gl.GenBuffers(1, &c.vbo)
	gl.BindVertexArray(c.vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, c.vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(c.Vertices)*4, gl.Ptr(c.Vertices), gl.STATIC_DRAW)
	stride := int32(8 * 4) // 8 float32 par vertex
	// Position
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, stride, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)
	// Coordonnées de texture
	gl.VertexAttribPointer(1, 2, gl.FLOAT, false, stride, gl.PtrOffset(3*4))
	gl.EnableVertexAttribArray(1)
	// Normale
	gl.VertexAttribPointer(2, 3, gl.FLOAT, false, stride, gl.PtrOffset(5*4))
	gl.EnableVertexAttribArray(2)
	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
	gl.BindVertexArray(0)
}

func (chunk *Chunk) Init(world *World) {
	var err error
	chunk.TextureID, err = loadTexture("resources/hd/assets/minecraft/textures/block/grass_block_top.png")
	if err != nil {
		log.Fatalln(err)
	}

	chunk.ShaderProgram, err = newProgram(vertexShaderSource, fragmentShaderSource)
	if err != nil {
		log.Fatalln(err)
	}
	gl.UseProgram(chunk.ShaderProgram)
	projLoc := gl.GetUniformLocation(chunk.ShaderProgram, gl.Str("projection\x00"))
	gl.UniformMatrix4fv(projLoc, 1, false, &world.Projection[0])
	model := mgl32.Ident4()
	modelLoc := gl.GetUniformLocation(chunk.ShaderProgram, gl.Str("model\x00"))
	gl.UniformMatrix4fv(modelLoc, 1, false, &model[0])
	lightColorLoc := gl.GetUniformLocation(chunk.ShaderProgram, gl.Str("lightColor\x00"))
	gl.Uniform3f(lightColorLoc, 1.0, 1.0, 1.0)
	ambientLoc := gl.GetUniformLocation(chunk.ShaderProgram, gl.Str("ambient\x00"))
	gl.Uniform1f(ambientLoc, 0.4)
	specularStrengthLoc := gl.GetUniformLocation(chunk.ShaderProgram, gl.Str("specularStrength\x00"))
	gl.Uniform1f(specularStrengthLoc, 0.5)
	shininessLoc := gl.GetUniformLocation(chunk.ShaderProgram, gl.Str("shininess\x00"))
	gl.Uniform1f(shininessLoc, 32.0)

}

func (chunk *Chunk) Render(world *World) {
	gl.UseProgram(chunk.ShaderProgram)

	gl.Enable(gl.DEPTH_TEST)

	sunPosLoc := gl.GetUniformLocation(chunk.ShaderProgram, gl.Str("sunPos\x00"))
	sunColorLoc := gl.GetUniformLocation(chunk.ShaderProgram, gl.Str("sunColor\x00"))
	sunAmbientLoc := gl.GetUniformLocation(chunk.ShaderProgram, gl.Str("sunAmbient\x00"))
	gl.Uniform3fv(sunPosLoc, 1, &world.Sun.Position[0])
	// Utilisation de la couleur du soleil définie dans world.Sun.Color (les composantes R, G, B)
	gl.Uniform3f(sunColorLoc, world.Sun.Color.X(), world.Sun.Color.Y(), world.Sun.Color.Z())
	gl.Uniform1f(sunAmbientLoc, 0.2)

	moonPosLoc := gl.GetUniformLocation(chunk.ShaderProgram, gl.Str("moonPos\x00"))
	moonColorLoc := gl.GetUniformLocation(chunk.ShaderProgram, gl.Str("moonColor\x00"))
	moonAmbientLoc := gl.GetUniformLocation(chunk.ShaderProgram, gl.Str("moonAmbient\x00"))
	gl.Uniform3fv(moonPosLoc, 1, &world.Moon.Position[0])
	// Utilisation de la couleur de la lune définie dans world.Moon.Color (les composantes R, G, B)
	gl.Uniform3f(moonColorLoc, world.Moon.Color.X(), world.Moon.Color.Y(), world.Moon.Color.Z())
	gl.Uniform1f(moonAmbientLoc, 1)

	// Mise à jour de la vue et de la caméra
	view := world.Camera.GetViewMatrix()
	viewLoc := gl.GetUniformLocation(chunk.ShaderProgram, gl.Str("view\x00"))
	lightPosLoc := gl.GetUniformLocation(chunk.ShaderProgram, gl.Str("lightPos\x00"))
	viewPosLoc := gl.GetUniformLocation(chunk.ShaderProgram, gl.Str("viewPos\x00"))
	gl.UniformMatrix4fv(viewLoc, 1, false, &view[0])
	gl.Uniform3fv(viewPosLoc, 1, &world.Camera.Position[0])
	gl.Uniform3fv(lightPosLoc, 1, &world.Camera.Position[0])

	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, chunk.TextureID)
	gl.BindVertexArray(chunk.vao)
	gl.DrawArrays(gl.TRIANGLES, 0, int32(len(chunk.Vertices)/8))
	gl.BindVertexArray(0)
}

func (chunk *Chunk) Clean() {
	gl.DeleteVertexArrays(1, &chunk.vao)
	gl.DeleteBuffers(1, &chunk.vbo)
	gl.DeleteProgram(chunk.ShaderProgram)
}

/////////////////////////////////////////////////////////////
// Génération du Chunk via bruit de Perlin
/////////////////////////////////////////////////////////////
