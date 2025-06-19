package camera

import (
	"github.com/go-gl/mathgl/mgl32"
)

// Frustum represents a view frustum for culling
type Frustum struct {
	planes [6]mgl32.Vec4 // Left, Right, Bottom, Top, Near, Far
}

// NewFrustum creates a new frustum
func NewFrustum() *Frustum {
	return &Frustum{}
}

// Update updates the frustum planes based on view and projection matrices
func (f *Frustum) Update(view, projection mgl32.Mat4) {
	// Combine view and projection matrices
	clip := projection.Mul4(view)

	// Extract frustum planes from the combined matrix
	// Left plane
	f.planes[0] = mgl32.Vec4{
		clip.At(0, 3) + clip.At(0, 0),
		clip.At(1, 3) + clip.At(1, 0),
		clip.At(2, 3) + clip.At(2, 0),
		clip.At(3, 3) + clip.At(3, 0),
	}

	// Right plane
	f.planes[1] = mgl32.Vec4{
		clip.At(0, 3) - clip.At(0, 0),
		clip.At(1, 3) - clip.At(1, 0),
		clip.At(2, 3) - clip.At(2, 0),
		clip.At(3, 3) - clip.At(3, 0),
	}

	// Bottom plane
	f.planes[2] = mgl32.Vec4{
		clip.At(0, 3) + clip.At(0, 1),
		clip.At(1, 3) + clip.At(1, 1),
		clip.At(2, 3) + clip.At(2, 1),
		clip.At(3, 3) + clip.At(3, 1),
	}

	// Top plane
	f.planes[3] = mgl32.Vec4{
		clip.At(0, 3) - clip.At(0, 1),
		clip.At(1, 3) - clip.At(1, 1),
		clip.At(2, 3) - clip.At(2, 1),
		clip.At(3, 3) - clip.At(3, 1),
	}

	// Near plane
	f.planes[4] = mgl32.Vec4{
		clip.At(0, 3) + clip.At(0, 2),
		clip.At(1, 3) + clip.At(1, 2),
		clip.At(2, 3) + clip.At(2, 2),
		clip.At(3, 3) + clip.At(3, 2),
	}

	// Far plane
	f.planes[5] = mgl32.Vec4{
		clip.At(0, 3) - clip.At(0, 2),
		clip.At(1, 3) - clip.At(1, 2),
		clip.At(2, 3) - clip.At(2, 2),
		clip.At(3, 3) - clip.At(3, 2),
	}

	// Normalize all planes
	for i := 0; i < 6; i++ {
		length := float32(1.0) / mgl32.Vec3{f.planes[i].X(), f.planes[i].Y(), f.planes[i].Z()}.Len()
		f.planes[i] = f.planes[i].Mul(length)
	}
}

// IsPointVisible checks if a point is inside the frustum
func (f *Frustum) IsPointVisible(point mgl32.Vec3) bool {
	for i := 0; i < 6; i++ {
		plane := f.planes[i]
		distance := plane.X()*point.X() + plane.Y()*point.Y() + plane.Z()*point.Z() + plane.W()
		if distance < 0 {
			return false
		}
	}
	return true
}

// IsSphereVisible checks if a sphere is inside the frustum
func (f *Frustum) IsSphereVisible(center mgl32.Vec3, radius float32) bool {
	for i := 0; i < 6; i++ {
		plane := f.planes[i]
		distance := plane.X()*center.X() + plane.Y()*center.Y() + plane.Z()*center.Z() + plane.W()
		if distance < -radius {
			return false
		}
	}
	return true
}

// IsBoxVisible checks if an axis-aligned bounding box is inside the frustum
func (f *Frustum) IsBoxVisible(min, max mgl32.Vec3) bool {
	for i := 0; i < 6; i++ {
		plane := f.planes[i]

		// Find the corner of the box that is farthest from the plane
		var x, y, z float32
		if plane.X() >= 0 {
			x = max.X()
		} else {
			x = min.X()
		}
		if plane.Y() >= 0 {
			y = max.Y()
		} else {
			y = min.Y()
		}
		if plane.Z() >= 0 {
			z = max.Z()
		} else {
			z = min.Z()
		}

		// If this corner is outside the plane, the box is outside the frustum
		distance := plane.X()*x + plane.Y()*y + plane.Z()*z + plane.W()
		if distance < 0 {
			return false
		}
	}
	return true
}
