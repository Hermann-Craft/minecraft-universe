package physics

import (
	"fmt"
	"log"
	"math"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/hermann-craft/unicube/internal/core/geom"
	"github.com/hermann-craft/unicube/internal/core/world"
)

// WorldCollisionSystem gère les collisions avec le monde de blocs
type WorldCollisionSystem struct {
	physicsManager    *PhysicsManager
	world             *world.Planet
	gravity           mgl32.Vec3
	debugFrameCounter int

	// Dynamic gravity system
	gravitySystem *GravitySystem
}

// NewWorldCollisionSystem crée un nouveau système de collision monde
func NewWorldCollisionSystem(physicsManager *PhysicsManager, world *world.Planet) *WorldCollisionSystem {
	return &WorldCollisionSystem{
		physicsManager: physicsManager,
		world:          world,
		gravity:        mgl32.Vec3{0, -50, 0}, // Fallback gravity (will be overridden by dynamic system)
	}
}

// SetGravitySystem sets the dynamic gravity system
func (wcs *WorldCollisionSystem) SetGravitySystem(gravitySystem *GravitySystem) {
	wcs.gravitySystem = gravitySystem
}

// SetGravity définit la gravité du monde
func (wcs *WorldCollisionSystem) SetGravity(gravity mgl32.Vec3) {
	wcs.gravity = gravity
}

// GetGravity retourne la gravité actuelle
func (wcs *WorldCollisionSystem) GetGravity() mgl32.Vec3 {
	return wcs.gravity
}

// worldToBlockCoords converts world coordinates to chunk and local block coordinates.
// It correctly handles negative coordinates.
func (wcs *WorldCollisionSystem) worldToBlockCoords(pos mgl32.Vec3) (chunkX, chunkY, chunkZ, bx, by, bz int, err error) {
	if wcs.world == nil {
		return 0, 0, 0, 0, 0, 0, fmt.Errorf("world is not initialized")
	}
	// Floor the coordinates to handle negatives correctly
	wx, wy, wz := int(math.Floor(float64(pos.X()))), int(math.Floor(float64(pos.Y()))), int(math.Floor(float64(pos.Z())))

	chunkX = wx / wcs.world.ChunkSize.Width
	chunkY = wy / wcs.world.ChunkSize.Height
	chunkZ = wz / wcs.world.ChunkSize.Depth

	bx = wx % wcs.world.ChunkSize.Width
	by = wy % wcs.world.ChunkSize.Height
	bz = wz % wcs.world.ChunkSize.Depth

	// Adjust for negative modulo result
	if bx < 0 {
		bx += wcs.world.ChunkSize.Width
	}
	if by < 0 {
		by += wcs.world.ChunkSize.Height
	}
	if bz < 0 {
		bz += wcs.world.ChunkSize.Depth
	}
	if pos.X() < 0 && bx != 0 {
		chunkX--
	}
	if pos.Y() < 0 && by != 0 {
		chunkY--
	}
	if pos.Z() < 0 && bz != 0 {
		chunkZ--
	}
	// Validate chunk boundaries
	if chunkX < 0 || chunkY < 0 || chunkZ < 0 ||
		chunkX >= int(wcs.world.Size.X()) ||
		chunkY >= int(wcs.world.Size.Y()) ||
		chunkZ >= int(wcs.world.Size.Z()) {
		return 0, 0, 0, 0, 0, 0, fmt.Errorf("position out of world bounds")
	}

	return chunkX, chunkY, chunkZ, bx, by, bz, nil
}

