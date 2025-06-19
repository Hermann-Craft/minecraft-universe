package render

import (
	"time"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/hermann-craft/unicube/internal/core/geom"
	"github.com/hermann-craft/unicube/internal/core/mesh"
)

// Shader interface pour les shaders
type Shader interface {
	Activate()
	SetMat4(name string, value mgl32.Mat4)
	SetVec3(name string, value mgl32.Vec3)
	SetFloat(name string, value float32)
	SetInt(name string, value int32)
	GetID() uint32
	Cleanup()
}

// MeshManager interface pour le gestionnaire de meshes
type MeshManager interface {
	CreateMesh(id string) (*mesh.Mesh, error)
	GetMesh(id string) (*mesh.Mesh, bool)
	GenerateMesh(id string) error
	RenderMesh(id string, useIndices bool) error
	SetMeshData(id string, vertices []float32, indices []uint32) error
	DeleteMesh(id string) error
	GetStats() mesh.MeshManagerStats
	GetAllMeshes() map[string]*mesh.Mesh
	CleanupAll()
	CleanupUnused(maxAge time.Duration) int
}

// TextureManager interface pour le gestionnaire de textures
type TextureManager interface {
	LoadTexture(path string) (Texture, error)
	GetTexture(id string) (Texture, bool)
	CreateAtlas(textures []string) (TextureAtlas, error)
	Cleanup()
}

// Mesh interface pour les meshes (legacy - use mesh.Mesh instead)
type Mesh interface {
	Render()
	GetVertexCount() int
	GetTriangleCount() int
	Cleanup()
}

// Texture interface pour les textures
type Texture interface {
	Bind(unit uint32)
	GetID() uint32
	GetWidth() int
	GetHeight() int
	Cleanup()
}

// TextureAtlas interface pour les atlas de textures
type TextureAtlas interface {
	Bind(unit uint32)
	GetTextureCoords(textureName string) (float32, float32, float32, float32)
	GetID() uint32
	Cleanup()
}

// Frustum interface pour le frustum de caméra
type Frustum interface {
	IsPointVisible(point mgl32.Vec3) bool
	IsBoxVisible(box geom.BoundingBox) bool
	Update(view, projection mgl32.Mat4)
}
