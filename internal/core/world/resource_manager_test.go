package world

import (
	"image"
	"testing"
)

// TestNewBlockRegistry teste la création d'un nouveau BlockRegistry
func TestNewBlockRegistry(t *testing.T) {
	registry := NewBlockRegistry()

	if registry == nil {
		t.Fatal("NewBlockRegistry returned nil")
	}

	if registry.Models == nil {
		t.Error("Expected Models map to be initialized")
	}

	if registry.RawModels == nil {
		t.Error("Expected RawModels map to be initialized")
	}

	if registry.textureList == nil {
		t.Error("Expected textureList map to be initialized")
	}

	if registry.TextureCoords == nil {
		t.Error("Expected TextureCoords map to be initialized")
	}

	if registry.blockModelsMap == nil {
		t.Error("Expected blockModelsMap to be initialized")
	}
}

// TestBlockRegistryModelMapping teste l'enregistrement et la récupération des modèles
func TestBlockRegistryModelMapping(t *testing.T) {
	registry := NewBlockRegistry()

	// Créer un modèle de test
	testModel := &BlockModel{
		Textures: map[string]string{
			"all": "minecraft:block/test",
		},
		Elements: []BlockElement{
			{
				From: []float32{0, 0, 0},
				To:   []float32{16, 16, 16},
				Faces: map[string]BlockFace{
					"north": {Texture: "minecraft:block/test"},
					"south": {Texture: "minecraft:block/test"},
					"east":  {Texture: "minecraft:block/test"},
					"west":  {Texture: "minecraft:block/test"},
					"up":    {Texture: "minecraft:block/test"},
					"down":  {Texture: "minecraft:block/test"},
				},
			},
		},
	}

	// Ajouter le modèle
	modelName := "minecraft:block/test"
	registry.Models[modelName] = testModel

	// Enregistrer le mapping
	registry.RegisterBlockModel(BlockTypeStone, modelName)

	// Tester la récupération
	retrievedModel, found := registry.GetModelForBlock(BlockTypeStone)
	if !found {
		t.Error("Expected to find model for BlockTypeStone")
	}

	if retrievedModel != testModel {
		t.Error("Expected retrieved model to be the same as the original")
	}

	// Tester un bloc non enregistré
	_, found = registry.GetModelForBlock(BlockTypeAir)
	if found {
		t.Error("Expected not to find model for BlockTypeAir")
	}
}

// TestBlockRegistryGetModel teste la récupération de modèles par nom
func TestBlockRegistryGetModel(t *testing.T) {
	registry := NewBlockRegistry()

	// Créer un modèle de test
	testModel := &BlockModel{
		Textures: map[string]string{
			"all": "minecraft:block/test",
		},
		Elements: []BlockElement{},
	}

	modelName := "minecraft:block/test"
	registry.Models[modelName] = testModel

	// Tester la récupération par nom
	retrievedModel, found := registry.GetModel(modelName)
	if !found {
		t.Errorf("Expected to find model '%s'", modelName)
	}

	if retrievedModel != testModel {
		t.Error("Expected retrieved model to be the same as the original")
	}

	// Tester un modèle inexistant
	_, found = registry.GetModel("minecraft:block/nonexistent")
	if found {
		t.Error("Expected not to find nonexistent model")
	}
}

// TestBlockRegistryRegisterBlockModel teste l'enregistrement des modèles de blocs
func TestBlockRegistryRegisterBlockModel(t *testing.T) {
	registry := NewBlockRegistry()

	// Tester l'enregistrement
	modelName := "minecraft:block/custom_stone"
	registry.RegisterBlockModel(BlockTypeStone, modelName)

	// Vérifier que le mapping a été créé
	if registeredName, exists := registry.blockModelsMap[BlockTypeStone]; !exists {
		t.Error("Expected BlockTypeStone to be registered")
	} else if registeredName != modelName {
		t.Errorf("Expected registered model name '%s', got '%s'", modelName, registeredName)
	}

	// Tester l'écrasement d'un enregistrement existant
	newModelName := "minecraft:block/another_stone"
	registry.RegisterBlockModel(BlockTypeStone, newModelName)

	if registeredName, exists := registry.blockModelsMap[BlockTypeStone]; !exists {
		t.Error("Expected BlockTypeStone to still be registered")
	} else if registeredName != newModelName {
		t.Errorf("Expected updated model name '%s', got '%s'", newModelName, registeredName)
	}
}

