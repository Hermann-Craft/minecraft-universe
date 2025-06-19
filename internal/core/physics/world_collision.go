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
}

// NewWorldCollisionSystem crée un nouveau système de collision monde
func NewWorldCollisionSystem(physicsManager *PhysicsManager, world *world.Planet) *WorldCollisionSystem {
	return &WorldCollisionSystem{
		physicsManager: physicsManager,
		world:          world,
		gravity:        mgl32.Vec3{0, -50, 0}, // Increased gravity
	}
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

// ResolveCollision résout une collision entre un objet et un bloc
func (wcs *WorldCollisionSystem) ResolveCollision(obj *PhysicsObject, normal mgl32.Vec3, blockBox *geom.BoundingBox) {
	if blockBox == nil {
		return
	}
	const epsilon = 0.001 // Petite marge pour éviter de rester coincé
	objBox := obj.GetBoundingBox()

	pX := (objBox.GetSize().X()/2 + blockBox.GetSize().X()/2) - mgl32.Abs(objBox.GetCenter().X()-blockBox.GetCenter().X())
	pY := (objBox.GetSize().Y()/2 + blockBox.GetSize().Y()/2) - mgl32.Abs(objBox.GetCenter().Y()-blockBox.GetCenter().Y())
	pZ := (objBox.GetSize().Z()/2 + blockBox.GetSize().Z()/2) - mgl32.Abs(objBox.GetCenter().Z()-blockBox.GetCenter().Z())

	pos := obj.GetPosition()
	// Résoudre en priorité la collision avec la plus grande pénétration
	if pX < pY && pX < pZ {
		pos[0] += normal.X() * (pX + epsilon)
	} else if pY < pX && pY < pZ {
		pos[1] += normal.Y() * (pY + epsilon)
	} else {
		pos[2] += normal.Z() * (pZ + epsilon)
	}

	obj.SetPosition(pos)

	vel := obj.GetVelocity()
	// Si la collision est avec le sol, on stoppe la vélocité verticale.
	// Pour les murs, on la laisse intacte pour permettre de glisser.
	if normal.Y() > 0.5 {
		vel[1] = 0
	}

	// Projeter la vélocité pour glisser le long des murs
	vel = vel.Sub(normal.Mul(vel.Dot(normal)))
	obj.SetVelocity(vel)

	if normal.Y() > 0 {
		obj.SetGrounded(true)
	}
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
		obj.SetGrounded(wcs.isObjectGrounded(obj))

		if !obj.IsGrounded() {
			obj.ApplyForce(wcs.gravity.Mul(obj.GetMass()))
		}

		obj.Update(deltaTime)

		collisionCount := 0
		for i := 0; i < maxCollisionIterations; i++ {
			hasCollision, normal, blockBox := wcs.CheckCollision(obj)
			if !hasCollision {
				break
			}
			collisionCount++
			wcs.ResolveCollision(obj, normal, blockBox)
		}

		// 4. Appliquer la friction et réinitialiser l'accélération
		obj.SetGrounded(wcs.isObjectGrounded(obj))
		if obj.IsGrounded() {
			currentVel := obj.GetVelocity()
			currentVel[0] *= 0.9 // Friction horizontale
			currentVel[2] *= 0.9 // Friction horizontale
			obj.SetVelocity(currentVel)
		}
		obj.SetAcceleration(mgl32.Vec3{0, 0, 0})

		// Debug logging for player
		if isPlayer {
			endPos := obj.GetPosition()
			endGrounded := obj.IsGrounded()
			velocity := obj.GetVelocity()

			// Log only when there are significant changes or issues
			if collisionCount > 3 || startGrounded != endGrounded ||
				(wcs.debugFrameCounter%300 == 0 && (velocity.Len() > 0.1 || collisionCount > 0)) {
				log.Printf("Player Physics: Pos(%.1f,%.1f,%.1f)→(%.1f,%.1f,%.1f) Vel(%.1f,%.1f,%.1f) Grounded:%v→%v Collisions:%d",
					startPos.X(), startPos.Y(), startPos.Z(),
					endPos.X(), endPos.Y(), endPos.Z(),
					velocity.X(), velocity.Y(), velocity.Z(),
					startGrounded, endGrounded, collisionCount)
			}
		}
	}
	wcs.debugFrameCounter++
}

// isObjectGrounded vérifie si un objet est au sol
func (wcs *WorldCollisionSystem) isObjectGrounded(obj *PhysicsObject) bool {
	const groundCheckDistance = 0.2
	box := obj.GetBoundingBox()
	center := box.GetCenter()

	checkPoints := []mgl32.Vec3{
		{center.X(), box.Min.Y(), center.Z()},
		{box.Min.X(), box.Min.Y(), box.Min.Z()},
		{box.Max.X(), box.Min.Y(), box.Min.Z()},
		{box.Min.X(), box.Min.Y(), box.Max.Z()},
		{box.Max.X(), box.Min.Y(), box.Max.Z()},
	}

	for _, point := range checkPoints {
		checkPos := point.Add(mgl32.Vec3{0, -groundCheckDistance, 0})
		block, _, err := wcs.world.GetBlockAt(int(checkPos.X()), int(checkPos.Y()), int(checkPos.Z()))

		if err == nil && block != nil && block.Type != world.BlockTypeAir {
			// Found solid ground beneath one of the check points
			return true
		}
	}
	return false
}

// PlaceBlock place un bloc à la position spécifiée
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
