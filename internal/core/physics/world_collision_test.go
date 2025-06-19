package physics

import (
	"testing"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/hermann-craft/unicube/internal/core/world"
	goecs "github.com/oneforx/go-ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWorldCollisionSystem(t *testing.T) {
	physicsManager := NewPhysicsManager()
	planet := world.NewPlanet(
		goecs.Identifier{Namespace: "test", Path: "planet"},
		mgl32.Vec3{0, 0, 0},
		mgl32.Vec3{3, 3, 3},
		world.ChunkSize{Width: 16, Height: 32, Depth: 16},
		42,
	)

	wcs := NewWorldCollisionSystem(physicsManager, planet)
	assert.NotNil(t, wcs)
	assert.Equal(t, physicsManager, wcs.physicsManager)
	assert.Equal(t, planet, wcs.world)
	assert.Equal(t, mgl32.Vec3{0, -9.81, 0}, wcs.gravity)
	assert.Equal(t, float32(0.1), wcs.stepSize)
}

func TestWorldCollisionSystem_SetGetGravity(t *testing.T) {
	physicsManager := NewPhysicsManager()
	planet := world.NewPlanet(
		goecs.Identifier{Namespace: "test", Path: "planet"},
		mgl32.Vec3{0, 0, 0},
		mgl32.Vec3{3, 3, 3},
		world.ChunkSize{Width: 16, Height: 32, Depth: 16},
		42,
	)

	wcs := NewWorldCollisionSystem(physicsManager, planet)

	// Test gravité par défaut
	assert.Equal(t, mgl32.Vec3{0, -9.81, 0}, wcs.GetGravity())

	// Test changement de gravité
	newGravity := mgl32.Vec3{0, -5.0, 0}
	wcs.SetGravity(newGravity)
	assert.Equal(t, newGravity, wcs.GetGravity())
}

func TestWorldCollisionSystem_RaycastBlock(t *testing.T) {
	physicsManager := NewPhysicsManager()
	planet := world.NewPlanet(
		goecs.Identifier{Namespace: "test", Path: "planet"},
		mgl32.Vec3{0, 0, 0},
		mgl32.Vec3{1, 1, 1},
		world.ChunkSize{Width: 16, Height: 32, Depth: 16},
		42,
	)

	// Initialiser la planète
	err := planet.InitializeChunks()
	require.NoError(t, err)

	wcs := NewWorldCollisionSystem(physicsManager, planet)

	// Test raycast sans monde
	wcs.world = nil
	chunk, bx, by, bz, pos, hit := wcs.RaycastBlock(mgl32.Vec3{0, 0, 0}, mgl32.Vec3{1, 0, 0}, 10.0)
	assert.Nil(t, chunk)
	assert.Equal(t, 0, bx)
	assert.Equal(t, 0, by)
	assert.Equal(t, 0, bz)
	assert.Equal(t, mgl32.Vec3{}, pos)
	assert.False(t, hit)

	// Restaurer le monde
	wcs.world = planet

	// Test raycast dans l'air (pas de hit)
	chunk, bx, by, bz, pos, hit = wcs.RaycastBlock(mgl32.Vec3{0, 0, 0}, mgl32.Vec3{1, 0, 0}, 10.0)
	assert.False(t, hit)

	// Placer un bloc et tester le raycast
	err = wcs.PlaceBlock(mgl32.Vec3{5, 0, 0}, world.BlockTypeStone)
	require.NoError(t, err)

	chunk, bx, by, bz, pos, hit = wcs.RaycastBlock(mgl32.Vec3{0, 0, 0}, mgl32.Vec3{1, 0, 0}, 10.0)
	assert.True(t, hit)
	assert.NotNil(t, chunk)
	assert.Equal(t, 5, bx)
	assert.Equal(t, 0, by)
	assert.Equal(t, 0, bz)
}

