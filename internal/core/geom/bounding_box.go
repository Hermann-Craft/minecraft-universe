package geom

import "github.com/go-gl/mathgl/mgl32"

// BoundingBox represents an axis-aligned bounding box
type BoundingBox struct {
	Min mgl32.Vec3
	Max mgl32.Vec3
}

// NewBoundingBox creates a new bounding box
func NewBoundingBox(min, max mgl32.Vec3) BoundingBox {
	return BoundingBox{Min: min, Max: max}
}

// Contains checks if a point is inside the bounding box
func (bb BoundingBox) Contains(point mgl32.Vec3) bool {
	return point.X() >= bb.Min.X() && point.X() <= bb.Max.X() &&
		point.Y() >= bb.Min.Y() && point.Y() <= bb.Max.Y() &&
		point.Z() >= bb.Min.Z() && point.Z() <= bb.Max.Z()
}

// Intersects checks if two bounding boxes intersect
func (bb BoundingBox) Intersects(other BoundingBox) bool {
	return bb.Min.X() <= other.Max.X() && bb.Max.X() >= other.Min.X() &&
		bb.Min.Y() <= other.Max.Y() && bb.Max.Y() >= other.Min.Y() &&
		bb.Min.Z() <= other.Max.Z() && bb.Max.Z() >= other.Min.Z()
}

// GetCenter returns the center of the bounding box
func (bb BoundingBox) GetCenter() mgl32.Vec3 {
	return bb.Min.Add(bb.Max).Mul(0.5)
}

// GetSize returns the size of the bounding box
func (bb BoundingBox) GetSize() mgl32.Vec3 {
	return bb.Max.Sub(bb.Min)
}

// GetBlockBounds est une aide pour obtenir les limites en coordonnées de bloc d'une BoundingBox
func (bb *BoundingBox) GetBlockBounds() (minX, minY, minZ, maxX, maxY, maxZ int) {
	minX = int(bb.Min.X())
	minY = int(bb.Min.Y())
	minZ = int(bb.Min.Z())
	maxX = int(bb.Max.X())
	maxY = int(bb.Max.Y())
	maxZ = int(bb.Max.Z())
	return
}

// Expand expands the bounding box by the given amount
func (bb BoundingBox) Expand(amount float32) BoundingBox {
	return BoundingBox{
		Min: bb.Min.Sub(mgl32.Vec3{amount, amount, amount}),
		Max: bb.Max.Add(mgl32.Vec3{amount, amount, amount}),
	}
}
