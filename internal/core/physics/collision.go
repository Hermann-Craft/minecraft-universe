package physics

import (
	"fmt"
	"math"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/hermann-craft/unicube/internal/core/geom"
)

// Collider interface for different types of colliders
type Collider interface {
	// GetBoundingBox returns the bounding box for the collider at the given position, rotation, and scale
	GetBoundingBox(position mgl32.Vec3, rotation mgl32.Quat, scale mgl32.Vec3) geom.BoundingBox

	// Intersects checks if this collider intersects with another collider
	Intersects(other Collider, position1 mgl32.Vec3, rotation1 mgl32.Quat, scale1 mgl32.Vec3, position2 mgl32.Vec3, rotation2 mgl32.Quat, scale2 mgl32.Vec3) bool

	// Raycast performs a raycast against this collider
	Raycast(origin, direction, position mgl32.Vec3, rotation mgl32.Quat, scale mgl32.Vec3) (float32, bool)
}

// BoxCollider represents a box-shaped collider
type BoxCollider struct {
	Size mgl32.Vec3 // Half-extents
}

// NewBoxCollider creates a new box collider
func NewBoxCollider(size mgl32.Vec3) *BoxCollider {
	return &BoxCollider{
		Size: size.Mul(0.5), // Store as half-extents
	}
}

// GetBoundingBox returns the bounding box for the box collider
func (bc *BoxCollider) GetBoundingBox(position mgl32.Vec3, rotation mgl32.Quat, scale mgl32.Vec3) geom.BoundingBox {
	// The 8 corners of the OBB in local space
	scaledHalfExtents := mgl32.Vec3{bc.Size[0] * scale[0], bc.Size[1] * scale[1], bc.Size[2] * scale[2]}
	corners := [8]mgl32.Vec3{
		{-scaledHalfExtents[0], -scaledHalfExtents[1], -scaledHalfExtents[2]},
		{scaledHalfExtents[0], -scaledHalfExtents[1], -scaledHalfExtents[2]},
		{scaledHalfExtents[0], scaledHalfExtents[1], -scaledHalfExtents[2]},
		{-scaledHalfExtents[0], scaledHalfExtents[1], -scaledHalfExtents[2]},
		{-scaledHalfExtents[0], -scaledHalfExtents[1], scaledHalfExtents[2]},
		{scaledHalfExtents[0], -scaledHalfExtents[1], scaledHalfExtents[2]},
		{scaledHalfExtents[0], scaledHalfExtents[1], scaledHalfExtents[2]},
		{-scaledHalfExtents[0], scaledHalfExtents[1], scaledHalfExtents[2]},
	}

	// Rotate corners and translate them to world space to find the new min/max for the AABB
	// Initialize with the first corner
	worldCorner := position.Add(rotation.Rotate(corners[0]))
	min, max := worldCorner, worldCorner

	for i := 1; i < 8; i++ {
		worldCorner = position.Add(rotation.Rotate(corners[i]))
		min[0] = float32(math.Min(float64(min[0]), float64(worldCorner[0])))
		min[1] = float32(math.Min(float64(min[1]), float64(worldCorner[1])))
		min[2] = float32(math.Min(float64(min[2]), float64(worldCorner[2])))
		max[0] = float32(math.Max(float64(max[0]), float64(worldCorner[0])))
		max[1] = float32(math.Max(float64(max[1]), float64(worldCorner[1])))
		max[2] = float32(math.Max(float64(max[2]), float64(worldCorner[2])))
	}

	return geom.NewBoundingBox(min, max)
}

// Intersects checks if two box colliders intersect
func (bc *BoxCollider) Intersects(other Collider, position1 mgl32.Vec3, rotation1 mgl32.Quat, scale1 mgl32.Vec3, position2 mgl32.Vec3, rotation2 mgl32.Quat, scale2 mgl32.Vec3) bool {
	// AABB intersection is a broad-phase check.
	// For accurate OBB-OBB intersection, we would need SAT (Separating Axis Theorem).
	box1 := bc.GetBoundingBox(position1, rotation1, scale1)
	otherBox := other.GetBoundingBox(position2, rotation2, scale2)
	return box1.Intersects(otherBox)
}