func TestWorldCollisionSystem_CheckCollision(t *testing.T) {
	physicsManager := NewPhysicsManager()
	planet := world.NewPlanet(
		goecs.Identifier{Namespace: "test", Path: "planet"},
		mgl32.Vec3{0, 0, 0},
		mgl32.Vec3{1, 1, 1},
		world.ChunkSize{Width: 16, Height: 32, Depth: 16},
		42,
	)

	err := planet.InitializeChunks()
	require.NoError(t, err)

	wcs := NewWorldCollisionSystem(physicsManager, planet)

	// Test sans objet
	hasCollision, normal := wcs.CheckCollision(nil)
	assert.False(t, hasCollision)
	assert.Equal(t, mgl32.Vec3{}, normal)

	// Créer un objet physique
	obj := NewPhysicsObject("test")
	obj.SetPosition(mgl32.Vec3{0, 0, 0})
	obj.SetCollider(NewBoxCollider(mgl32.Vec3{1, 1, 1}))

	// Test sans collision
	hasCollision, normal = wcs.CheckCollision(obj)
	assert.False(t, hasCollision)

	// Placer un bloc et tester la collision
	err = wcs.PlaceBlock(mgl32.Vec3{0, 0, 0}, world.BlockTypeStone)
	require.NoError(t, err)

	hasCollision, normal = wcs.CheckCollision(obj)
	assert.True(t, hasCollision)
	assert.NotEqual(t, mgl32.Vec3{}, normal)
}

func TestWorldCollisionSystem_ResolveCollision(t *testing.T) {
	physicsManager := NewPhysicsManager()
	planet := world.NewPlanet(
		goecs.Identifier{Namespace: "test", Path: "planet"},
		mgl32.Vec3{0, 0, 0},
		mgl32.Vec3{1, 1, 1},
		world.ChunkSize{Width: 16, Height: 32, Depth: 16},
		42,
	)

	wcs := NewWorldCollisionSystem(physicsManager, planet)

	// Test sans objet
	wcs.ResolveCollision(nil, mgl32.Vec3{0, 1, 0})

	// Créer un objet avec vélocité qui a une composante vers le haut
	obj := NewPhysicsObject("test")
	obj.SetPosition(mgl32.Vec3{0, 0, 0})
	obj.SetVelocity(mgl32.Vec3{1, 1, 0}) // Vélocité avec composante Y positive

	// Résoudre une collision avec une normale vers le bas
	normal := mgl32.Vec3{0, -1, 0} // Collision depuis le haut (normale vers le bas)
	wcs.ResolveCollision(obj, normal)

	// La vélocité devrait être ajustée (composante Y supprimée, friction appliquée)
	velocity := obj.GetVelocity()
	// Avec une normale {0, -1, 0} et une vélocité initiale {1, 1, 0},
	// la vélocité finale devrait être {0.8, 0, 0} (composante Y supprimée, friction appliquée)
	expectedVelocity := mgl32.Vec3{0.8, 0, 0}
	assert.Equal(t, expectedVelocity, velocity)
}

func TestWorldCollisionSystem_PlaceRemoveBlock(t *testing.T) {
	physicsManager := NewPhysicsManager()
	planet := world.NewPlanet(
		goecs.Identifier{Namespace: "test", Path: "planet"},
		mgl32.Vec3{0, 0, 0},
		mgl32.Vec3{1, 1, 1},
		world.ChunkSize{Width: 16, Height: 32, Depth: 16},
		42,
	)

	err := planet.InitializeChunks()
	require.NoError(t, err)

	wcs := NewWorldCollisionSystem(physicsManager, planet)

	// Test placement de bloc
	position := mgl32.Vec3{5, 10, 15}
	err = wcs.PlaceBlock(position, world.BlockTypeStone)
	assert.NoError(t, err)

	// Vérifier que le bloc a été placé
	block, err := wcs.GetBlockAt(position)
	assert.NoError(t, err)
	assert.Equal(t, world.BlockTypeStone, block.Type)

	// Test suppression de bloc
	err = wcs.RemoveBlock(position)
	assert.NoError(t, err)

	// Vérifier que le bloc a été supprimé
	block, err = wcs.GetBlockAt(position)
	assert.NoError(t, err)
	assert.Equal(t, world.BlockTypeAir, block.Type)
}

