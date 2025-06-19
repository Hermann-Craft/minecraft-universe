package mesh

import (
	"fmt"
	"testing"
	"time"
)

func TestNewMeshManager(t *testing.T) {
	config := DefaultMeshManagerConfig()
	manager := NewMeshManager(config)

	if manager == nil {
		t.Fatal("Expected mesh manager to be created")
	}

	stats := manager.GetStats()
	if stats.TotalMeshes != 0 {
		t.Errorf("Expected initial total meshes to be 0, got %d", stats.TotalMeshes)
	}

	if stats.GeneratedMeshes != 0 {
		t.Errorf("Expected initial generated meshes to be 0, got %d", stats.GeneratedMeshes)
	}
}

func TestMeshManagerCreateMesh(t *testing.T) {
	manager := NewMeshManager(DefaultMeshManagerConfig())

	// Test creating a valid mesh
	mesh, err := manager.CreateMesh("test_mesh")
	if err != nil {
		t.Errorf("Expected no error creating mesh, got %v", err)
	}

	if mesh.GetID() != "test_mesh" {
		t.Errorf("Expected mesh ID to be 'test_mesh', got %s", mesh.GetID())
	}

	// Test creating mesh with empty ID
	_, err = manager.CreateMesh("")
	if err == nil {
		t.Error("Expected error when creating mesh with empty ID")
	}

	// Test creating duplicate mesh
	_, err = manager.CreateMesh("test_mesh")
	if err == nil {
		t.Error("Expected error when creating duplicate mesh")
	}

	// Test mesh limit
	config := MeshManagerConfig{MaxMeshes: 1}
	limitedManager := NewMeshManager(config)
	_, err = limitedManager.CreateMesh("mesh1")
	if err != nil {
		t.Errorf("Expected no error creating first mesh, got %v", err)
	}

	_, err = limitedManager.CreateMesh("mesh2")
	if err == nil {
		t.Error("Expected error when exceeding mesh limit")
	}
}

func TestMeshManagerGetMesh(t *testing.T) {
	manager := NewMeshManager(DefaultMeshManagerConfig())

	// Create a mesh
	createdMesh, err := manager.CreateMesh("test_mesh")
	if err != nil {
		t.Fatalf("Failed to create mesh: %v", err)
	}

	// Get the mesh
	retrievedMesh, exists := manager.GetMesh("test_mesh")
	if !exists {
		t.Error("Expected mesh to exist")
	}

	if retrievedMesh != createdMesh {
		t.Error("Expected retrieved mesh to be the same as created mesh")
	}

	// Test getting non-existent mesh
	_, exists = manager.GetMesh("non_existent")
	if exists {
		t.Error("Expected non-existent mesh to not exist")
	}
}

func TestMeshManagerSetMeshData(t *testing.T) {
	manager := NewMeshManager(DefaultMeshManagerConfig())

	// Create a mesh
	_, err := manager.CreateMesh("test_mesh")
	if err != nil {
		t.Fatalf("Failed to create mesh: %v", err)
	}

	// Set vertex data
	vertices := []float32{
		0, 0, 0, 0, 0, 1, 0, 0,
		1, 0, 0, 0, 0, 1, 1, 0,
		1, 1, 0, 0, 0, 1, 1, 1,
	}
	indices := []uint32{0, 1, 2}

	err = manager.SetMeshData("test_mesh", vertices, indices)
	if err != nil {
		t.Errorf("Expected no error setting mesh data, got %v", err)
	}

	// Verify mesh state
	mesh, exists := manager.GetMesh("test_mesh")
	if !exists {
		t.Fatal("Mesh should exist")
	}

	if mesh.GetState() != MeshStateInitialized {
		t.Errorf("Expected mesh state to be Initialized, got %s", mesh.GetState())
	}

	if mesh.GetVertexCount() != 3 {
		t.Errorf("Expected vertex count to be 3, got %d", mesh.GetVertexCount())
	}

	if mesh.GetTriangleCount() != 1 {
		t.Errorf("Expected triangle count to be 1, got %d", mesh.GetTriangleCount())
	}
}

func TestMeshManagerDeleteMesh(t *testing.T) {
	manager := NewMeshManager(DefaultMeshManagerConfig())

	// Create a mesh
	_, err := manager.CreateMesh("test_mesh")
	if err != nil {
		t.Fatalf("Failed to create mesh: %v", err)
	}

	// Verify mesh exists
	_, exists := manager.GetMesh("test_mesh")
	if !exists {
		t.Fatal("Mesh should exist before deletion")
	}

	// Delete the mesh
	err = manager.DeleteMesh("test_mesh")
	if err != nil {
		t.Errorf("Expected no error deleting mesh, got %v", err)
	}

	// Verify mesh no longer exists
	_, exists = manager.GetMesh("test_mesh")
	if exists {
		t.Error("Mesh should not exist after deletion")
	}

	// Test deleting non-existent mesh
	err = manager.DeleteMesh("non_existent")
	if err == nil {
		t.Error("Expected error when deleting non-existent mesh")
	}
}