// Raycast performs a raycast against the box collider
func (bc *BoxCollider) Raycast(origin, direction, position mgl32.Vec3, rotation mgl32.Quat, scale mgl32.Vec3) (float32, bool) {
	// Transform ray to local space of the box
	invRotation := rotation.Inverse()
	localOrigin := invRotation.Rotate(origin.Sub(position))
	localDirection := invRotation.Rotate(direction)

	effectiveHalfExtents := mgl32.Vec3{bc.Size[0] * scale[0], bc.Size[1] * scale[1], bc.Size[2] * scale[2]}

	// Ray-box intersection using slab method in local space
	tMin := (effectiveHalfExtents.X()*-1 - localOrigin.X()) / localDirection.X()
	tMax := (effectiveHalfExtents.X() - localOrigin.X()) / localDirection.X()

	if tMin > tMax {
		tMin, tMax = tMax, tMin
	}

	tyMin := (effectiveHalfExtents.Y()*-1 - localOrigin.Y()) / localDirection.Y()
	tyMax := (effectiveHalfExtents.Y() - localOrigin.Y()) / localDirection.Y()

	if tyMin > tyMax {
		tyMin, tyMax = tyMax, tyMin
	}

	if tMin > tyMax || tyMin > tMax {
		return 0, false
	}

	if tyMin > tMin {
		tMin = tyMin
	}

	if tyMax < tMax {
		tMax = tyMax
	}

	tzMin := (effectiveHalfExtents.Z()*-1 - localOrigin.Z()) / localDirection.Z()
	tzMax := (effectiveHalfExtents.Z() - localOrigin.Z()) / localDirection.Z()

	if tzMin > tzMax {
		tzMin, tzMax = tzMax, tzMin
	}

	if tMin > tzMax || tzMin > tMax {
		return 0, false
	}

	if tzMin > tMin {
		tMin = tzMin
	}

	if tzMax < tMax {
		tMax = tzMax
	}

	if tMin < 0 {
		return 0, false
	}

	return tMin, true
}

// SphereCollider represents a sphere-shaped collider
type SphereCollider struct {
	Radius float32
}

// NewSphereCollider creates a new sphere collider
func NewSphereCollider(radius float32) *SphereCollider {
	return &SphereCollider{
		Radius: radius,
	}
}

// GetBoundingBox returns the bounding box for the sphere collider
func (sc *SphereCollider) GetBoundingBox(position mgl32.Vec3, rotation mgl32.Quat, scale mgl32.Vec3) geom.BoundingBox {
	// Rotation does not affect the AABB of a sphere
	scaledRadius := sc.Radius * scale.X() // Use X scale as radius scale
	return geom.NewBoundingBox(
		position.Sub(mgl32.Vec3{scaledRadius, scaledRadius, scaledRadius}),
		position.Add(mgl32.Vec3{scaledRadius, scaledRadius, scaledRadius}),
	)
}

// Intersects checks if two sphere colliders intersect
func (sc *SphereCollider) Intersects(other Collider, position1 mgl32.Vec3, rotation1 mgl32.Quat, scale1 mgl32.Vec3, position2 mgl32.Vec3, rotation2 mgl32.Quat, scale2 mgl32.Vec3) bool {
	if otherSphere, ok := other.(*SphereCollider); ok {
		scaledRadius1 := sc.Radius * scale1.X()
		scaledRadius2 := otherSphere.Radius * scale2.X()
		distance := position1.Sub(position2).Len()
		return distance <= scaledRadius1+scaledRadius2
	}

	// For other collider types, use bounding box intersection
	box1 := sc.GetBoundingBox(position1, rotation1, scale1)
	otherBox := other.GetBoundingBox(position2, rotation2, scale2)
	return box1.Intersects(otherBox)
}

