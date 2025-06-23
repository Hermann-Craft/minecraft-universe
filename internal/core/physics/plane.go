package physics

import "github.com/go-gl/mathgl/mgl32"

// Plane represents a plane in 3D space defined by the equation Ax + By + Cz + D = 0.
// The normal vector (A, B, C) is expected to be normalized.
type Plane struct {
	Normal mgl32.Vec3
	D      float32
}

// NewPlaneFromPoints creates a new plane from three points in space.
// The points must be in a counter-clockwise order to produce a normal pointing outwards.
func NewPlaneFromPoints(p1, p2, p3 mgl32.Vec3) Plane {
	normal := p2.Sub(p1).Cross(p3.Sub(p1)).Normalize()
	d := -normal.Dot(p1)
	return Plane{Normal: normal, D: d}
}

// DistanceToPoint calculates the signed distance from the plane to a point.
// A positive distance means the point is on the side of the normal.
// A negative distance means the point is on the opposite side.
// A zero distance means the point is on the plane.
func (p *Plane) DistanceToPoint(point mgl32.Vec3) float32 {
	// Equation: Ax + By + Cz + D
	return p.Normal.Dot(point) + p.D
}
