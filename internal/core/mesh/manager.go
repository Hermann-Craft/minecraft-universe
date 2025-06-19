package mesh

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// MeshManager manages mesh creation, generation, and cleanup
type MeshManager struct {
	// Mesh storage
	meshes map[string]*Mesh

	// Mesh generator
	generator *MeshGenerator

	// Thread safety
	mu sync.RWMutex

	// Statistics
	stats MeshManagerStats

	// Configuration
	config MeshManagerConfig
}

// MeshManagerStats tracks mesh manager statistics
type MeshManagerStats struct {
	TotalMeshes     int
	GeneratedMeshes int
	ErrorMeshes     int
	LastUpdate      time.Time
}

// MeshManagerConfig holds mesh manager configuration
type MeshManagerConfig struct {
	MaxMeshes       int
	AutoCleanup     bool
	CleanupInterval time.Duration
	GenerateTimeout time.Duration
}

// DefaultMeshManagerConfig returns default configuration
func DefaultMeshManagerConfig() MeshManagerConfig {
	return MeshManagerConfig{
		MaxMeshes:       1000,
		AutoCleanup:     true,
		CleanupInterval: 5 * time.Minute,
		GenerateTimeout: 30 * time.Second,
	}
}

// NewMeshManager creates a new mesh manager
func NewMeshManager(config MeshManagerConfig) *MeshManager {
	return &MeshManager{
		meshes:    make(map[string]*Mesh),
		generator: NewMeshGenerator(),
		config:    config,
		stats:     MeshManagerStats{LastUpdate: time.Now()},
	}
}

// CreateMesh creates a new mesh with the given ID
func (mm *MeshManager) CreateMesh(id string) (*Mesh, error) {
	if id == "" {
		return nil, fmt.Errorf("mesh ID cannot be empty")
	}

	mm.mu.Lock()
	defer mm.mu.Unlock()

	// Check if mesh already exists
	if _, exists := mm.meshes[id]; exists {
		return nil, fmt.Errorf("mesh with ID %s already exists", id)
	}

	// Check mesh limit
	if len(mm.meshes) >= mm.config.MaxMeshes {
		return nil, fmt.Errorf("mesh limit reached (%d)", mm.config.MaxMeshes)
	}

	// Create new mesh
	mesh := NewMesh(id)
	mm.meshes[id] = mesh
	mm.stats.TotalMeshes++
	mm.stats.LastUpdate = time.Now()

	log.Printf("Created mesh %s (total: %d)", id, mm.stats.TotalMeshes)
	return mesh, nil
}

// GetMesh retrieves a mesh by ID
func (mm *MeshManager) GetMesh(id string) (*Mesh, bool) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	mesh, exists := mm.meshes[id]
	return mesh, exists
}

// GenerateMesh generates OpenGL buffers for a mesh
func (mm *MeshManager) GenerateMesh(id string) error {
	mesh, exists := mm.GetMesh(id)
	if !exists {
		return fmt.Errorf("mesh %s not found", id)
	}

	if mesh.IsGenerated() {
		return nil // Already generated
	}

	if mesh.GetState() == MeshStateError {
		return fmt.Errorf("mesh %s is in error state: %v", id, mesh.GetLastError())
	}

	// Generate OpenGL buffers
	err := mm.generator.GenerateOpenGLBuffers(mesh)
	if err != nil {
		mm.stats.ErrorMeshes++
		return fmt.Errorf("failed to generate mesh %s: %w", id, err)
	}

	mm.mu.Lock()
	mm.stats.GeneratedMeshes++
	mm.stats.LastUpdate = time.Now()
	mm.mu.Unlock()

	log.Printf("Generated mesh %s (generated: %d)", id, mm.stats.GeneratedMeshes)
	return nil
}

// RenderMesh renders a mesh
func (mm *MeshManager) RenderMesh(id string, useIndices bool) error {
	mesh, exists := mm.GetMesh(id)
	if !exists {
		return fmt.Errorf("mesh %s not found", id)
	}

	if !mesh.IsGenerated() {
		return fmt.Errorf("mesh %s is not generated", id)
	}

	return mm.generator.RenderMesh(mesh, useIndices)
}

// SetMeshData sets vertex and index data for a mesh
func (mm *MeshManager) SetMeshData(id string, vertices []float32, indices []uint32) error {
	mesh, exists := mm.GetMesh(id)
	if !exists {
		return fmt.Errorf("mesh %s not found", id)
	}

	return mesh.SetVertexData(vertices, indices)
}

// DeleteMesh deletes a mesh and cleans up its resources
func (mm *MeshManager) DeleteMesh(id string) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	mesh, exists := mm.meshes[id]
	if !exists {
		return fmt.Errorf("mesh %s not found", id)
	}

	// Clean up OpenGL resources
	mm.generator.CleanupMesh(mesh)

	// Remove from map
	delete(mm.meshes, id)
	mm.stats.TotalMeshes--
	mm.stats.LastUpdate = time.Now()

	log.Printf("Deleted mesh %s (total: %d)", id, mm.stats.TotalMeshes)
	return nil
}

// GetStats returns mesh manager statistics
func (mm *MeshManager) GetStats() MeshManagerStats {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	return mm.stats
}

// GetAllMeshes returns all meshes
func (mm *MeshManager) GetAllMeshes() map[string]*Mesh {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	// Return a copy to avoid race conditions
	meshes := make(map[string]*Mesh)
	for id, mesh := range mm.meshes {
		meshes[id] = mesh
	}
	return meshes
}

// CleanupAll cleans up all meshes
func (mm *MeshManager) CleanupAll() {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	for id, mesh := range mm.meshes {
		mm.generator.CleanupMesh(mesh)
		log.Printf("Cleaned up mesh %s", id)
	}

	mm.meshes = make(map[string]*Mesh)
	mm.stats = MeshManagerStats{LastUpdate: time.Now()}

	log.Printf("Cleaned up all meshes")
}

// CleanupUnused cleans up meshes that haven't been used recently
func (mm *MeshManager) CleanupUnused(maxAge time.Duration) int {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	cleaned := 0
	now := time.Now()

	for id, mesh := range mm.meshes {
		// For now, we'll clean up meshes in error state
		// In a real implementation, you'd track last usage time
		if mesh.GetState() == MeshStateError {
			mm.generator.CleanupMesh(mesh)
			delete(mm.meshes, id)
			cleaned++
			log.Printf("Cleaned up unused mesh %s", id)
		}
	}

	mm.stats.TotalMeshes -= cleaned
	mm.stats.LastUpdate = now

	return cleaned
}
