package world

// BlockType est une énumération des types de blocs
type BlockType int

const (
	BlockTypeAir BlockType = iota
	BlockTypeDirt
	BlockTypeGrass
	BlockTypeStone
	BlockTypeCobblestone
	BlockTypeOakLog
	BlockTypeOakLeaves
	BlockTypeSand
	BlockTypeWater
	BlockTypeBedrock
	// ... ajoutez d'autres types de blocs ici
)

// Block représente un bloc unique dans le monde
type Block struct {
	Type BlockType
}

// GetTextureName retourne le nom de la texture pour ce bloc
func (b Block) GetTextureName(face string) string {
	switch b.Type {
	case BlockTypeStone:
		return "stone.png"
	case BlockTypeDirt:
		return "dirt.png"
	case BlockTypeGrass:
		switch face {
		case "top":
			return "grass_block_top.png"
		case "bottom":
			return "dirt.png"
		default:
			return "grass_block_side.png"
		}
	default:
		return "stone.png"
	}
}

// NewBlock crée un nouveau bloc
func NewBlock(blockType BlockType) Block {
	return Block{Type: blockType}
}

// NewAirBlock crée un bloc d'air
func NewAirBlock() Block {
	return NewBlock(BlockTypeAir)
}

// NewStoneBlock crée un bloc de pierre
func NewStoneBlock() Block {
	return NewBlock(BlockTypeStone)
}

// NewDirtBlock crée un bloc de terre
func NewDirtBlock() Block {
	return NewBlock(BlockTypeDirt)
}

// NewGrassBlock crée un bloc d'herbe
func NewGrassBlock() Block {
	return NewBlock(BlockTypeGrass)
}
