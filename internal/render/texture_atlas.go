package render

import (
	"fmt"
	"image"
	imagedraw "image/draw"
	"image/png"
	_ "image/png"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/go-gl/gl/v3.3-core/gl"
	xdraw "golang.org/x/image/draw"
)

const (
	AtlasSize      = 2048
	TextureSize    = 128
	TexturesPerRow = AtlasSize / TextureSize
)

// TextureAtlasImpl est une implémentation de l'interface TextureAtlas.
type TextureAtlasImpl struct {
	ID        uint32
	textures  map[string]image.Image
	positions map[string]struct{ x, y int }
	mutex     sync.RWMutex
}

// NewTextureAtlas crée et initialise un nouvel atlas de textures.
func NewTextureAtlas(resourcePackPaths []string, textureFiles []string) (TextureAtlas, error) {
	atlas := &TextureAtlasImpl{
		textures:  make(map[string]image.Image),
		positions: make(map[string]struct{ x, y int }),
	}

	if err := atlas.init(resourcePackPaths, textureFiles); err != nil {
		return nil, fmt.Errorf("failed to initialize texture atlas: %w", err)
	}

	return atlas, nil
}

// init initialise l'atlas de textures
func (a *TextureAtlasImpl) init(resourcePackPaths []string, textureFiles []string) error {
	atlasImage := image.NewRGBA(image.Rect(0, 0, AtlasSize, AtlasSize))

	currentX, currentY := 0, 0
	for _, filename := range textureFiles {
		img, err := findAndLoadTexture(resourcePackPaths, filename)
		if err != nil {
			log.Printf("Warning: could not load texture %s from any resource pack: %v", filename, err)
			continue
		}

		bounds := img.Bounds()
		if bounds.Dx() != TextureSize || bounds.Dy() != TextureSize {
			resized := image.NewRGBA(image.Rect(0, 0, TextureSize, TextureSize))
			xdraw.NearestNeighbor.Scale(resized, resized.Bounds(), img, bounds, xdraw.Over, nil)
			img = resized
		}

		imagedraw.Draw(atlasImage, image.Rect(currentX, currentY, currentX+TextureSize, currentY+TextureSize),
			img, image.Point{0, 0}, imagedraw.Src)

		a.textures[filename] = img
		a.positions[filename] = struct{ x, y int }{currentX, currentY}

		currentX += TextureSize
		if currentX >= AtlasSize {
			currentX = 0
			currentY += TextureSize
		}
	}

	if err := saveAtlasImage(atlasImage, "atlas_debug.png"); err != nil {
		log.Printf("Warning: could not save debug atlas image: %v", err)
	}

	gl.GenTextures(1, &a.ID)
	gl.BindTexture(gl.TEXTURE_2D, a.ID)

	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST_MIPMAP_LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)

	gl.TexImage2D(
		gl.TEXTURE_2D,
		0,
		gl.RGBA,
		int32(AtlasSize),
		int32(AtlasSize),
		0,
		gl.RGBA,
		gl.UNSIGNED_BYTE,
		gl.Ptr(atlasImage.Pix),
	)
	gl.GenerateMipmap(gl.TEXTURE_2D)

	log.Printf("Texture atlas created successfully (size: %dx%d)", AtlasSize, AtlasSize)
	return nil
}

func saveAtlasImage(img *image.RGBA, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("could not create file %s: %w", filename, err)
	}
	defer file.Close()

	if err := png.Encode(file, img); err != nil {
		return fmt.Errorf("could not encode atlas to PNG: %w", err)
	}
	return nil
}

func (a *TextureAtlasImpl) GetTextureCoords(textureName string) (float32, float32, float32, float32) {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	pos, ok := a.positions[textureName]
	if !ok {
		return 0, 0, 1, 1
	}

	x1 := float32(pos.x) / AtlasSize
	y1 := float32(pos.y) / AtlasSize
	x2 := float32(pos.x+TextureSize) / AtlasSize
	y2 := float32(pos.y+TextureSize) / AtlasSize

	return x1, y1, x2, y2
}

func (a *TextureAtlasImpl) Bind(unit uint32) {
	gl.ActiveTexture(gl.TEXTURE0 + unit)
	gl.BindTexture(gl.TEXTURE_2D, a.ID)
}

func (a *TextureAtlasImpl) GetID() uint32 {
	return a.ID
}

func (a *TextureAtlasImpl) Cleanup() {
	if a.ID != 0 {
		gl.DeleteTextures(1, &a.ID)
		a.ID = 0
	}
}

// findAndLoadTexture cherche et charge une image de texture à partir des resource packs.
func findAndLoadTexture(packPaths []string, textureFilename string) (image.Image, error) {
	for _, packPath := range packPaths {
		// Le chemin de base pour les textures de bloc dans un pack de ressources.
		texturePath := filepath.Join(packPath, "assets/minecraft/textures/block", textureFilename)
		if _, err := os.Stat(texturePath); err == nil {
			// Le fichier existe, on tente de le charger.
			img, err := loadTextureImage(texturePath)
			if err == nil {
				return img, nil // Trouvé et chargé avec succès.
			}
			// Si le chargement échoue, on logue une erreur mais on continue,
			// au cas où une version valide existerait dans un pack de plus faible priorité.
			log.Printf("Warning: failed to decode texture '%s' found in '%s', will check other packs. Error: %v", textureFilename, packPath, err)
		}
	}
	return nil, fmt.Errorf("texture '%s' not found in any of the provided resource packs", textureFilename)
}

func loadTextureImage(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	return img, nil
}