// RaycastBlock effectue un raycast depuis une position et une direction
// Retourne : chunk, coordonnées locales du bloc (bx, by, bz), position exacte, et hit (bool)
func (wcs *WorldCollisionSystem) RaycastBlock(origin, direction mgl32.Vec3, maxDistance float32) (*world.Chunk, int, int, int, mgl32.Vec3, bool) {
	if wcs.world == nil {
		return nil, 0, 0, 0, mgl32.Vec3{}, false
	}

	dir := direction.Normalize()
	for t := float32(0); t < maxDistance; t += 0.1 {
		pos := origin.Add(dir.Mul(t))

		chunkX, chunkY, chunkZ, bx, by, bz, err := wcs.worldToBlockCoords(pos)
		if err != nil {
			continue // Out of bounds
		}

		chunk, err := wcs.world.GetChunk(chunkX, chunkY, chunkZ)
		if err != nil || chunk == nil {
			continue
		}

		block, err := chunk.GetBlock(bx, by, bz)
		if err != nil {
			continue
		}

		if block.Type != world.BlockTypeAir {
			return chunk, bx, by, bz, pos, true
		}
	}

	return nil, 0, 0, 0, mgl32.Vec3{}, false
}

// CheckCollision vérifie si un objet physique entre en collision avec le monde
// Retourne : aCollision, normale, boîte de collision du bloc touché
func (wcs *WorldCollisionSystem) CheckCollision(obj *PhysicsObject) (bool, mgl32.Vec3, *geom.BoundingBox) {
	if wcs.world == nil || obj == nil {
		return false, mgl32.Vec3{}, nil
	}
	objBox := obj.GetBoundingBox()

	minX, minY, minZ, maxX, maxY, maxZ := objBox.GetBlockBounds()

	for x := minX; x <= maxX; x++ {
		for y := minY; y <= maxY; y++ {
			for z := minZ; z <= maxZ; z++ {
				block, _, err := wcs.world.GetBlockAt(x, y, z)
				if err != nil || block.Type == world.BlockTypeAir {
					continue
				}

				blockBox := geom.NewBoundingBox(
					mgl32.Vec3{float32(x), float32(y), float32(z)},
					mgl32.Vec3{float32(x + 1), float32(y + 1), float32(z + 1)},
				)

				if objBox.Intersects(blockBox) {
					normal := wcs.calculateCollisionNormal(objBox, blockBox)
					// log.Printf("COLLISION DETECTED! Block at (%d,%d,%d), normal: (%.1f,%.1f,%.1f)",
					// 	x, y, z, normal.X(), normal.Y(), normal.Z())
					return true, normal, &blockBox
				}
			}
		}
	}

	return false, mgl32.Vec3{}, nil
}

// calculateCollisionNormal calcule la normale de collision entre deux boîtes
func (wcs *WorldCollisionSystem) calculateCollisionNormal(objBox, blockBox geom.BoundingBox) mgl32.Vec3 {
	dist := objBox.GetCenter().Sub(blockBox.GetCenter())
	pX := (objBox.GetSize().X() / 2) + (blockBox.GetSize().X() / 2) - mgl32.Abs(dist.X())
	pY := (objBox.GetSize().Y() / 2) + (blockBox.GetSize().Y() / 2) - mgl32.Abs(dist.Y())
	pZ := (objBox.GetSize().Z() / 2) + (blockBox.GetSize().Z() / 2) - mgl32.Abs(dist.Z())

	if pX < pY && pX < pZ {
		if dist.X() > 0 {
			return mgl32.Vec3{1, 0, 0}
		}
		return mgl32.Vec3{-1, 0, 0}
	} else if pY < pZ {
		if dist.Y() > 0 {
			return mgl32.Vec3{0, 1, 0}
		}
		return mgl32.Vec3{0, -1, 0}
	} else {
		if dist.Z() > 0 {
			return mgl32.Vec3{0, 0, 1}
		}
		return mgl32.Vec3{0, 0, -1}
	}
}

// Epsilon utilisé pour éviter les problèmes d'arrondi lors du repositionnement
const groundEpsilon = 0.1