func TestMeshManagerGetAllMeshes(t *testing.T) {
	manager := NewMeshManager(DefaultMeshManagerConfig())

	// Create multiple meshes
	meshIDs := []string{"mesh1", "mesh2", "mesh3"}
	for _, id := range meshIDs {
		_, err := manager.CreateMesh(id)
		if err != nil {
			t.Fatalf("Failed to create mesh %s: %v", id, err)
		}
	}

	// Get all meshes
	meshes := manager.GetAllMeshes()

	if len(meshes) != len(meshIDs) {
		t.Errorf("Expected %d meshes, got %d", len(meshIDs), len(meshes))
	}

	// Verify all meshes exist
	for _, id := range meshIDs {
		if _, exists := meshes[id]; !exists {
			t.Errorf("Expected mesh %s to exist", id)
		}
	}
}

func TestMeshManagerCleanupAll(t *testing.T) {
	manager := NewMeshManager(DefaultMeshManagerConfig())

	// Create multiple meshes
	meshIDs := []string{"mesh1", "mesh2", "mesh3"}
	for _, id := range meshIDs {
		_, err := manager.CreateMesh(id)
		if err != nil {
			t.Fatalf("Failed to create mesh %s: %v", id, err)
		}
	}

	// Verify meshes exist
	meshes := manager.GetAllMeshes()
	if len(meshes) != len(meshIDs) {
		t.Errorf("Expected %d meshes before cleanup, got %d", len(meshIDs), len(meshes))
	}

	// Cleanup all
	manager.CleanupAll()

	// Verify all meshes are gone
	meshes = manager.GetAllMeshes()
	if len(meshes) != 0 {
		t.Errorf("Expected 0 meshes after cleanup, got %d", len(meshes))
	}

	// Verify stats are reset
	stats := manager.GetStats()
	if stats.TotalMeshes != 0 {
		t.Errorf("Expected total meshes to be 0 after cleanup, got %d", stats.TotalMeshes)
	}
}

func TestMeshManagerCleanupUnused(t *testing.T) {
	manager := NewMeshManager(DefaultMeshManagerConfig())

	// Create meshes
	_, err := manager.CreateMesh("good_mesh")
	if err != nil {
		t.Fatalf("Failed to create good mesh: %v", err)
	}

	errorMesh, err := manager.CreateMesh("error_mesh")
	if err != nil {
		t.Fatalf("Failed to create error mesh: %v", err)
	}

	// Set error state on one mesh
	errorMesh.SetLastError(fmt.Errorf("test error"))

	// Cleanup unused (should clean up error mesh)
	cleaned := manager.CleanupUnused(time.Hour)
	if cleaned != 1 {
		t.Errorf("Expected 1 mesh to be cleaned up, got %d", cleaned)
	}

	// Verify error mesh is gone but good mesh remains
	meshes := manager.GetAllMeshes()
	if len(meshes) != 1 {
		t.Errorf("Expected 1 mesh after cleanup, got %d", len(meshes))
	}

	if _, exists := meshes["good_mesh"]; !exists {
		t.Error("Expected good_mesh to remain after cleanup")
	}

	if _, exists := meshes["error_mesh"]; exists {
		t.Error("Expected error_mesh to be cleaned up")
	}
}

func TestMeshManagerStats(t *testing.T) {
	manager := NewMeshManager(DefaultMeshManagerConfig())

	// Create a mesh
	_, err := manager.CreateMesh("test_mesh")
	if err != nil {
		t.Fatalf("Failed to create mesh: %v", err)
	}

	// Check initial stats
	stats := manager.GetStats()
	if stats.TotalMeshes != 1 {
		t.Errorf("Expected total meshes to be 1, got %d", stats.TotalMeshes)
	}

	if stats.GeneratedMeshes != 0 {
		t.Errorf("Expected generated meshes to be 0, got %d", stats.GeneratedMeshes)
	}

	// Set mesh data and generate
	vertices := []float32{
		0, 0, 0, 0, 0, 1, 0, 0,
		1, 0, 0, 0, 0, 1, 1, 0,
		1, 1, 0, 0, 0, 1, 1, 1,
	}
	indices := []uint32{0, 1, 2}

	err = manager.SetMeshData("test_mesh", vertices, indices)
	if err != nil {
		t.Fatalf("Failed to set mesh data: %v", err)
	}

	// Note: We can't test actual OpenGL generation without a context,
	// but we can test the stats update logic by checking the current state
	stats = manager.GetStats()
	if stats.TotalMeshes != 1 {
		t.Errorf("Expected total meshes to still be 1, got %d", stats.TotalMeshes)
	}
}
