package render

import (
	"fmt"
	"image"
	"log"
	"os"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/mathgl/mgl32"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

//////////////////////////////////////////////////////////
// SHADERS POUR LE TEXTE (rendu 2D via FreeType)
//////////////////////////////////////////////////////////

const textVertexShaderSource = `
#version 330 core
layout (location = 0) in vec4 vertex; // <vec2 pos, vec2 tex>
out vec2 TexCoords;
uniform mat4 projection;
void main() {
    gl_Position = projection * vec4(vertex.xy, 0.0, 1.0);
    TexCoords = vertex.zw;
}
` + "\x00"

const textFragmentShaderSource = `
#version 330 core
in vec2 TexCoords;
out vec4 FragColor;
uniform sampler2D text;
uniform vec3 textColor;
void main() {    
    vec4 tex = texture(text, TexCoords);
    FragColor = vec4(textColor, tex.a) * tex;
}

` + "\x00"

//////////////////////////////////////////////////////////
// Rendu du texte via une font (FreeType)
//////////////////////////////////////////////////////////

// Structure contenant les données d'un glyphe.
type Character struct {
	TextureID uint32     // ID de la texture OpenGL du glyphe
	Size      mgl32.Vec2 // Taille du glyphe en pixels
	Bearing   mgl32.Vec2 // Décalage par rapport à la ligne de base
	Advance   uint32     // Avance vers le prochain glyphe (valeur en 1/64ème de pixel)
}

var Characters map[rune]Character

var (
	screenWidth, screenHeight = 800, 600
)

// SetScreenDimensions updates the screen dimensions for text projection
func SetScreenDimensions(w, h int) {
	screenWidth = w
	screenHeight = h
}

type TextRenderer struct {
	textVAO, textVBO  uint32
	textShaderProgram uint32
}

func (textRenderer *TextRenderer) Init() error {
	// Initialisation du rendu du texte :
	var err error
	textRenderer.textShaderProgram, err = newProgram(textVertexShaderSource, textFragmentShaderSource)
	if err != nil {
		log.Fatalln("Erreur lors de la création du shader de texte:", err)
		return err
	}
	// Chargez la police (ici "resources/arial.ttf", taille 48 par exemple)
	if err := loadFontAtlas("resources/PixelifySans-Regular.ttf", 16); err != nil {
		log.Fatalln("Erreur lors du chargement de la police:", err)
		return err
	}
	gl.GenVertexArrays(1, &textRenderer.textVAO)
	gl.GenBuffers(1, &textRenderer.textVBO)
	gl.BindVertexArray(textRenderer.textVAO)
	gl.BindBuffer(gl.ARRAY_BUFFER, textRenderer.textVBO)
	// On alloue suffisamment d'espace pour 6 sommets * 4 floats (x, y, u, v)
	gl.BufferData(gl.ARRAY_BUFFER, 6*4*4, nil, gl.DYNAMIC_DRAW)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 4, gl.FLOAT, false, 4*4, gl.PtrOffset(0))
	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
	gl.BindVertexArray(0)

	return nil
}

// generateTextureFromImage crée une texture OpenGL à partir d'une image RGBA.
func generateTextureFromImage(img *image.RGBA) uint32 {
	var texture uint32
	gl.GenTextures(1, &texture)
	gl.BindTexture(gl.TEXTURE_2D, texture)
	gl.PixelStorei(gl.UNPACK_ALIGNMENT, 1)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, int32(img.Rect.Dx()), int32(img.Rect.Dy()), 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(img.Pix))
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	return texture
}

