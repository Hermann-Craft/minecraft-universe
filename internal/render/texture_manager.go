package render

import (
	"fmt"
	"sync"
)

// textureManagerImpl implémentation du TextureManager
type textureManagerImpl struct {
	textures map[string]Texture
	mu       sync.RWMutex
}

// NewTextureManager crée un nouveau gestionnaire de textures
func NewTextureManager() *textureManagerImpl {
	return &textureManagerImpl{
		textures: make(map[string]Texture),
	}
}

// LoadTexture charge une texture depuis un fichier
func (tm *textureManagerImpl) LoadTexture(path string) (Texture, error) {
	// TODO: Implémenter le chargement de texture
	return nil, fmt.Errorf("texture loading not implemented yet")
}

// GetTexture récupère une texture par ID
func (tm *textureManagerImpl) GetTexture(id string) (Texture, bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	texture, exists := tm.textures[id]
	return texture, exists
}

// CreateAtlas crée un atlas de textures
func (tm *textureManagerImpl) CreateAtlas(textures []string) (TextureAtlas, error) {
	// TODO: Implémenter la création d'atlas
	return nil, fmt.Errorf("atlas creation not implemented yet")
}

// Cleanup nettoie les ressources
func (tm *textureManagerImpl) Cleanup() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	for _, texture := range tm.textures {
		texture.Cleanup()
	}
	tm.textures = make(map[string]Texture)
}