func TestWorldCollisionSystem_GetBlockAt(t *testing.T) {
	physicsManager := NewPhysicsManager()
	planet := world.NewPlanet(
		goecs.Identifier{Namespace: "test", Path: "planet"},
		mgl32.Vec3{0, 0, 0},
		mgl32.Vec3{1, 1, 1},
		world.ChunkSize{Width: 16, Height: 32, Depth: 16},
		42,
	)

	err := planet.InitializeChunks()
	require.NoError(t, err)

	wcs := NewWorldCollisionSystem(physicsManager, planet)

	// Test sans monde
	wcs.world = nil
	block, err := wcs.GetBlockAt(mgl32.Vec3{0, 0, 0})
	assert.Error(t, err)
	assert.Nil(t, block)

	// Restaurer le monde
	wcs.world = planet

	// Test position hors limites
	block, err = wcs.GetBlockAt(mgl32.Vec3{100, 100, 100})
	assert.Error(t, err)
	assert.Nil(t, block)

	// Test position valide
	block, err = wcs.GetBlockAt(mgl32.Vec3{0, 0, 0})
	assert.NoError(t, err)
	assert.NotNil(t, block)
	assert.Equal(t, world.BlockTypeAir, block.Type)
}

func TestWorldCollisionSystem_Update(t *testing.T) {
	physicsManager := NewPhysicsManager()
	planet := world.NewPlanet(
		goecs.Identifier{Namespace: "test", Path: "planet"},
		mgl32.Vec3{0, 0, 0},
		mgl32.Vec3{1, 1, 1},
		world.ChunkSize{Width: 16, Height: 32, Depth: 16},
		42,
	)

	err := planet.InitializeChunks()
	require.NoError(t, err)

	wcs := NewWorldCollisionSystem(physicsManager, planet)

	// Test sans monde
	wcs.world = nil
	wcs.Update(0.016) // Pas d'erreur attendue

	// Restaurer le monde
	wcs.world = planet

	// Test sans objet physique
	wcs.Update(0.016) // Pas d'erreur attendue

	// Ajouter un objet physique
	obj := NewPhysicsObject("test")
	obj.SetPosition(mgl32.Vec3{0, 10, 0})
	obj.SetCollider(NewBoxCollider(mgl32.Vec3{1, 1, 1}))
	obj.SetState(PhysicsStateDynamic)
	physicsManager.AddObject(obj)

	// Placer un bloc pour créer une collision
	wcs.PlaceBlock(mgl32.Vec3{0, 0, 0}, world.BlockTypeStone)

	// Mettre à jour
	wcs.Update(0.016)

	// L'objet devrait avoir été affecté par la gravité et les collisions
	velocity := obj.GetVelocity()
	assert.NotEqual(t, mgl32.Vec3{0, 0, 0}, velocity)
}

func TestCalculateCollisionNormal(t *testing.T) {
	physicsManager := NewPhysicsManager()
	planet := world.NewPlanet(
		goecs.Identifier{Namespace: "test", Path: "planet"},
		mgl32.Vec3{0, 0, 0},
		mgl32.Vec3{1, 1, 1},
		world.ChunkSize{Width: 16, Height: 32, Depth: 16},
		42,
	)

	wcs := NewWorldCollisionSystem(physicsManager, planet)

	// Test collision depuis la droite (objet à droite du bloc)
	objBox := BoundingBox{
		Min: mgl32.Vec3{1, 0, 0},
		Max: mgl32.Vec3{2, 1, 1},
	}
	blockBox := BoundingBox{
		Min: mgl32.Vec3{0, 0, 0},
		Max: mgl32.Vec3{1, 1, 1},
	}

	normal := wcs.calculateCollisionNormal(objBox, blockBox)
	// L'objet est à droite du bloc, donc la normale pointe vers la gauche (négatif X)
	assert.Equal(t, mgl32.Vec3{-1, 0, 0}, normal)

	// Test collision depuis le bas (objet au-dessus du bloc)
	objBox = BoundingBox{
		Min: mgl32.Vec3{0, 1, 0},
		Max: mgl32.Vec3{1, 2, 1},
	}
	blockBox = BoundingBox{
		Min: mgl32.Vec3{0, 0, 0},
		Max: mgl32.Vec3{1, 1, 1},
	}

	normal = wcs.calculateCollisionNormal(objBox, blockBox)
	// L'objet est au-dessus du bloc, donc la normale pointe vers le bas (négatif Y)
	assert.Equal(t, mgl32.Vec3{0, -1, 0}, normal)
}

func TestMin(t *testing.T) {
	assert.Equal(t, float32(1.0), min(1.0, 2.0))
	assert.Equal(t, float32(1.0), min(2.0, 1.0))
	assert.Equal(t, float32(1.0), min(1.0, 1.0))
	assert.Equal(t, float32(-2.0), min(-2.0, 1.0))
}
