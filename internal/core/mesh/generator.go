package mesh

import (
	"fmt"
	"log"
	"sync"

	"github.com/go-gl/gl/v4.6-core/gl"
)

// MeshGenerator handles OpenGL mesh generation and management
type MeshGenerator struct {
	// Thread safety
	mu sync.RWMutex
}

// NewMeshGenerator creates a new mesh generator
func NewMeshGenerator() *MeshGenerator {
	return &MeshGenerator{}
}

// GenerateOpenGLBuffers generates OpenGL buffers for a mesh
func (mg *MeshGenerator) GenerateOpenGLBuffers(mesh *Mesh) error {
	if mesh == nil {
		return fmt.Errorf("mesh cannot be nil")
	}

	if mesh.GetState() != MeshStateInitialized {
		return fmt.Errorf("mesh must be in Initialized state to generate OpenGL buffers")
	}

	mesh.SetState(MeshStateGenerating)
	defer func() {
		if mesh.GetLastError() != nil {
			mesh.SetState(MeshStateError)
		}
	}()

	vertices, indices := mesh.GetVertexData()

	// Clean up existing buffers if they exist
	mg.cleanupMeshBuffers(mesh)

	// Generate VAO
	var vao uint32
	gl.GenVertexArrays(1, &vao)
	if vao == 0 {
		err := fmt.Errorf("failed to generate VAO")
		mesh.SetLastError(err)
		return err
	}

	// Generate VBO
	var vbo uint32
	gl.GenBuffers(1, &vbo)
	if vbo == 0 {
		gl.DeleteVertexArrays(1, &vao)
		err := fmt.Errorf("failed to generate VBO")
		mesh.SetLastError(err)
		return err
	}

	// Generate EBO (if indices are provided)
	var ebo uint32
	if len(indices) > 0 {
		gl.GenBuffers(1, &ebo)
		if ebo == 0 {
			gl.DeleteVertexArrays(1, &vao)
			gl.DeleteBuffers(1, &vbo)
			err := fmt.Errorf("failed to generate EBO")
			mesh.SetLastError(err)
			return err
		}
	}

	// Bind VAO and set up vertex attributes
	gl.BindVertexArray(vao)

	// Bind VBO and upload vertex data
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*4, gl.Ptr(vertices), gl.STATIC_DRAW)

	// Set up vertex attributes based on vertex format
	format := mesh.vertexFormat
	offset := 0

	// Position attribute (location = 0)
	gl.VertexAttribPointer(0, int32(format.PositionSize), gl.FLOAT, false, int32(format.Stride), gl.PtrOffset(offset))
	gl.EnableVertexAttribArray(0)
	offset += format.PositionSize * 4

	// Normal attribute (location = 2)
	if format.NormalSize > 0 {
		gl.VertexAttribPointer(2, int32(format.NormalSize), gl.FLOAT, false, int32(format.Stride), gl.PtrOffset(offset))
		gl.EnableVertexAttribArray(2)
		offset += format.NormalSize * 4
	}

	// Texture coordinate attribute (location = 1)
	if format.TexCoordSize > 0 {
		gl.VertexAttribPointer(1, int32(format.TexCoordSize), gl.FLOAT, false, int32(format.Stride), gl.PtrOffset(offset))
		gl.EnableVertexAttribArray(1)
		offset += format.TexCoordSize * 4
	}

	// Color attribute (location = 3)
	if format.ColorSize > 0 {
		gl.VertexAttribPointer(3, int32(format.ColorSize), gl.FLOAT, false, int32(format.Stride), gl.PtrOffset(offset))
		gl.EnableVertexAttribArray(3)
	}

	// Bind EBO and upload index data (if indices are provided)
	if len(indices) > 0 {
		gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, ebo)
		gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, len(indices)*4, gl.Ptr(indices), gl.STATIC_DRAW)
	}

	// Unbind VAO
	gl.BindVertexArray(0)

	// Check for OpenGL errors
	if err := gl.GetError(); err != gl.NO_ERROR {
		mg.cleanupMeshBuffers(mesh)
		glErr := fmt.Errorf("OpenGL error during mesh generation: %v", err)
		mesh.SetLastError(glErr)
		return glErr
	}

	// Store buffer IDs in mesh
	mesh.mu.Lock()
	mesh.vao = vao
	mesh.vbo = vbo
	mesh.ebo = ebo
	mesh.state = MeshStateGenerated
	mesh.mu.Unlock()

	log.Printf("Generated OpenGL buffers for mesh %s: VAO=%d, VBO=%d, EBO=%d", mesh.GetID(), vao, vbo, ebo)
	return nil
}

// cleanupMeshBuffers cleans up existing OpenGL buffers for a mesh
func (mg *MeshGenerator) cleanupMeshBuffers(mesh *Mesh) {
	mesh.mu.Lock()
	defer mesh.mu.Unlock()

	if mesh.vao != 0 {
		gl.DeleteVertexArrays(1, &mesh.vao)
		mesh.vao = 0
	}
	if mesh.vbo != 0 {
		gl.DeleteBuffers(1, &mesh.vbo)
		mesh.vbo = 0
	}
	if mesh.ebo != 0 {
		gl.DeleteBuffers(1, &mesh.ebo)
		mesh.ebo = 0
	}
}

// CleanupMesh cleans up all OpenGL resources for a mesh
func (mg *MeshGenerator) CleanupMesh(mesh *Mesh) {
	if mesh == nil {
		return
	}

	mg.cleanupMeshBuffers(mesh)

	mesh.mu.Lock()
	mesh.vertices = nil
	mesh.indices = nil
	mesh.state = MeshStateUninitialized
	mesh.lastError = nil
	mesh.mu.Unlock()
}

// RenderMesh renders a mesh using OpenGL
func (mg *MeshGenerator) RenderMesh(mesh *Mesh, useIndices bool) error {
	if mesh == nil {
		return fmt.Errorf("mesh cannot be nil")
	}

	if !mesh.IsGenerated() {
		return fmt.Errorf("mesh must be generated before rendering")
	}

	vao, _, ebo := mesh.GetOpenGLBuffers()

	gl.BindVertexArray(vao)

	if useIndices && ebo != 0 {
		gl.DrawElements(gl.TRIANGLES, int32(mesh.GetTriangleCount()*3), gl.UNSIGNED_INT, nil)
	} else {
		gl.DrawArrays(gl.TRIANGLES, 0, int32(mesh.GetVertexCount()))
	}

	gl.BindVertexArray(0)

	// Check for OpenGL errors
	if err := gl.GetError(); err != gl.NO_ERROR {
		return fmt.Errorf("OpenGL error during mesh rendering: %v", err)
	}

	return nil
}
