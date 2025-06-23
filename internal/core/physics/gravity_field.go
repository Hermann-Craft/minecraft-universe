package physics

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/hermann-craft/unicube/internal/core/world"
)

// GravityPyramid represents one of the six gravity zones of a cubic planet.
// It is defined by 4 planes whose normals point inwards into the pyramid's volume.
type GravityPyramid struct {
	Face   world.WorldFace
	Planes [4]Plane
}

// IsPointInside checks if a point is within the volume of the pyramid.
// It assumes the plane normals are pointing inwards.
func (p *GravityPyramid) IsPointInside(point mgl32.Vec3) bool {
	for i := 0; i < 4; i++ {
		if p.Planes[i].DistanceToPoint(point) > 0 {
			return false // Point is outside this plane
		}
	}
	return true
}

// PlanetaryGravityField manages the gravity pyramids for a cubic body.
// It performs calculations in the object's local space.
type PlanetaryGravityField struct {
	planet        *world.Planet
	pyramids      [6]GravityPyramid
	halfSize      mgl32.Vec3 // The half-dimensions of the planet, stored for coordinate transformations.
	isInitialized bool
}

// NewPlanetaryGravityField creates a new gravity field manager for a planet.
func NewPlanetaryGravityField(planet *world.Planet) *PlanetaryGravityField {
	return &PlanetaryGravityField{
		planet:        planet,
		isInitialized: false,
	}
}

