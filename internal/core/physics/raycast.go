package physics

import (
	"fmt"
	"math"

	"github.com/go-gl/mathgl/mgl32"
)

// RaycastResult contains information about a raycast hit
type RaycastResult struct {
	Hit      bool
	Distance float32
	Point    mgl32.Vec3
	Normal   mgl32.Vec3
	Object   *PhysicsObject
	Collider Collider
}

// RaycastHit represents a single raycast hit
type RaycastHit struct {
	Distance float32
	Point    mgl32.Vec3
	Normal   mgl32.Vec3
	Object   *PhysicsObject
}

// RaycastSystem handles raycasting operations
type RaycastSystem struct {
	// Raycast settings
	maxDistance float32
	stepSize    float32
}

// NewRaycastSystem creates a new raycast system
func NewRaycastSystem() *RaycastSystem {
	return &RaycastSystem{
		maxDistance: 1000.0,
		stepSize:    0.1,
	}
}

// SetMaxDistance sets the maximum raycast distance
func (rs *RaycastSystem) SetMaxDistance(distance float32) error {
	if distance <= 0 {
		return fmt.Errorf("max distance must be positive")
	}
	rs.maxDistance = distance
	return nil
}

// GetMaxDistance returns the maximum raycast distance
func (rs *RaycastSystem) GetMaxDistance() float32 {
	return rs.maxDistance
}

// SetStepSize sets the raycast step size
func (rs *RaycastSystem) SetStepSize(stepSize float32) error {
	if stepSize <= 0 {
		return fmt.Errorf("step size must be positive")
	}
	rs.stepSize = stepSize
	return nil
}

// GetStepSize returns the raycast step size
func (rs *RaycastSystem) GetStepSize() float32 {
	return rs.stepSize
}

// Raycast performs a raycast against physics objects
func (rs *RaycastSystem) Raycast(origin, direction mgl32.Vec3, objects []*PhysicsObject) RaycastResult {
	if len(objects) == 0 {
		return RaycastResult{Hit: false}
	}

	// Normalize direction
	dir := direction.Normalize()

	var closestHit *RaycastHit

	// Check each object
	for _, obj := range objects {
		if !obj.IsCollisionEnabled() {
			continue
		}

		collider := obj.GetCollider()
		if collider == nil {
			continue
		}

		position := obj.GetPosition()
		scale := mgl32.Vec3{1, 1, 1} // TODO: Get actual scale

		distance, hit := collider.Raycast(origin, dir, position, scale)
		if hit && distance <= rs.maxDistance {
			if closestHit == nil || distance < closestHit.Distance {
				closestHit = &RaycastHit{
					Distance: distance,
					Point:    origin.Add(dir.Mul(distance)),
					Normal:   rs.calculateNormal(origin, dir, distance, collider, position, scale),
					Object:   obj,
				}
			}
		}
	}

	if closestHit != nil {
		return RaycastResult{
			Hit:      true,
			Distance: closestHit.Distance,
			Point:    closestHit.Point,
			Normal:   closestHit.Normal,
			Object:   closestHit.Object,
			Collider: closestHit.Object.GetCollider(),
		}
	}

	return RaycastResult{Hit: false}
}

// calculateNormal calculates the normal at the raycast hit point
func (rs *RaycastSystem) calculateNormal(origin, direction mgl32.Vec3, distance float32, collider Collider, position, scale mgl32.Vec3) mgl32.Vec3 {
	// For box colliders, calculate normal based on which face was hit
	if boxCollider, ok := collider.(*BoxCollider); ok {
		return rs.calculateBoxNormal(origin, direction, distance, boxCollider, position, scale)
	}

	// For sphere colliders, normal points from center to hit point
	if _, ok := collider.(*SphereCollider); ok {
		hitPoint := origin.Add(direction.Mul(distance))
		normal := hitPoint.Sub(position).Normalize()
		return normal
	}

	// Default: return the direction (not ideal but safe)
	return direction
}

// calculateBoxNormal calculates the normal for a box collider hit
func (rs *RaycastSystem) calculateBoxNormal(origin, direction mgl32.Vec3, distance float32, collider *BoxCollider, position, scale mgl32.Vec3) mgl32.Vec3 {
	hitPoint := origin.Add(direction.Mul(distance))
	localPoint := hitPoint.Sub(position)
	scaledSize := collider.Size.Mul(scale.X()) // Use X scale for all dimensions

	// Find which face was hit by checking which component is closest to the half-extent
	normal := mgl32.Vec3{0, 0, 0}

	// Check X faces
	if float32(math.Abs(float64(localPoint.X()))) >= scaledSize.X()-0.01 {
		if localPoint.X() > 0 {
			normal = mgl32.Vec3{1, 0, 0}
		} else {
			normal = mgl32.Vec3{-1, 0, 0}
		}
	} else if float32(math.Abs(float64(localPoint.Y()))) >= scaledSize.Y()-0.01 {
		// Check Y faces
		if localPoint.Y() > 0 {
			normal = mgl32.Vec3{0, 1, 0}
		} else {
			normal = mgl32.Vec3{0, -1, 0}
		}
	} else if float32(math.Abs(float64(localPoint.Z()))) >= scaledSize.Z()-0.01 {
		// Check Z faces
		if localPoint.Z() > 0 {
			normal = mgl32.Vec3{0, 0, 1}
		} else {
			normal = mgl32.Vec3{0, 0, -1}
		}
	}

	return normal
}