// resolveWorldCollisions résout les collisions pour un objet donné et retourne le nombre de collisions.
// Il gère maintenant la détection du sol en fonction de la gravité dynamique.
func (wcs *WorldCollisionSystem) resolveWorldCollisions(obj *PhysicsObject, maxIterations int) int {
	collisionCount := 0
	obj.SetGrounded(false) // Réinitialiser l'état 'grounded' à chaque frame

	// Obtenir la direction de la gravité pour cet objet
	var gravityDir mgl32.Vec3
	if wcs.gravitySystem != nil {
		gravityDir = wcs.gravitySystem.CalculateGravityInfo(obj.GetPosition()).Direction
	} else {
		gravityDir = mgl32.Vec3{0, -1, 0} // Fallback
	}
	upDir := gravityDir.Mul(-1) // La direction "vers le haut" est l'opposé de la gravité

	for i := 0; i < maxIterations; i++ {
		collided, normal, blockBox := wcs.CheckCollision(obj)
		if !collided {
			break // Pas de collision, on arrête
		}
		collisionCount++

		// --- Résolution de la Pénétration ---
		const epsilon = 0.001
		objBox := obj.GetBoundingBox()
		centerDist := objBox.GetCenter().Sub(blockBox.GetCenter())
		pX := (objBox.GetSize().X()/2 + blockBox.GetSize().X()/2) - mgl32.Abs(centerDist.X())
		pY := (objBox.GetSize().Y()/2 + blockBox.GetSize().Y()/2) - mgl32.Abs(centerDist.Y())
		pZ := (objBox.GetSize().Z()/2 + blockBox.GetSize().Z()/2) - mgl32.Abs(centerDist.Z())

		pos := obj.GetPosition()
		if pX < pY && pX < pZ {
			pos[0] += normal.X() * (pX + epsilon)
		} else if pY < pZ {
			pos[1] += normal.Y() * (pY + epsilon)
		} else {
			pos[2] += normal.Z() * (pZ + epsilon)
		}
		obj.SetPosition(pos)

		// --- Détection du Sol (Grounded) ---
		// Un objet est au sol si la normale de collision est opposée à la direction "vers le haut"
		// Le produit scalaire doit être proche de 1 (vecteurs alignés)
		dot := normal.Dot(upDir)
		if dot > 0.7 { // Seuil généreux pour les surfaces non parfaitement plates
			obj.SetGrounded(true)
		}

		// --- Réponse de la Vitesse ---
		vel := obj.GetVelocity()
		// Si l'objet est au sol, on annule toute la vitesse dans la direction de la gravité
		if obj.IsGrounded() {
			// Projeter la vitesse sur le plan du sol
			vel = vel.Sub(gravityDir.Mul(vel.Dot(gravityDir)))
		} else {
			// Sinon, c'est un mur, on ne fait que glisser
			vel = vel.Sub(normal.Mul(vel.Dot(normal) * 1.1)) // Appliquer une légère restitution
		}
		obj.SetVelocity(vel)
	}
	return collisionCount
}

// Update met à jour le système de collision monde
func (wcs *WorldCollisionSystem) Update(deltaTime float32) {
	const maxCollisionIterations = 8

	for _, obj := range wcs.physicsManager.objects {
		if !obj.IsDynamic() {
			continue
		}

		// Track player physics debugging
		isPlayer := obj.GetID() == "player"
		startPos := obj.GetPosition()
		startGrounded := obj.IsGrounded()

		// Reset grounded state at the beginning of the frame
		obj.SetGrounded(false)

		// Apply gravity and orientation - use dynamic gravity if available, otherwise fallback
		if wcs.gravitySystem != nil {
			// Use dynamic gravity system which applies both force and orientation
			wcs.gravitySystem.ApplyGravityToObject(obj)
		} else {
			// Fallback to static gravity (force only)
			gravityForce := wcs.gravity.Mul(obj.GetMass())
			obj.ApplyForce(gravityForce)
		}

		// Update physics (integration step)
		obj.Update(deltaTime)

		// Resolve collisions with the world
		collisions := wcs.resolveWorldCollisions(obj, maxCollisionIterations)

		if isPlayer {
			wcs.debugFrameCounter++
			if wcs.debugFrameCounter%5 == 0 { // Log every 5 frames for player
				log.Printf("Player Physics: Pos(%.1f,%.1f,%.1f)→(%.1f,%.1f,%.1f) Vel(%.1f,%.1f,%.1f) Grounded:%t→%t Collisions:%d",
					startPos.X(), startPos.Y(), startPos.Z(),
					obj.GetPosition().X(), obj.GetPosition().Y(), obj.GetPosition().Z(),
					obj.GetVelocity().X(), obj.GetVelocity().Y(), obj.GetVelocity().Z(),
					startGrounded, obj.IsGrounded(), collisions)
			}
		}
	}
}

