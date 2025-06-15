package main

import (
	"image"
	"image/draw"
	_ "image/png"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/go-gl/gl/v4.6-core/gl"
)

const (
	AtlasSize      = 1024 // Taille de l'atlas en pixels
	TextureSize    = 16   // Taille de chaque texture en pixels
	TexturesPerRow = AtlasSize / TextureSize
)

type TextureAtlas struct {
	ID        uint32
	textures  map[string]image.Image
	positions map[string]struct{ x, y int }
	mutex     sync.Mutex
}

var (
	atlasInstance *TextureAtlas
	atlasOnce     sync.Once
)

// GetTextureAtlas retourne l'instance unique de TextureAtlas
func GetTextureAtlas() *TextureAtlas {
	atlasOnce.Do(func() {
		atlasInstance = &TextureAtlas{
			textures:  make(map[string]image.Image),
			positions: make(map[string]struct{ x, y int }),
		}
		atlasInstance.init()
	})
	return atlasInstance
}

// init initialise l'atlas de textures
func (a *TextureAtlas) init() {
	// Créer une image pour l'atlas
	atlasImage := image.NewRGBA(image.Rect(0, 0, AtlasSize, AtlasSize))

	// Charger toutes les textures
	textureFiles := []string{
		"grass_block_top.png",
		"sand.png",
		"blackstone.png",
		"polished_blackstone.png",
		"deepslate_bricks.png",
		"end_stone_bricks.png",
		"quartz_bricks.png",
		"red_nether_bricks.png",
		"crimson_planks.png",
		"warped_planks.png",
		"acacia_planks.png",
		"birch_planks.png",
		"bamboo_planks.png",
		"bamboo_block.png",
		"bamboo_block_top.png",
		"birch_log.png",
		"birch_log_top.png",
		"cherry_log.png",
		"cherry_log_top.png",
		"mangrove_log.png",
		"mangrove_log_top.png",
		"birch_leaves.png",
		"bamboo_large_leaves.png",
		"bamboo_small_leaves.png",
		"bamboo_stalk.png",
		"bamboo_singleleaf.png",
		"bamboo_stage0.png",
		"birch_sapling.png",
		"dark_oak_sapling.png",
		"jungle_sapling.png",
		"oak_sapling.png",
		"spruce_sapling.png",
		"tall_grass_top.png",
		"tall_grass_bottom.png",
		"short_grass.png",
		"red_sand.png",
		"soul_sand.png",
		"ancient_debris_side.png",
		"ancient_debris_top.png",
		"furnace_front.png",
		"furnace_front_on.png",
		"furnace_side.png",
		"furnace_top.png",
		"beehive_side.png",
		"beehive_front.png",
		"beehive_front_honey.png",
		"beehive_end.png",
		"bee_nest_top.png",
		"bee_nest_side.png",
		"soul_campfire_fire.png",
		"soul_campfire_log_lit.png",
		"soul_lantern.png",
		"soul_torch.png",
		"respawn_anchor_top.png",
		"respawn_anchor_side0.png",
		"respawn_anchor_side1.png",
		"respawn_anchor_side2.png",
		"respawn_anchor_side3.png",
		"respawn_anchor_side4.png",
		"respawn_anchor_bottom.png",
		"respawn_anchor_top_off.png",
		"lodestone_side.png",
		"lodestone_top.png",
		"polished_basalt_side.png",
		"polished_basalt_top.png",
		"gilded_blackstone.png",
		"chiseled_polished_blackstone.png",
		"cracked_polished_blackstone_bricks.png",
		"cracked_deepslate_bricks.png",
		"cracked_nether_bricks.png",
		"chiseled_nether_bricks.png",
		"chain.png",
		"jigsaw_lock.png",
	}

	currentX, currentY := 0, 0
	for _, filename := range textureFiles {
		// Charger la texture
		img, err := loadTextureImage(filepath.Join("resources/hd/assets/minecraft/textures/block", filename))
		if err != nil {
			log.Printf("Erreur lors du chargement de la texture %s: %v", filename, err)
			continue
		}

		// Ajouter la texture à l'atlas
		draw.Draw(atlasImage, image.Rect(currentX, currentY, currentX+TextureSize, currentY+TextureSize),
			img, image.Point{0, 0}, draw.Src)

		// Enregistrer la position de la texture
		a.textures[filename] = img
		a.positions[filename] = struct{ x, y int }{currentX, currentY}

		// Mettre à jour la position pour la prochaine texture
		currentX += TextureSize
		if currentX >= AtlasSize {
			currentX = 0
			currentY += TextureSize
		}
	}

	// Créer la texture OpenGL
	gl.GenTextures(1, &a.ID)
	gl.BindTexture(gl.TEXTURE_2D, a.ID)

	// Configurer les paramètres de la texture
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)

	// Charger les données de l'atlas
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
}

// GetTextureCoords retourne les coordonnées de texture pour une texture donnée
func (a *TextureAtlas) GetTextureCoords(textureName string) (float32, float32, float32, float32) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	pos, ok := a.positions[textureName]
	if !ok {
		// Retourner des coordonnées par défaut si la texture n'est pas trouvée
		return 0, 0, 1, 1
	}

	// Convertir les coordonnées en pixels en coordonnées de texture (0-1)
	x1 := float32(pos.x) / AtlasSize
	y1 := float32(pos.y) / AtlasSize
	x2 := float32(pos.x+TextureSize) / AtlasSize
	y2 := float32(pos.y+TextureSize) / AtlasSize

	return x1, y1, x2, y2
}

// Bind active l'atlas de textures
func (a *TextureAtlas) Bind() {
	gl.BindTexture(gl.TEXTURE_2D, a.ID)
}

// loadTextureImage charge une image depuis un fichier
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

// loadTexture est maintenant déprécié, utilisez GetTextureAtlas() à la place
func loadTexture(path string) (uint32, error) {
	atlas := GetTextureAtlas()
	return atlas.ID, nil
}
