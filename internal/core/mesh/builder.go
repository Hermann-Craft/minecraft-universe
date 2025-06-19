package mesh

import (
	"fmt"
	"math"

	"github.com/go-gl/mathgl/mgl32"
)

// MeshBuilder provides utilities for building meshes
type MeshBuilder struct {
	// Builder state
	currentMesh *Mesh
	vertices    []float32
	indices     []uint32
	indexOffset uint32
}

// NewMeshBuilder creates a new mesh builder
func NewMeshBuilder() *MeshBuilder {
	return &MeshBuilder{
		vertices: make([]float32, 0),
		indices:  make([]uint32, 0),
	}
}

// BeginMesh starts building a new mesh
func (mb *MeshBuilder) BeginMesh(id string) *MeshBuilder {
	mb.currentMesh = NewMesh(id)
	mb.vertices = make([]float32, 0)
	mb.indices = make([]uint32, 0)
	mb.indexOffset = 0
	return mb
}

// AddVertex adds a vertex with position, normal, and texture coordinates
func (mb *MeshBuilder) AddVertex(pos mgl32.Vec3, normal mgl32.Vec3, texCoord mgl32.Vec2) *MeshBuilder {
	// Position (x, y, z)
	mb.vertices = append(mb.vertices, pos.X(), pos.Y(), pos.Z())

	// Normal (nx, ny, nz)
	mb.vertices = append(mb.vertices, normal.X(), normal.Y(), normal.Z())

	// Texture coordinates (u, v)
	mb.vertices = append(mb.vertices, texCoord.X(), texCoord.Y())

	return mb
}

// AddTriangle adds a triangle using vertex indices
func (mb *MeshBuilder) AddTriangle(i0, i1, i2 uint32) *MeshBuilder {
	mb.indices = append(mb.indices, mb.indexOffset+i0, mb.indexOffset+i1, mb.indexOffset+i2)
	return mb
}

// AddQuad adds a quad using vertex indices (two triangles)
func (mb *MeshBuilder) AddQuad(i0, i1, i2, i3 uint32) *MeshBuilder {
	mb.AddTriangle(i0, i1, i2)
	mb.AddTriangle(i0, i2, i3)
	return mb
}

// AddCube adds a complete cube mesh
func (mb *MeshBuilder) AddCube(center mgl32.Vec3, size float32) *MeshBuilder {
	halfSize := size * 0.5

	// Define cube vertices (8 vertices)
	vertices := []mgl32.Vec3{
		{center.X() - halfSize, center.Y() - halfSize, center.Z() - halfSize}, // 0: bottom-left-back
		{center.X() + halfSize, center.Y() - halfSize, center.Z() - halfSize}, // 1: bottom-right-back
		{center.X() + halfSize, center.Y() + halfSize, center.Z() - halfSize}, // 2: top-right-back
		{center.X() - halfSize, center.Y() + halfSize, center.Z() - halfSize}, // 3: top-left-back
		{center.X() - halfSize, center.Y() - halfSize, center.Z() + halfSize}, // 4: bottom-left-front
		{center.X() + halfSize, center.Y() - halfSize, center.Z() + halfSize}, // 5: bottom-right-front
		{center.X() + halfSize, center.Y() + halfSize, center.Z() + halfSize}, // 6: top-right-front
		{center.X() - halfSize, center.Y() + halfSize, center.Z() + halfSize}, // 7: top-left-front
	}

	// Define face normals
	normals := []mgl32.Vec3{
		{0, 0, -1}, // Back face
		{1, 0, 0},  // Right face
		{0, 0, 1},  // Front face
		{-1, 0, 0}, // Left face
		{0, 1, 0},  // Top face
		{0, -1, 0}, // Bottom face
	}

	// Define texture coordinates for each face
	texCoords := []mgl32.Vec2{
		{0, 0}, {1, 0}, {1, 1}, {0, 1}, // Standard quad texture coordinates
	}

	// Add vertices for each face
	faceIndices := [][]uint32{
		{0, 1, 2, 3}, // Back face
		{1, 5, 6, 2}, // Right face
		{5, 4, 7, 6}, // Front face
		{4, 0, 3, 7}, // Left face
		{3, 2, 6, 7}, // Top face
		{4, 5, 1, 0}, // Bottom face
	}

	// Add each face
	for faceIdx, face := range faceIndices {
		normal := normals[faceIdx]

		// Add vertices for this face
		for i, vertexIdx := range face {
			pos := vertices[vertexIdx]
			texCoord := texCoords[i]
			mb.AddVertex(pos, normal, texCoord)
		}

		// Add triangles for this face
		mb.AddQuad(0, 1, 2, 3)

		// Update index offset for next face
		mb.indexOffset += 4
	}

	return mb
}