// Raycast performs a raycast against the sphere collider
func (sc *SphereCollider) Raycast(origin, direction, position mgl32.Vec3, rotation mgl32.Quat, scale mgl32.Vec3) (float32, bool) {
	// Rotation is ignored for spheres
	// Transform ray to local space
	localOrigin := origin.Sub(position)
	scaledRadius := sc.Radius * scale.X()

	// Ray-sphere intersection
	a := direction.Dot(direction)
	b := 2.0 * direction.Dot(localOrigin)
	c := localOrigin.Dot(localOrigin) - scaledRadius*scaledRadius

	discriminant := b*b - 4*a*c
	if discriminant < 0 {
		return 0, false
	}

	t1 := (-b - float32(math.Sqrt(float64(discriminant)))) / (2 * a)
	t2 := (-b + float32(math.Sqrt(float64(discriminant)))) / (2 * a)

	if t1 < 0 && t2 < 0 {
		return 0, false
	}

	if t1 < 0 {
		return t2, true
	}

	return t1, true
}

// CollisionInfo contains information about a collision
type CollisionInfo struct {
	Object1     *PhysicsObject
	Object2     *PhysicsObject
	Normal      mgl32.Vec3
	Penetration float32
	Point       mgl32.Vec3
}

// CollisionDetector handles collision detection between physics objects
type CollisionDetector struct {
	// Spatial partitioning for optimization
	gridSize float32
	grid     map[string][]*PhysicsObject
}

// NewCollisionDetector creates a new collision detector
func NewCollisionDetector(gridSize float32) *CollisionDetector {
	return &CollisionDetector{
		gridSize: gridSize,
		grid:     make(map[string][]*PhysicsObject),
	}
}

// GetGridKey returns the grid key for a position
func (cd *CollisionDetector) GetGridKey(position mgl32.Vec3) string {
	x := int(position.X() / cd.gridSize)
	y := int(position.Y() / cd.gridSize)
	z := int(position.Z() / cd.gridSize)
	return fmt.Sprintf("%d,%d,%d", x, y, z)
}

// AddObject adds a physics object to the collision detector
func (cd *CollisionDetector) AddObject(obj *PhysicsObject) {
	if !obj.IsCollisionEnabled() {
		return
	}

	key := cd.GetGridKey(obj.GetPosition())
	cd.grid[key] = append(cd.grid[key], obj)
}

// RemoveObject removes a physics object from the collision detector
func (cd *CollisionDetector) RemoveObject(obj *PhysicsObject) {
	key := cd.GetGridKey(obj.GetPosition())
	if objects, exists := cd.grid[key]; exists {
		for i, o := range objects {
			if o == obj {
				cd.grid[key] = append(objects[:i], objects[i+1:]...)
				break
			}
		}
	}
}

// DetectCollisions detects all collisions between physics objects
func (cd *CollisionDetector) DetectCollisions() []CollisionInfo {
	var collisions []CollisionInfo

	// Check each grid cell
	for _, objects := range cd.grid {
		// Check all pairs in this cell
		for i := 0; i < len(objects); i++ {
			for j := i + 1; j < len(objects); j++ {
				obj1 := objects[i]
				obj2 := objects[j]

				if !obj1.IsCollisionEnabled() || !obj2.IsCollisionEnabled() {
					continue
				}

				if obj1.IsStatic() && obj2.IsStatic() {
					continue
				}

				if collision := cd.CheckCollision(obj1, obj2); collision != nil {
					collisions = append(collisions, *collision)
				}
			}
		}
	}

	return collisions
}

// CheckCollision checks for collision between two physics objects
func (cd *CollisionDetector) CheckCollision(obj1, obj2 *PhysicsObject) *CollisionInfo {
	if obj1.GetCollider() == nil || obj2.GetCollider() == nil {
		return nil
	}

	if obj1.GetCollider().Intersects(
		obj2.GetCollider(),
		obj1.GetPosition(), obj1.GetRotation(), obj1.GetScale(),
		obj2.GetPosition(), obj2.GetRotation(), obj2.GetScale(),
	) {
		// Basic collision info for now, no resolution yet
		return &CollisionInfo{
			Object1:     obj1,
			Object2:     obj2,
			Normal:      obj2.GetPosition().Sub(obj1.GetPosition()).Normalize(),
			Penetration: 0, // Placeholder
		}
	}

	return nil
}
