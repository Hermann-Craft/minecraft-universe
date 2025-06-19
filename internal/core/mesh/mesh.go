package mesh

import (
	"fmt"
	"sync"
)

// MeshState represents the explicit state of a mesh
type MeshState int

const (
	MeshStateUninitialized MeshState = iota
	MeshStateInitialized
	MeshStateGenerating
	MeshStateGenerated
	MeshStateError
)

// String returns the string representation of the mesh state
func (s MeshState) String() string {
	switch s {
	case MeshStateUninitialized:
		return "Uninitialized"
	case MeshStateInitialized:
		return "Initialized"
	case MeshStateGenerating:
		return "Generating"
	case MeshStateGenerated:
		return "Generated"
	case MeshStateError:
		return "Error"
	default:
		return "Unknown"
	}
}

// Mesh represents a 3D mesh with OpenGL buffers
type Mesh struct {
	// Core state
	state MeshState
	id    string

	// Vertex data
	vertices []float32
	indices  []uint32

	// OpenGL buffers
	vao uint32
	vbo uint32
	ebo uint32

	// Mesh properties
	vertexCount   int
	triangleCount int
	vertexFormat  VertexFormat

	// Thread safety
	mu sync.RWMutex

	// Error tracking
	lastError error
}

// VertexFormat defines the structure of vertex data
type VertexFormat struct {
	PositionSize int // Number of position components (usually 3 for x,y,z)
	NormalSize   int // Number of normal components (usually 3 for nx,ny,nz)
	TexCoordSize int // Number of texture coordinate components (usually 2 for u,v)
	ColorSize    int // Number of color components (usually 4 for r,g,b,a)
	Stride       int // Total stride in bytes
}

// NewMesh creates a new mesh with the given ID
func NewMesh(id string) *Mesh {
	return &Mesh{
		state: MeshStateUninitialized,
		id:    id,
		vertexFormat: VertexFormat{
			PositionSize: 3,
			NormalSize:   3,
			TexCoordSize: 2,
			ColorSize:    0,
			Stride:       8 * 4, // 8 floats * 4 bytes per float
		},
	}
}

// GetID returns the mesh ID
func (c *Mesh) GetID() string {
	return c.id
}

// GetState returns the current mesh state
func (c *Mesh) GetState() MeshState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

// SetState sets the mesh state
func (c *Mesh) SetState(state MeshState) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state = state
}

// GetLastError returns the last error that occurred
func (c *Mesh) GetLastError() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastError
}

// SetLastError sets the last error
func (c *Mesh) SetLastError(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastError = err
	if err != nil {
		c.state = MeshStateError
	}
}

// SetVertexData sets the vertex and index data
func (c *Mesh) SetVertexData(vertices []float32, indices []uint32) error {
	if c.state == MeshStateGenerating {
		return fmt.Errorf("cannot set vertex data while mesh is generating")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.vertices = make([]float32, len(vertices))
	copy(c.vertices, vertices)

	c.indices = make([]uint32, len(indices))
	copy(c.indices, indices)

	c.vertexCount = len(vertices) / (c.vertexFormat.Stride / 4) // Convert bytes to floats
	c.triangleCount = len(indices) / 3

	c.state = MeshStateInitialized
	return nil
}

// GetVertexCount returns the number of vertices
func (c *Mesh) GetVertexCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.vertexCount
}

// GetTriangleCount returns the number of triangles
func (c *Mesh) GetTriangleCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.triangleCount
}

// GetVertexData returns a copy of the vertex data
func (c *Mesh) GetVertexData() ([]float32, []uint32) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	vertices := make([]float32, len(c.vertices))
	copy(vertices, c.vertices)

	indices := make([]uint32, len(c.indices))
	copy(indices, c.indices)

	return vertices, indices
}

// IsGenerated returns true if the mesh has been generated
func (c *Mesh) IsGenerated() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state == MeshStateGenerated && c.vao != 0
}

// GetOpenGLBuffers returns the OpenGL buffer IDs
func (c *Mesh) GetOpenGLBuffers() (vao, vbo, ebo uint32) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.vao, c.vbo, c.ebo
}