// Initialize calculates and builds the 6 gravity pyramids for the planet.
// This should be called once the planet's size is known.
func (pgf *PlanetaryGravityField) Initialize() {
	center := mgl32.Vec3{0, 0, 0} // In local space, the center is the origin

	// Correctly calculate the total size of the planet by multiplying each
	// dimension of the planet (in chunks) by the corresponding chunk dimension.
	totalSize := mgl32.Vec3{
		pgf.planet.Size.X() * float32(pgf.planet.ChunkSize.Width),
		pgf.planet.Size.Y() * float32(pgf.planet.ChunkSize.Height),
		pgf.planet.Size.Z() * float32(pgf.planet.ChunkSize.Depth),
	}
	pgf.halfSize = totalSize.Mul(0.5) // Store half size for later use
	hs := pgf.halfSize                // Use a shorter alias for initialization

	// Define the 8 vertices of the planet cube in local space
	vertices := [8]mgl32.Vec3{
		{-hs.X(), -hs.Y(), -hs.Z()}, // 0: left-bottom-back
		{hs.X(), -hs.Y(), -hs.Z()},  // 1: right-bottom-back
		{hs.X(), hs.Y(), -hs.Z()},   // 2: right-top-back
		{-hs.X(), hs.Y(), -hs.Z()},  // 3: left-top-back
		{-hs.X(), -hs.Y(), hs.Z()},  // 4: left-bottom-front
		{hs.X(), -hs.Y(), hs.Z()},   // 5: right-bottom-front
		{hs.X(), hs.Y(), hs.Z()},    // 6: right-top-front
		{-hs.X(), hs.Y(), hs.Z()},   // 7: left-top-front
	}

	// Create pyramids for each face
	// The order of vertices for plane creation is crucial to ensure normals point inwards.
	// The order is (center, corner1, corner2) - reversing the corner order to fix normal direction
	pgf.pyramids = [6]GravityPyramid{
		// Top Face (+Y)
		{Face: world.WorldFaceTop, Planes: [4]Plane{
			NewPlaneFromPoints(center, vertices[6], vertices[7]), // Top-Front edge (reversed)
			NewPlaneFromPoints(center, vertices[2], vertices[6]), // Top-Right edge (reversed)
			NewPlaneFromPoints(center, vertices[3], vertices[2]), // Top-Back edge (reversed)
			NewPlaneFromPoints(center, vertices[7], vertices[3]), // Top-Left edge (reversed)
		}},
		// Bottom Face (-Y)
		{Face: world.WorldFaceBottom, Planes: [4]Plane{
			NewPlaneFromPoints(center, vertices[1], vertices[0]), // Bottom-Back edge (reversed)
			NewPlaneFromPoints(center, vertices[5], vertices[1]), // Bottom-Right edge (reversed)
			NewPlaneFromPoints(center, vertices[4], vertices[5]), // Bottom-Front edge (reversed)
			NewPlaneFromPoints(center, vertices[0], vertices[4]), // Bottom-Left edge (reversed)
		}},
		// Right Face (+X)
		{Face: world.WorldFaceRight, Planes: [4]Plane{
			NewPlaneFromPoints(center, vertices[6], vertices[2]), // Right-Top edge (reversed)
			NewPlaneFromPoints(center, vertices[5], vertices[6]), // Right-Front edge (reversed)
			NewPlaneFromPoints(center, vertices[1], vertices[5]), // Right-Bottom edge (reversed)
			NewPlaneFromPoints(center, vertices[2], vertices[1]), // Right-Back edge (reversed)
		}},
		// Left Face (-X)
		{Face: world.WorldFaceLeft, Planes: [4]Plane{
			NewPlaneFromPoints(center, vertices[7], vertices[4]), // Left-Front edge (reversed)
			NewPlaneFromPoints(center, vertices[3], vertices[7]), // Left-Top edge (reversed)
			NewPlaneFromPoints(center, vertices[0], vertices[3]), // Left-Back edge (reversed)
			NewPlaneFromPoints(center, vertices[4], vertices[0]), // Left-Bottom edge (reversed)
		}},
		// Front Face (+Z)
		{Face: world.WorldFaceFront, Planes: [4]Plane{
			NewPlaneFromPoints(center, vertices[7], vertices[6]), // Front-Top edge (reversed)
			NewPlaneFromPoints(center, vertices[4], vertices[7]), // Front-Left edge (reversed)
			NewPlaneFromPoints(center, vertices[5], vertices[4]), // Front-Bottom edge (reversed)
			NewPlaneFromPoints(center, vertices[6], vertices[5]), // Front-Right edge (reversed)
		}},
		// Back Face (-Z)
		{Face: world.WorldFaceBack, Planes: [4]Plane{
			NewPlaneFromPoints(center, vertices[0], vertices[1]), // Back-Bottom edge (reversed)
			NewPlaneFromPoints(center, vertices[3], vertices[0]), // Back-Left edge (reversed)
			NewPlaneFromPoints(center, vertices[2], vertices[3]), // Back-Top edge (reversed)
			NewPlaneFromPoints(center, vertices[1], vertices[2]), // Back-Right edge (reversed)
		}},
	}
	pgf.isInitialized = true
}

// DetectFaceForPoint determines which gravity pyramid a world-space point resides in.
func (pgf *PlanetaryGravityField) DetectFaceForPoint(worldPoint mgl32.Vec3) world.WorldFace {
	if !pgf.isInitialized {
		pgf.Initialize()
	}

	// Get the inverse of the planet's world transformation matrix
	planetRotation := pgf.planet.Rotation.Mat4()
	planetTranslation := mgl32.Translate3D(pgf.planet.Position.X(), pgf.planet.Position.Y(), pgf.planet.Position.Z())
	worldMatrix := planetTranslation.Mul4(planetRotation)
	inverseWorldMatrix := worldMatrix.Inv()

	// Transform the point from world space to the planet's local space (where the origin is the planet's corner).
	localPoint := inverseWorldMatrix.Mul4x1(worldPoint.Vec4(1)).Vec3()

	// Adjust the local point to be relative to the planet's center, which is the
	// origin (0,0,0) for the pyramid definitions.
	pointRelativeToCenter := localPoint.Sub(pgf.halfSize)

	// Check which pyramid the adjusted point is inside
	for _, pyramid := range pgf.pyramids {
		if pyramid.IsPointInside(pointRelativeToCenter) {
			return pyramid.Face
		}
	}

	return world.WorldFaceNone
}