// RaycastAll performs a raycast and returns all hits
func (rs *RaycastSystem) RaycastAll(origin, direction mgl32.Vec3, objects []*PhysicsObject) []RaycastHit {
	if len(objects) == 0 {
		return nil
	}

	// Normalize direction
	dir := direction.Normalize()

	var hits []RaycastHit

	// Check each object
	for _, obj := range objects {
		if !obj.IsCollisionEnabled() {
			continue
		}

		collider := obj.GetCollider()
		if collider == nil {
			continue
		}

		position := obj.GetPosition()
		scale := mgl32.Vec3{1, 1, 1} // TODO: Get actual scale

		distance, hit := collider.Raycast(origin, dir, position, scale)
		if hit && distance <= rs.maxDistance {
			hits = append(hits, RaycastHit{
				Distance: distance,
				Point:    origin.Add(dir.Mul(distance)),
				Normal:   rs.calculateNormal(origin, dir, distance, collider, position, scale),
				Object:   obj,
			})
		}
	}

	// Sort hits by distance
	for i := 0; i < len(hits)-1; i++ {
		for j := i + 1; j < len(hits); j++ {
			if hits[i].Distance > hits[j].Distance {
				hits[i], hits[j] = hits[j], hits[i]
			}
		}
	}

	return hits
}

// SphereCast performs a sphere cast (raycast with radius)
func (rs *RaycastSystem) SphereCast(origin, direction mgl32.Vec3, radius float32, objects []*PhysicsObject) RaycastResult {
	if len(objects) == 0 {
		return RaycastResult{Hit: false}
	}

	// Normalize direction
	dir := direction.Normalize()

	var closestHit *RaycastHit

	// Check each object
	for _, obj := range objects {
		if !obj.IsCollisionEnabled() {
			continue
		}

		collider := obj.GetCollider()
		if collider == nil {
			continue
		}

		position := obj.GetPosition()
		scale := mgl32.Vec3{1, 1, 1} // TODO: Get actual scale

		// Expand the collider by the sphere radius
		expandedCollider := rs.expandCollider(collider, radius)

		distance, hit := expandedCollider.Raycast(origin, dir, position, scale)
		if hit && distance <= rs.maxDistance {
			if closestHit == nil || distance < closestHit.Distance {
				closestHit = &RaycastHit{
					Distance: distance,
					Point:    origin.Add(dir.Mul(distance)),
					Normal:   rs.calculateNormal(origin, dir, distance, collider, position, scale),
					Object:   obj,
				}
			}
		}
	}

	if closestHit != nil {
		return RaycastResult{
			Hit:      true,
			Distance: closestHit.Distance,
			Point:    closestHit.Point,
			Normal:   closestHit.Normal,
			Object:   closestHit.Object,
			Collider: closestHit.Object.GetCollider(),
		}
	}

	return RaycastResult{Hit: false}
}

// expandCollider creates an expanded version of a collider for sphere casting
func (rs *RaycastSystem) expandCollider(collider Collider, radius float32) Collider {
	switch c := collider.(type) {
	case *BoxCollider:
		// Expand box by radius
		expandedSize := c.Size.Add(mgl32.Vec3{radius, radius, radius})
		return NewBoxCollider(expandedSize.Mul(2)) // Convert back to full size
	case *SphereCollider:
		// Expand sphere by radius
		return NewSphereCollider(c.Radius + radius)
	default:
		// For unknown collider types, return the original
		return collider
	}
}

// LineOfSight checks if there's a clear line of sight between two points
func (rs *RaycastSystem) LineOfSight(start, end mgl32.Vec3, objects []*PhysicsObject) bool {
	direction := end.Sub(start)
	distance := direction.Len()

	if distance == 0 {
		return true
	}

	dir := direction.Normalize()
	result := rs.Raycast(start, dir, objects)

	return !result.Hit || result.Distance >= distance
}

// GetClosestObject finds the closest physics object to a point
func (rs *RaycastSystem) GetClosestObject(point mgl32.Vec3, objects []*PhysicsObject) (*PhysicsObject, float32) {
	if len(objects) == 0 {
		return nil, 0
	}

	var closestObject *PhysicsObject
	var closestDistance float32 = math.MaxFloat32

	for _, obj := range objects {
		if !obj.IsCollisionEnabled() {
			continue
		}

		position := obj.GetPosition()
		distance := point.Sub(position).Len()

		if distance < closestDistance {
			closestDistance = distance
			closestObject = obj
		}
	}

	if closestObject == nil {
		return nil, 0
	}

	return closestObject, closestDistance
}
