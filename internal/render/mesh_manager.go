package render

import (
	"time"

	"github.com/hermann-craft/unicube/internal/core/mesh"
)

// meshManagerImpl implémentation du MeshManager
type meshManagerImpl struct {
	manager *mesh.MeshManager
}

// NewMeshManager crée un nouveau gestionnaire de meshes
func NewMeshManager() *meshManagerImpl {
	config := mesh.DefaultMeshManagerConfig()
	manager := mesh.NewMeshManager(config)
	return &meshManagerImpl{
		manager: manager,
	}
}

// CreateMesh crée un nouveau mesh
func (mm *meshManagerImpl) CreateMesh(id string) (*mesh.Mesh, error) {
	return mm.manager.CreateMesh(id)
}

// GetMesh récupère un mesh par ID
func (mm *meshManagerImpl) GetMesh(id string) (*mesh.Mesh, bool) {
	return mm.manager.GetMesh(id)
}

// GenerateMesh génère les buffers OpenGL pour un mesh
func (mm *meshManagerImpl) GenerateMesh(id string) error {
	return mm.manager.GenerateMesh(id)
}

// RenderMesh rend un mesh
func (mm *meshManagerImpl) RenderMesh(id string, useIndices bool) error {
	return mm.manager.RenderMesh(id, useIndices)
}

// SetMeshData définit les données de vertex et d'index pour un mesh
func (mm *meshManagerImpl) SetMeshData(id string, vertices []float32, indices []uint32) error {
	return mm.manager.SetMeshData(id, vertices, indices)
}

// DeleteMesh supprime un mesh et nettoie ses ressources
func (mm *meshManagerImpl) DeleteMesh(id string) error {
	return mm.manager.DeleteMesh(id)
}

// GetStats retourne les statistiques du gestionnaire de meshes
func (mm *meshManagerImpl) GetStats() mesh.MeshManagerStats {
	return mm.manager.GetStats()
}

// GetAllMeshes retourne tous les meshes
func (mm *meshManagerImpl) GetAllMeshes() map[string]*mesh.Mesh {
	return mm.manager.GetAllMeshes()
}

// CleanupAll nettoie tous les meshes
func (mm *meshManagerImpl) CleanupAll() {
	mm.manager.CleanupAll()
}

// CleanupUnused nettoie les meshes non utilisés
func (mm *meshManagerImpl) CleanupUnused(maxAge time.Duration) int {
	return mm.manager.CleanupUnused(maxAge)
}