// isObjectGrounded vérifie si l'objet est sur le sol
// DEPRECATED: La logique est maintenant dans resolveWorldCollisions
func (wcs *WorldCollisionSystem) isObjectGrounded(obj *PhysicsObject) bool {
	if obj == nil {
		return false
	}
	// On se fie maintenant à la détection faite pendant la résolution des collisions.
	// On pourrait garder une vérification par raycast ici si nécessaire.
	return obj.IsGrounded()
}

// PlaceBlock place un bloc dans le monde et met à jour les chunks adjacents
func (wcs *WorldCollisionSystem) PlaceBlock(position mgl32.Vec3, blockType world.BlockType) error {
	if wcs.world == nil {
		return fmt.Errorf("world is nil")
	}

	chunkX, chunkY, chunkZ, bx, by, bz, err := wcs.worldToBlockCoords(position)
	if err != nil {
		return fmt.Errorf("failed to get chunk for placing block: %w", err)
	}

	chunk, err := wcs.world.GetChunk(chunkX, chunkY, chunkZ)
	if err != nil {
		return fmt.Errorf("failed to get chunk: %w", err)
	}

	if err := chunk.SetBlock(bx, by, bz, world.Block{Type: blockType}); err != nil {
		return fmt.Errorf("failed to set block: %w", err)
	}

	log.Printf("Block placed at chunk (%d,%d,%d) block (%d,%d,%d)", chunkX, chunkY, chunkZ, bx, by, bz)
	return nil
}

// RemoveBlock supprime un bloc à la position spécifiée
func (wcs *WorldCollisionSystem) RemoveBlock(position mgl32.Vec3) error {
	if wcs.world == nil {
		return fmt.Errorf("world is nil")
	}

	chunkX, chunkY, chunkZ, bx, by, bz, err := wcs.worldToBlockCoords(position)
	if err != nil {
		return err
	}

	chunk, err := wcs.world.GetChunk(chunkX, chunkY, chunkZ)
	if err != nil {
		return fmt.Errorf("failed to get chunk: %w", err)
	}

	if err := chunk.SetBlock(bx, by, bz, world.Block{Type: world.BlockTypeAir}); err != nil {
		return fmt.Errorf("failed to set block: %w", err)
	}

	log.Printf("Block removed at chunk (%d,%d,%d) block (%d,%d,%d)", chunkX, chunkY, chunkZ, bx, by, bz)
	return nil
}

// GetBlockAt retrieves a block at the given world coordinates.
func (wcs *WorldCollisionSystem) GetBlockAt(position mgl32.Vec3) (*world.Block, error) {
	if wcs.world == nil {
		return nil, fmt.Errorf("world is nil")
	}
	chunkX, chunkY, chunkZ, bx, by, bz, err := wcs.worldToBlockCoords(position)
	if err != nil {
		return nil, err
	}
	chunk, err := wcs.world.GetChunk(chunkX, chunkY, chunkZ)
	if err != nil {
		return nil, fmt.Errorf("failed to get chunk: %w", err)
	}

	block, err := chunk.GetBlock(bx, by, bz)
	if err != nil {
		return nil, fmt.Errorf("failed to get block: %w", err)
	}

	return &block, nil
}

// min retourne le minimum de deux valeurs float32
func min(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}
