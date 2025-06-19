package world

import (
	"encoding/json"
	"fmt"
	"image"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// BlockFace représente la texture et d'autres propriétés d'une face de bloc.
type BlockFace struct {
	UV        []float32 `json:"uv"`
	Texture   string    `json:"texture"`
	CullFace  string    `json:"cullface"`
	TintIndex int       `json:"tintindex"`
}

// BlockElement représente un élément cubique dans un modèle de bloc.
type BlockElement struct {
	From  []float32            `json:"from"`
	To    []float32            `json:"to"`
	Faces map[string]BlockFace `json:"faces"`
}

// BlockModel représente le modèle JSON d'un bloc.
type BlockModel struct {
	Parent   string            `json:"parent"`
	Textures map[string]string `json:"textures"`
	Elements []BlockElement    `json:"elements"`
}

// BlockRegistry gère le chargement et l'accès aux définitions des blocs.
type BlockRegistry struct {
	Models            map[string]*BlockModel
	RawModels         map[string]*BlockModel // Temporaire pour le processus de chargement
	textureList       map[string]struct{}
	resourcePackPaths []string
	TextureCoords     map[string]image.Rectangle
	blockModelsMap    map[BlockType]string
	basePath          string
}

// NewBlockRegistry crée une nouvelle instance de BlockRegistry.
func NewBlockRegistry() *BlockRegistry {
	br := &BlockRegistry{
		Models:         make(map[string]*BlockModel),
		RawModels:      make(map[string]*BlockModel),
		textureList:    make(map[string]struct{}),
		TextureCoords:  make(map[string]image.Rectangle),
		blockModelsMap: make(map[BlockType]string),
	}

	// Pré-enregistrer les modèles de blocs de base du jeu.
	// C'est ici que nous lions notre enum interne BlockType aux noms de modèles de Minecraft.
	br.RegisterBlockModel(BlockTypeStone, "minecraft:block/stone")
	br.RegisterBlockModel(BlockTypeDirt, "minecraft:block/dirt")
	br.RegisterBlockModel(BlockTypeGrass, "minecraft:block/grass_block")
	br.RegisterBlockModel(BlockTypeCobblestone, "minecraft:block/cobblestone")
	br.RegisterBlockModel(BlockTypeOakLog, "minecraft:block/oak_log")
	br.RegisterBlockModel(BlockTypeOakLeaves, "minecraft:block/oak_leaves")
	br.RegisterBlockModel(BlockTypeSand, "minecraft:block/sand")
	br.RegisterBlockModel(BlockTypeWater, "minecraft:block/water")
	br.RegisterBlockModel(BlockTypeBedrock, "minecraft:block/bedrock")
	// Air (BlockTypeAir) n'a pas de modèle, donc on ne l'enregistre pas.

	return br
}

// RegisterBlockModel maps a block type to a model name.
func (br *BlockRegistry) RegisterBlockModel(blockType BlockType, modelName string) {
	br.blockModelsMap[blockType] = modelName
}

// GetModelForBlock returns the model for a given block type.
func (br *BlockRegistry) GetModelForBlock(blockType BlockType) (*BlockModel, bool) {
	modelName, ok := br.blockModelsMap[blockType]
	if !ok {
		// This is not an error, as some blocks (like air) have no model.
		return nil, false
	}
	model, ok := br.Models[modelName]
	if !ok {
		log.Printf("[ERROR] Model '%s' not found in loaded models registry (for block type %d)", modelName, blockType)
		return nil, false
	}
	return model, true
}

// LoadResourcePack charge un seul resource pack.
// C'est un wrapper pratique pour LoadResourcePacks.
func (br *BlockRegistry) LoadResourcePack(packPath string) ([]string, error) {
	return br.LoadResourcePacks([]string{packPath})
}

// normalizeIdentifier normalise un identifiant de modèle en s'assurant qu'il a le préfixe "minecraft:block/".
func normalizeIdentifier(id string) string {
	if !strings.Contains(id, ":") {
		// Gère les cas comme "block/cube" et "cube".
		// Si l'ID ne contient pas de '/', nous supposons qu'il s'agit d'un préfixe de bloc.
		if !strings.Contains(id, "/") {
			id = "block/" + id
		}
		return "minecraft:" + id
	}
	return id
}

// LoadResourcePacks scanne les resource packs, charge tous les modèles de blocs et retourne la liste des textures requises.
// Les packs sont chargés dans l'ordre fourni, avec les premiers ayant la priorité.
func (br *BlockRegistry) LoadResourcePacks(packPaths []string) ([]string, error) {
	br.resourcePackPaths = packPaths

	// L'ordre est important : le premier pack a la priorité.
	// Nous inversons la liste pour que les packs de plus basse priorité (comme vanilla)
	// soient traités en premier. Leurs modèles seront écrasés par les packs de plus
	// haute priorité (comme les resource packs personnalisés) si des modèles existent aux mêmes noms.
	for i := len(br.resourcePackPaths) - 1; i >= 0; i-- {
		packPath := br.resourcePackPaths[i]
		log.Printf("[INFO] Loading resource pack from: %s (priority: %d)", packPath, len(br.resourcePackPaths)-1-i)
		modelsPath := filepath.Join(packPath, "assets/minecraft/models/block")
		log.Printf("Scanning for block models in: %s", modelsPath)

		err := filepath.Walk(modelsPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(info.Name(), ".json") {
				// ex: models/block/stone.json -> minecraft:block/stone
				modelsRoot := filepath.Join(packPath, "assets", "minecraft", "models")
				relPath, _ := filepath.Rel(modelsRoot, path)
				modelIdentifier := "minecraft:" + strings.TrimSuffix(relPath, ".json")
				modelIdentifier = strings.ReplaceAll(modelIdentifier, "\\", "/")

				// Contrairement à avant, nous permettons maintenant l'écrasement.
				// Le dernier pack chargé (qui a la plus haute priorité) gagnera.
				model, loadErr := br.loadModel(path)
				if loadErr != nil {
					log.Printf("Warning: could not load raw model %s: %v", modelIdentifier, loadErr)
					return nil // Continue
				}
				br.RawModels[modelIdentifier] = model
			}
			return nil
		})
		if err != nil {
			log.Printf("Warning: failed to walk models directory %s: %v. This might be a non-standard pack.", modelsPath, err)
			// On ne retourne pas d'erreur ici pour permettre de continuer avec d'autres packs
		}
	}

	// Étape 2: Résoudre seulement les modèles enregistrés pour nos types de blocs
	log.Println("Resolving registered block models...")
	for _, modelName := range br.blockModelsMap {
		if _, err := br.resolveModel(modelName, make(map[string]bool)); err != nil {
			log.Printf("Warning: could not resolve registered model %s: %v", modelName, err)
		}
	}

	// Étape 3: Collecter toutes les textures des modèles résolus
	for _, model := range br.Models {
		// The textures map should now contain only resolved texture paths.
		for _, texturePath := range model.Textures {
			if strings.HasPrefix(texturePath, "#") {
				// This should not happen if resolution is correct.
				log.Printf("Warning: Unresolved texture variable '%s' found in final model, skipping.", texturePath)
				continue
			}
			cleanPath := strings.TrimPrefix(texturePath, "minecraft:block/")
			cleanPath = strings.TrimPrefix(cleanPath, "block/")
			br.textureList[cleanPath] = struct{}{}
		}
	}

	textures := make([]string, 0, len(br.textureList))
	for texture := range br.textureList {
		if !strings.HasSuffix(texture, ".png") {
			textures = append(textures, texture+".png")
		} else {
			textures = append(textures, texture)
		}
	}

	log.Printf("Loaded %d raw models, resolved %d final models, found %d unique textures.", len(br.RawModels), len(br.Models), len(textures))
	br.RawModels = nil // Libérer la mémoire
	return textures, nil
}

