package mesh

import (
	"fmt"
	"testing"

	"github.com/go-gl/mathgl/mgl32"
)

func TestMeshState(t *testing.T) {
	// Test state string representation
	states := []MeshState{
		MeshStateUninitialized,
		MeshStateInitialized,
		MeshStateGenerating,
		MeshStateGenerated,
		MeshStateError,
	}

	expectedStrings := []string{
		"Uninitialized",
		"Initialized",
		"Generating",
		"Generated",
		"Error",
	}

	for i, state := range states {
		if state.String() != expectedStrings[i] {
			t.Errorf("Expected state %d to be %s, got %s", i, expectedStrings[i], state.String())
		}
	}
}

func TestNewMesh(t *testing.T) {
	mesh := NewMesh("test_mesh")

	if mesh.GetID() != "test_mesh" {
		t.Errorf("Expected mesh ID to be 'test_mesh', got %s", mesh.GetID())
	}

	if mesh.GetState() != MeshStateUninitialized {
		t.Errorf("Expected initial state to be Uninitialized, got %s", mesh.GetState())
	}

	if mesh.GetVertexCount() != 0 {
		t.Errorf("Expected initial vertex count to be 0, got %d", mesh.GetVertexCount())
	}

	if mesh.GetTriangleCount() != 0 {
		t.Errorf("Expected initial triangle count to be 0, got %d", mesh.GetTriangleCount())
	}

	if mesh.IsGenerated() {
		t.Error("Expected mesh to not be generated initially")
	}

	if mesh.GetLastError() != nil {
		t.Errorf("Expected no initial error, got %v", mesh.GetLastError())
	}
}

func TestMeshSetVertexData(t *testing.T) {
	mesh := NewMesh("test_mesh")

	// Test valid vertex data
	vertices := []float32{
		// Position (x,y,z), Normal (nx,ny,nz), TexCoord (u,v)
		0, 0, 0, 0, 0, 1, 0, 0,
		1, 0, 0, 0, 0, 1, 1, 0,
		1, 1, 0, 0, 0, 1, 1, 1,
	}
	indices := []uint32{0, 1, 2}

	err := mesh.SetVertexData(vertices, indices)
	if err != nil {
		t.Errorf("Expected no error setting vertex data, got %v", err)
	}

	if mesh.GetState() != MeshStateInitialized {
		t.Errorf("Expected state to be Initialized, got %s", mesh.GetState())
	}

	if mesh.GetVertexCount() != 3 {
		t.Errorf("Expected vertex count to be 3, got %d", mesh.GetVertexCount())
	}

	if mesh.GetTriangleCount() != 1 {
		t.Errorf("Expected triangle count to be 1, got %d", mesh.GetTriangleCount())
	}

	// Test that vertex data is copied
	retrievedVertices, retrievedIndices := mesh.GetVertexData()
	if len(retrievedVertices) != len(vertices) {
		t.Errorf("Expected %d vertices, got %d", len(vertices), len(retrievedVertices))
	}
	if len(retrievedIndices) != len(indices) {
		t.Errorf("Expected %d indices, got %d", len(indices), len(retrievedIndices))
	}
}

func TestMeshErrorHandling(t *testing.T) {
	mesh := NewMesh("test_mesh")

	// Test setting error
	testError := fmt.Errorf("test error")
	mesh.SetLastError(testError)

	if mesh.GetState() != MeshStateError {
		t.Errorf("Expected state to be Error after setting error, got %s", mesh.GetState())
	}

	if mesh.GetLastError() != testError {
		t.Errorf("Expected error to be %v, got %v", testError, mesh.GetLastError())
	}
}