// AddPlane adds a plane mesh
func (mb *MeshBuilder) AddPlane(center mgl32.Vec3, normal mgl32.Vec3, size float32) *MeshBuilder {
	halfSize := size * 0.5

	// Calculate tangent and bitangent vectors
	tangent := mgl32.Vec3{1, 0, 0}
	if math.Abs(float64(normal.Dot(tangent))) > 0.9 {
		tangent = mgl32.Vec3{0, 1, 0}
	}
	bitangent := normal.Cross(tangent).Normalize()
	tangent = bitangent.Cross(normal).Normalize()

	// Calculate corner vertices
	corners := []mgl32.Vec3{
		center.Add(tangent.Mul(-halfSize)).Add(bitangent.Mul(-halfSize)),
		center.Add(tangent.Mul(halfSize)).Add(bitangent.Mul(-halfSize)),
		center.Add(tangent.Mul(halfSize)).Add(bitangent.Mul(halfSize)),
		center.Add(tangent.Mul(-halfSize)).Add(bitangent.Mul(halfSize)),
	}

	// Add vertices
	for _, corner := range corners {
		mb.AddVertex(corner, normal, mgl32.Vec2{0, 0}) // Texture coordinates will be set by caller
	}

	// Add quad
	mb.AddQuad(0, 1, 2, 3)
	mb.indexOffset += 4

	return mb
}

// AddSphere adds a sphere mesh
func (mb *MeshBuilder) AddSphere(center mgl32.Vec3, radius float32, segments int) *MeshBuilder {
	if segments < 3 {
		segments = 3
	}

	// Generate sphere vertices
	for lat := 0; lat <= segments; lat++ {
		theta := float32(lat) * math.Pi / float32(segments)
		sinTheta := float32(math.Sin(float64(theta)))
		cosTheta := float32(math.Cos(float64(theta)))

		for lon := 0; lon <= segments; lon++ {
			phi := float32(lon) * 2 * math.Pi / float32(segments)
			sinPhi := float32(math.Sin(float64(phi)))
			cosPhi := float32(math.Cos(float64(phi)))

			x := cosPhi * sinTheta
			y := cosTheta
			z := sinPhi * sinTheta

			pos := center.Add(mgl32.Vec3{x, y, z}.Mul(radius))
			normal := mgl32.Vec3{x, y, z}.Normalize()
			texCoord := mgl32.Vec2{
				float32(lon) / float32(segments),
				float32(lat) / float32(segments),
			}

			mb.AddVertex(pos, normal, texCoord)
		}
	}

	// Generate indices
	for lat := 0; lat < segments; lat++ {
		for lon := 0; lon < segments; lon++ {
			current := uint32(lat*(segments+1) + lon)
			next := current + uint32(segments+1)

			mb.AddTriangle(current, next, current+1)
			mb.AddTriangle(next, next+1, current+1)
		}
	}

	mb.indexOffset += uint32((segments + 1) * (segments + 1))

	return mb
}

// BuildMesh builds and returns the final mesh
func (mb *MeshBuilder) BuildMesh() (*Mesh, error) {
	if mb.currentMesh == nil {
		return nil, fmt.Errorf("no mesh started, call BeginMesh first")
	}

	// Set vertex data
	err := mb.currentMesh.SetVertexData(mb.vertices, mb.indices)
	if err != nil {
		return nil, fmt.Errorf("failed to set vertex data: %w", err)
	}

	mesh := mb.currentMesh
	mb.currentMesh = nil
	mb.vertices = nil
	mb.indices = nil
	mb.indexOffset = 0

	return mesh, nil
}

// GetVertexCount returns the current vertex count
func (mb *MeshBuilder) GetVertexCount() int {
	return len(mb.vertices) / 8 // 8 floats per vertex (pos + normal + texCoord)
}

// GetIndexCount returns the current index count
func (mb *MeshBuilder) GetIndexCount() int {
	return len(mb.indices)
}