// resolveModel résout récursivement un modèle de bloc, en gérant l'héritage des parents.
func (br *BlockRegistry) resolveModel(modelIdentifier string, visited map[string]bool) (*BlockModel, error) {
	// 1. Check cache for already resolved model.
	if model, ok := br.Models[modelIdentifier]; ok {
		return model, nil
	}

	// 2. Prevent infinite recursion.
	if visited[modelIdentifier] {
		return nil, fmt.Errorf("infinite recursion detected for model '%s'", modelIdentifier)
	}
	visited[modelIdentifier] = true
	defer delete(visited, modelIdentifier)

	// 3. Get raw model.
	rawModel, ok := br.RawModels[modelIdentifier]
	if !ok {
		return nil, fmt.Errorf("raw model '%s' not found", modelIdentifier)
	}

	// 4. Resolve parent first to build the inheritance chain.
	var parentModel *BlockModel
	if rawModel.Parent != "" {
		parentIdentifier := normalizeIdentifier(rawModel.Parent)
		var err error
		parentModel, err = br.resolveModel(parentIdentifier, visited)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve parent '%s' for '%s': %w", parentIdentifier, modelIdentifier, err)
		}
	}

	resolvedModel := &BlockModel{}

	// 5. Inherit elements from parent or use child's elements.
	if len(rawModel.Elements) > 0 {
		// Child has elements, use them
		resolvedModel.Elements = make([]BlockElement, len(rawModel.Elements))
		for i, el := range rawModel.Elements {
			resolvedModel.Elements[i] = BlockElement{
				From:  append([]float32(nil), el.From...),
				To:    append([]float32(nil), el.To...),
				Faces: make(map[string]BlockFace),
			}
			for faceName, face := range el.Faces {
				resolvedModel.Elements[i].Faces[faceName] = face
			}
		}
	} else if parentModel != nil && len(parentModel.Elements) > 0 {
		// Use parent's elements
		resolvedModel.Elements = make([]BlockElement, len(parentModel.Elements))
		for i, el := range parentModel.Elements {
			resolvedModel.Elements[i] = BlockElement{
				From:  append([]float32(nil), el.From...),
				To:    append([]float32(nil), el.To...),
				Faces: make(map[string]BlockFace),
			}
			for faceName, face := range el.Faces {
				resolvedModel.Elements[i].Faces[faceName] = face
			}
		}
	}

	// 6. Build the complete texture map by merging parent and child textures.
	// We need to merge the RAW texture definitions, not the resolved ones.
	textureMap := make(map[string]string)

	// Recursively collect texture definitions from the inheritance chain
	var collectTextures func(modelId string) error
	collectTextures = func(modelId string) error {
		if modelId == "" {
			return nil
		}

		normalizedId := normalizeIdentifier(modelId)
		rawParent, exists := br.RawModels[normalizedId]
		if !exists {
			return fmt.Errorf("parent model '%s' not found", normalizedId)
		}

		// First collect from grandparent
		if rawParent.Parent != "" {
			if err := collectTextures(rawParent.Parent); err != nil {
				return err
			}
		}

		// Then add this level's textures (child overrides parent)
		if rawParent.Textures != nil {
			for k, v := range rawParent.Textures {
				textureMap[k] = v
			}
		}

		return nil
	}

	// Collect textures from parent chain
	if rawModel.Parent != "" {
		if err := collectTextures(rawModel.Parent); err != nil {
			log.Printf("Warning: failed to collect parent textures for %s: %v", modelIdentifier, err)
		}
	}

	// Finally, add child's textures (highest priority)
	if rawModel.Textures != nil {
		for k, v := range rawModel.Textures {
			textureMap[k] = v
		}
	}

	// 7. Resolve all texture variables in the texture map.
	// We need to iterate until no more variables can be resolved.
	maxIterations := 20
	for iteration := 0; iteration < maxIterations; iteration++ {
		changed := false
		for key, value := range textureMap {
			if strings.HasPrefix(value, "#") {
				refKey := strings.TrimPrefix(value, "#")
				if resolvedValue, exists := textureMap[refKey]; exists {
					if !strings.HasPrefix(resolvedValue, "#") {
						// Found a concrete texture
						textureMap[key] = resolvedValue
						changed = true
					}
				}
			}
		}
		if !changed {
			break
		}
	}

	// 8. Apply resolved textures to all faces in all elements.
	finalTextures := make(map[string]string)
	for i := range resolvedModel.Elements {
		for faceName, face := range resolvedModel.Elements[i].Faces {
			originalTexture := face.Texture
			finalTexture := originalTexture

			// Resolve texture variable if it exists
			if strings.HasPrefix(originalTexture, "#") {
				refKey := strings.TrimPrefix(originalTexture, "#")
				if resolvedTexture, exists := textureMap[refKey]; exists {
					if !strings.HasPrefix(resolvedTexture, "#") {
						finalTexture = resolvedTexture
						finalTextures[resolvedTexture] = resolvedTexture
					} else {
						log.Printf("Warning: Unresolved texture variable '%s' for face '%s' in model '%s'", resolvedTexture, faceName, modelIdentifier)
						continue
					}
				} else {
					log.Printf("Warning: Texture variable '%s' not found for face '%s' in model '%s'", originalTexture, faceName, modelIdentifier)
					continue
				}
			} else {
				// Already a concrete texture
				finalTextures[finalTexture] = finalTexture
			}

			// Update the face with the resolved texture
			face.Texture = finalTexture
			resolvedModel.Elements[i].Faces[faceName] = face
		}
	}

	// Add particle texture if it exists and is resolved
	if particleTexture, exists := textureMap["particle"]; exists {
		if !strings.HasPrefix(particleTexture, "#") {
			finalTextures[particleTexture] = particleTexture
		}
	}

	resolvedModel.Textures = finalTextures

	// 9. Cache and return.
	br.Models[modelIdentifier] = resolvedModel
	return resolvedModel, nil
}

// loadModel charge un modèle de bloc depuis un fichier JSON.
func (r *BlockRegistry) loadModel(path string) (*BlockModel, error) {
	jsonFile, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open model file %s: %w", path, err)
	}
	defer jsonFile.Close()

	bytes, err := io.ReadAll(jsonFile)
	if err != nil {
		return nil, fmt.Errorf("could not read model file %s: %w", path, err)
	}

	var model BlockModel
	if err := json.Unmarshal(bytes, &model); err != nil {
		return nil, fmt.Errorf("could not unmarshal model file %s: %w", path, err)
	}

	return &model, nil
}

// GetModel retourne un modèle de bloc par son nom (ex: "minecraft:grass_block").
func (r *BlockRegistry) GetModel(name string) (*BlockModel, bool) {
	model, found := r.Models[name]
	return model, found
}