func TestMeshBuilder(t *testing.T) {
	builder := NewMeshBuilder()

	// Test building a simple cube
	mesh, err := builder.BeginMesh("test_cube").
		AddCube(mgl32.Vec3{0, 0, 0}, 1.0).
		BuildMesh()

	if err != nil {
		t.Errorf("Expected no error building mesh, got %v", err)
	}

	if mesh.GetID() != "test_cube" {
		t.Errorf("Expected mesh ID to be 'test_cube', got %s", mesh.GetID())
	}

	if mesh.GetState() != MeshStateInitialized {
		t.Errorf("Expected mesh state to be Initialized, got %s", mesh.GetState())
	}

	// Cube should have 24 vertices (6 faces * 4 vertices per face)
	if mesh.GetVertexCount() != 24 {
		t.Errorf("Expected cube to have 24 vertices, got %d", mesh.GetVertexCount())
	}

	// Cube should have 12 triangles (6 faces * 2 triangles per face)
	if mesh.GetTriangleCount() != 12 {
		t.Errorf("Expected cube to have 12 triangles, got %d", mesh.GetTriangleCount())
	}
}

func TestMeshBuilderPlane(t *testing.T) {
	builder := NewMeshBuilder()

	// Test building a plane
	mesh, err := builder.BeginMesh("test_plane").
		AddPlane(mgl32.Vec3{0, 0, 0}, mgl32.Vec3{0, 1, 0}, 2.0).
		BuildMesh()

	if err != nil {
		t.Errorf("Expected no error building plane mesh, got %v", err)
	}

	// Plane should have 4 vertices
	if mesh.GetVertexCount() != 4 {
		t.Errorf("Expected plane to have 4 vertices, got %d", mesh.GetVertexCount())
	}

	// Plane should have 2 triangles
	if mesh.GetTriangleCount() != 2 {
		t.Errorf("Expected plane to have 2 triangles, got %d", mesh.GetTriangleCount())
	}
}

func TestMeshBuilderSphere(t *testing.T) {
	builder := NewMeshBuilder()

	// Test building a sphere
	mesh, err := builder.BeginMesh("test_sphere").
		AddSphere(mgl32.Vec3{0, 0, 0}, 1.0, 8).
		BuildMesh()

	if err != nil {
		t.Errorf("Expected no error building sphere mesh, got %v", err)
	}

	// Sphere with 8 segments should have (8+1)*(8+1) = 81 vertices
	expectedVertices := (8 + 1) * (8 + 1)
	if mesh.GetVertexCount() != expectedVertices {
		t.Errorf("Expected sphere to have %d vertices, got %d", expectedVertices, mesh.GetVertexCount())
	}

	// Sphere with 8 segments should have 8*8*2 = 128 triangles
	expectedTriangles := 8 * 8 * 2
	if mesh.GetTriangleCount() != expectedTriangles {
		t.Errorf("Expected sphere to have %d triangles, got %d", expectedTriangles, mesh.GetTriangleCount())
	}
}

func TestMeshBuilderWithoutBeginMesh(t *testing.T) {
	builder := NewMeshBuilder()

	// Try to build without calling BeginMesh
	_, err := builder.BuildMesh()
	if err == nil {
		t.Error("Expected error when building mesh without BeginMesh")
	}
}

func TestMeshBuilderVertexCount(t *testing.T) {
	builder := NewMeshBuilder()

	builder.BeginMesh("test")

	// Add some vertices
	builder.AddVertex(mgl32.Vec3{0, 0, 0}, mgl32.Vec3{0, 0, 1}, mgl32.Vec2{0, 0})
	builder.AddVertex(mgl32.Vec3{1, 0, 0}, mgl32.Vec3{0, 0, 1}, mgl32.Vec2{1, 0})
	builder.AddVertex(mgl32.Vec3{1, 1, 0}, mgl32.Vec3{0, 0, 1}, mgl32.Vec2{1, 1})

	if builder.GetVertexCount() != 3 {
		t.Errorf("Expected 3 vertices, got %d", builder.GetVertexCount())
	}

	// Add a triangle
	builder.AddTriangle(0, 1, 2)

	if builder.GetIndexCount() != 3 {
		t.Errorf("Expected 3 indices, got %d", builder.GetIndexCount())
	}
}