// TestBlockRegistryDefaultMappings teste les mappings par défaut
func TestBlockRegistryDefaultMappings(t *testing.T) {
	registry := NewBlockRegistry()

	// Vérifier que les mappings par défaut sont présents
	expectedMappings := map[BlockType]string{
		BlockTypeStone:       "minecraft:block/stone",
		BlockTypeDirt:        "minecraft:block/dirt",
		BlockTypeGrass:       "minecraft:block/grass_block",
		BlockTypeCobblestone: "minecraft:block/cobblestone",
		BlockTypeOakLog:      "minecraft:block/oak_log",
		BlockTypeOakLeaves:   "minecraft:block/oak_leaves",
		BlockTypeSand:        "minecraft:block/sand",
		BlockTypeWater:       "minecraft:block/water",
		BlockTypeBedrock:     "minecraft:block/bedrock",
	}

	for blockType, expectedModelName := range expectedMappings {
		if actualModelName, exists := registry.blockModelsMap[blockType]; !exists {
			t.Errorf("Expected %v to be mapped by default", blockType)
		} else if actualModelName != expectedModelName {
			t.Errorf("Expected %v to be mapped to '%s', got '%s'", blockType, expectedModelName, actualModelName)
		}
	}

	// Vérifier que BlockTypeAir n'est pas mappé (c'est normal)
	if _, exists := registry.blockModelsMap[BlockTypeAir]; exists {
		t.Error("Expected BlockTypeAir to not have a model mapping")
	}
}

// TestBlockRegistryEmptyRegistry teste le comportement avec un registry vide
func TestBlockRegistryEmptyRegistry(t *testing.T) {
	registry := &BlockRegistry{
		Models:         make(map[string]*BlockModel),
		RawModels:      make(map[string]*BlockModel),
		textureList:    make(map[string]struct{}),
		TextureCoords:  make(map[string]image.Rectangle),
		blockModelsMap: make(map[BlockType]string),
	}

	// Tester la récupération d'un modèle inexistant
	_, found := registry.GetModel("minecraft:block/nonexistent")
	if found {
		t.Error("Expected not to find model in empty registry")
	}

	// Tester la récupération d'un bloc sans mapping
	_, found = registry.GetModelForBlock(BlockTypeStone)
	if found {
		t.Error("Expected not to find model for unmapped block type")
	}
}

// TestBlockModelStructure teste la structure des modèles de blocs
func TestBlockModelStructure(t *testing.T) {
	// Créer un modèle complet
	model := &BlockModel{
		Parent: "minecraft:block/cube_all",
		Textures: map[string]string{
			"all": "minecraft:block/stone",
		},
		Elements: []BlockElement{
			{
				From: []float32{0, 0, 0},
				To:   []float32{16, 16, 16},
				Faces: map[string]BlockFace{
					"north": {
						UV:        []float32{0, 0, 16, 16},
						Texture:   "minecraft:block/stone",
						CullFace:  "north",
						TintIndex: 0,
					},
					"south": {
						UV:        []float32{0, 0, 16, 16},
						Texture:   "minecraft:block/stone",
						CullFace:  "south",
						TintIndex: 0,
					},
				},
			},
		},
	}

	// Vérifier la structure
	if model.Parent != "minecraft:block/cube_all" {
		t.Errorf("Expected parent 'minecraft:block/cube_all', got '%s'", model.Parent)
	}

	if len(model.Textures) != 1 {
		t.Errorf("Expected 1 texture, got %d", len(model.Textures))
	}

	if model.Textures["all"] != "minecraft:block/stone" {
		t.Errorf("Expected texture 'minecraft:block/stone', got '%s'", model.Textures["all"])
	}

	if len(model.Elements) != 1 {
		t.Errorf("Expected 1 element, got %d", len(model.Elements))
	}

	element := model.Elements[0]
	if len(element.From) != 3 {
		t.Errorf("Expected From to have 3 coordinates, got %d", len(element.From))
	}

	if len(element.To) != 3 {
		t.Errorf("Expected To to have 3 coordinates, got %d", len(element.To))
	}

	if len(element.Faces) != 2 {
		t.Errorf("Expected 2 faces, got %d", len(element.Faces))
	}

	// Vérifier une face spécifique
	northFace := element.Faces["north"]
	if len(northFace.UV) != 4 {
		t.Errorf("Expected UV to have 4 coordinates, got %d", len(northFace.UV))
	}

	if northFace.Texture != "minecraft:block/stone" {
		t.Errorf("Expected face texture 'minecraft:block/stone', got '%s'", northFace.Texture)
	}

	if northFace.CullFace != "north" {
		t.Errorf("Expected cullface 'north', got '%s'", northFace.CullFace)
	}
}