// loadFontAtlas charge la police depuis un fichier TrueType, et pour chaque glyphe (ASCII 0-127) génère une texture.
func loadFontAtlas(fontPath string, fontSize float64) error {
	fontBytes, err := os.ReadFile(fontPath)
	if err != nil {
		return fmt.Errorf("impossible de lire le fichier de police: %v", err)
	}
	fontParsed, err := opentype.Parse(fontBytes)
	if err != nil {
		return fmt.Errorf("impossible d'analyser la police: %v", err)
	}
	face, err := opentype.NewFace(fontParsed, &opentype.FaceOptions{
		Size:    fontSize,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return fmt.Errorf("impossible de créer la face: %v", err)
	}

	Characters = make(map[rune]Character)

	// Pour les 128 premiers caractères ASCII
	for c := rune(0); c < 128; c++ {
		bounds, advance, ok := face.GlyphBounds(c)
		if !ok {
			continue
		}
		glyphWidth := (bounds.Max.X - bounds.Min.X).Round()
		glyphHeight := (bounds.Max.Y - bounds.Min.Y).Round()
		// Certains glyphes (comme l'espace) n'ont pas de bitmap
		if glyphWidth <= 0 || glyphHeight <= 0 {
			Characters[c] = Character{
				TextureID: 0,
				Size:      mgl32.Vec2{0, 0},
				Bearing:   mgl32.Vec2{0, 0},
				Advance:   uint32(advance.Round()),
			}
			continue
		}

		// Créer une image Alpha pour avoir un canal unique
		img := image.NewRGBA(image.Rect(0, 0, glyphWidth, glyphHeight))
		// L'image est déjà initialisée à 0 (transparent)

		// Préparez un Drawer qui dessine dans une image Alpha.
		// Notez que vous pouvez utiliser image.White (qui a une valeur Alpha de 0xff) pour dessiner.
		d := &font.Drawer{
			Dst:  img,
			Src:  image.White,
			Face: face,
			Dot: fixed.Point26_6{
				X: -bounds.Min.X,
				Y: -bounds.Min.Y,
			},
		}
		d.DrawString(string(c))

		textureID := generateTextureFromImage(img)
		Characters[c] = Character{
			TextureID: textureID,
			Size:      mgl32.Vec2{float32(glyphWidth), float32(glyphHeight)},
			Bearing:   mgl32.Vec2{float32(bounds.Min.X.Round()), float32(bounds.Min.Y.Round())},
			Advance:   uint32(advance.Round()),
		}
	}
	log.Printf("Nombre de glyphes chargés : %d", len(Characters))
	return nil
}

// renderText dessine la chaîne 'text' à la position (x, y) (la ligne de base) avec une échelle donnée.
// La projection utilisée est orthographique (origine en haut à gauche).
// renderText dessine la chaîne 'text' à la position (x, y) (ligne de base) avec une échelle donnée.
func (textRenderer *TextRenderer) RenderText(text string, x, y, scale float32) {
	// Use text shader and disable depth test and face culling for overlay
	gl.UseProgram(textRenderer.textShaderProgram)
	gl.Disable(gl.DEPTH_TEST)
	gl.Disable(gl.CULL_FACE)
	// Projection avec origine en haut à gauche
	projection := mgl32.Ortho2D(0, float32(screenWidth), float32(screenHeight), 0)
	projLoc := gl.GetUniformLocation(textRenderer.textShaderProgram, gl.Str("projection\x00"))
	gl.UniformMatrix4fv(projLoc, 1, false, &projection[0])

	colorLoc := gl.GetUniformLocation(textRenderer.textShaderProgram, gl.Str("textColor\x00"))
	gl.Uniform3f(colorLoc, 1.0, 1.0, 1.0)

	gl.Uniform1i(gl.GetUniformLocation(textRenderer.textShaderProgram, gl.Str("text\x00")), 0)

	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindVertexArray(textRenderer.textVAO)
	gl.EnableVertexAttribArray(0)

	var xpos float32 = x
	for _, c := range text {
		ch, ok := Characters[c]
		if !ok {
			continue
		}
		// Positionnement en X en tenant compte du bearing
		ypos := y + float32(ch.Bearing.Y())*scale
		w := ch.Size.X() * scale
		h := ch.Size.Y() * scale

		// Construction des sommets du rectangle :
		// top-left, bottom-left, bottom-right, puis top-left, bottom-right, top-right
		vertices := []float32{
			xpos, ypos, 0.0, 0.0, // top-left
			xpos, ypos + (h), 0.0, 1.0, // bottom-left
			xpos + w, ypos + h, 1.0, 1.0, // bottom-right

			xpos, ypos, 0.0, 0.0, // top-left
			xpos + w, ypos + h, 1.0, 1.0, // bottom-right
			xpos + w, ypos, 1.0, 0.0, // top-right
		}

		gl.BindBuffer(gl.ARRAY_BUFFER, textRenderer.textVBO)
		gl.BufferSubData(gl.ARRAY_BUFFER, 0, len(vertices)*4, gl.Ptr(vertices))
		gl.BindTexture(gl.TEXTURE_2D, ch.TextureID)
		gl.DrawArrays(gl.TRIANGLES, 0, 6)
		err := gl.GetError()
		if err != gl.NO_ERROR {
			log.Printf("[TextRenderer] GL error %#x", err)
		}
		// On avance la position horizontale pour le prochain glyphe

		xpos += float32(ch.Advance) * scale
	}

	gl.BindVertexArray(0)
	gl.BindTexture(gl.TEXTURE_2D, 0)
	// gl.Disable(gl.BLEND)
	// gl.Enable(gl.DEPTH_TEST)
	// Restore face culling and depth test
	gl.Enable(gl.CULL_FACE)
	gl.Enable(gl.DEPTH_TEST)
}

// newProgram compiles vertex and fragment shader sources and links them into a program
func newProgram(vertexSource, fragmentSource string) (uint32, error) {
	// Compile vertex shader
	vertexShader, err := compileShader(vertexSource, gl.VERTEX_SHADER)
	if err != nil {
		return 0, fmt.Errorf("failed to compile vertex shader for text: %w", err)
	}
	defer gl.DeleteShader(vertexShader)

	// Compile fragment shader
	fragmentShader, err := compileShader(fragmentSource, gl.FRAGMENT_SHADER)
	if err != nil {
		return 0, fmt.Errorf("failed to compile fragment shader for text: %w", err)
	}
	defer gl.DeleteShader(fragmentShader)

	// Link program
	program := gl.CreateProgram()
	gl.AttachShader(program, vertexShader)
	gl.AttachShader(program, fragmentShader)
	gl.LinkProgram(program)

	var status int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLength)
		logBytes := make([]byte, logLength)
		gl.GetProgramInfoLog(program, logLength, nil, &logBytes[0])
		gl.DeleteProgram(program)
		return 0, fmt.Errorf("failed to link text shader program: %s", string(logBytes))
	}

	return program, nil
}
