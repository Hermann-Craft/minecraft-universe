package physics

import (
	"fmt"
	"math"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/hermann-craft/unicube/internal/core/geom"
)

// Collider interface for different types of colliders
type Collider interface {
	// GetBoundingBox returns the bounding box for the collider at the given position and scale
	GetBoundingBox(position, scale mgl32.Vec3) geom.BoundingBox

	// Intersects checks if this collider intersects with another collider
	Intersects(other Collider, position1, scale1, position2, scale2 mgl32.Vec3) bool

	// Raycast performs a raycast against this collider
	Raycast(origin, direction mgl32.Vec3, position, scale mgl32.Vec3) (float32, bool)
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
func (bc *BoxCollider) GetBoundingBox(position, scale mgl32.Vec3) geom.BoundingBox {
	scaledSize := bc.Size.Mul(scale.X()) // Use X scale for all dimensions
	return geom.NewBoundingBox(
		position.Sub(scaledSize),
		position.Add(scaledSize),
	)
}

// Intersects checks if two box colliders intersect
func (bc *BoxCollider) Intersects(other Collider, position1, scale1, position2, scale2 mgl32.Vec3) bool {
	box1 := bc.GetBoundingBox(position1, scale1)

	if otherBox, ok := other.(*BoxCollider); ok {
		box2 := otherBox.GetBoundingBox(position2, scale2)
		return box1.Intersects(box2)
	}

	// For other collider types, use bounding box intersection
	otherBox := other.GetBoundingBox(position2, scale2)
	return box1.Intersects(otherBox)
}

// Raycast performs a raycast against the box collider
func (bc *BoxCollider) Raycast(origin, direction mgl32.Vec3, position, scale mgl32.Vec3) (float32, bool) {
	// Transform ray to local space
	localOrigin := origin.Sub(position)
	scaledSize := bc.Size.Mul(scale.X()) // Use X scale for all dimensions

	// Ray-box intersection using slab method
	tMin := (localOrigin.X() - scaledSize.X()) / direction.X()
	tMax := (localOrigin.X() + scaledSize.X()) / direction.X()

	if tMin > tMax {
		tMin, tMax = tMax, tMin
	}

	tyMin := (localOrigin.Y() - scaledSize.Y()) / direction.Y()
	tyMax := (localOrigin.Y() + scaledSize.Y()) / direction.Y()

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

	tzMin := (localOrigin.Z() - scaledSize.Z()) / direction.Z()
	tzMax := (localOrigin.Z() + scaledSize.Z()) / direction.Z()

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
func (sc *SphereCollider) GetBoundingBox(position, scale mgl32.Vec3) geom.BoundingBox {
	scaledRadius := sc.Radius * scale.X() // Use X scale as radius scale
	return geom.NewBoundingBox(
		position.Sub(mgl32.Vec3{scaledRadius, scaledRadius, scaledRadius}),
		position.Add(mgl32.Vec3{scaledRadius, scaledRadius, scaledRadius}),
	)
}

// Intersects checks if two sphere colliders intersect
func (sc *SphereCollider) Intersects(other Collider, position1, scale1, position2, scale2 mgl32.Vec3) bool {
	if otherSphere, ok := other.(*SphereCollider); ok {
		scaledRadius1 := sc.Radius * scale1.X()
		scaledRadius2 := otherSphere.Radius * scale2.X()
		distance := position1.Sub(position2).Len()
		return distance <= scaledRadius1+scaledRadius2
	}

	// For other collider types, use bounding box intersection
	box1 := sc.GetBoundingBox(position1, scale1)
	otherBox := other.GetBoundingBox(position2, scale2)
	return box1.Intersects(otherBox)
}

// Raycast performs a raycast against the sphere collider
func (sc *SphereCollider) Raycast(origin, direction mgl32.Vec3, position, scale mgl32.Vec3) (float32, bool) {
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

				collision := cd.CheckCollision(obj1, obj2)
				if collision != nil {
					collisions = append(collisions, *collision)
				}
			}
		}
	}

	return collisions
}

// CheckCollision checks collision between two physics objects
func (cd *CollisionDetector) CheckCollision(obj1, obj2 *PhysicsObject) *CollisionInfo {
	collider1 := obj1.GetCollider()
	collider2 := obj2.GetCollider()

	if collider1 == nil || collider2 == nil {
		return nil
	}

	position1 := obj1.GetPosition()
	scale1 := mgl32.Vec3{1, 1, 1} // TODO: Get actual scale
	position2 := obj2.GetPosition()
	scale2 := mgl32.Vec3{1, 1, 1} // TODO: Get actual scale

	if !collider1.Intersects(collider2, position1, scale1, position2, scale2) {
		return nil
	}

	// Calculate collision normal and penetration
	box1 := collider1.GetBoundingBox(position1, scale1)
	box2 := collider2.GetBoundingBox(position2, scale2)

	// Simple collision response for now
	center1 := box1.GetCenter()
	center2 := box2.GetCenter()
	normal := center2.Sub(center1).Normalize()

	return &CollisionInfo{
		Object1:     obj1,
		Object2:     obj2,
		Normal:      normal,
		Penetration: 0.1, // TODO: Calculate actual penetration
		Point:       center1.Add(center2).Mul(0.5),
	}
}
