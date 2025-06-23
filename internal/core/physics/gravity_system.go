package physics

import (
	"log"
	"math"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/hermann-craft/unicube/internal/core/world"
)

// Debug frame counter for logging
var debugFrameCounter int

// GravityInfo contains all gravity-related information for an object
type GravityInfo struct {
	Face        world.WorldFace // Which face the object is on
	Direction   mgl32.Vec3      // Gravity direction (normalized)
	Magnitude   float32         // Gravity strength
	Orientation mgl32.Quat      // Object's "up" orientation relative to the face
}

// GravitySystem manages dynamic gravity based on planetary faces
type GravitySystem struct {
	planet          *world.Planet
	gravityField    *PlanetaryGravityField
	gravityStrength float32
}

// NewGravitySystem creates a new gravity system
func NewGravitySystem(planet *world.Planet, gravityField *PlanetaryGravityField) *GravitySystem {
	return &GravitySystem{
		planet:          planet,
		gravityField:    gravityField,
		gravityStrength: 50.0, // Default gravity strength
	}
}

// SetGravityStrength sets the strength of gravity
func (gs *GravitySystem) SetGravityStrength(strength float32) {
	gs.gravityStrength = strength
}

// GetGravityStrength returns the current gravity strength
func (gs *GravitySystem) GetGravityStrength() float32 {
	return gs.gravityStrength
}

// CalculateGravityInfo calculates gravity direction and orientation for a position
func (gs *GravitySystem) CalculateGravityInfo(worldPosition mgl32.Vec3) GravityInfo {
	// Detect which face the object is on
	face := gs.gravityField.DetectFaceForPoint(worldPosition)

	// Calculate gravity direction based on the face
	direction := gs.getGravityDirectionForFace(face, worldPosition)

	// Calculate the "up" orientation for this face
	orientation := gs.getOrientationForFace(face)

	return GravityInfo{
		Face:        face,
		Direction:   direction,
		Magnitude:   gs.gravityStrength,
		Orientation: orientation,
	}
}

// getGravityDirectionForFace returns the gravity direction for a specific face
func (gs *GravitySystem) getGravityDirectionForFace(face world.WorldFace, worldPosition mgl32.Vec3) mgl32.Vec3 {
	// Calculate the center of the planet in world space
	planetCenter := gs.planet.Position.Add(mgl32.Vec3{
		gs.planet.Size.X() * float32(gs.planet.ChunkSize.Width) * 0.5,
		gs.planet.Size.Y() * float32(gs.planet.ChunkSize.Height) * 0.5,
		gs.planet.Size.Z() * float32(gs.planet.ChunkSize.Depth) * 0.5,
	})

	switch face {
	case world.WorldFaceTop:
		// Gravity points towards the top face (down in Y)
		return mgl32.Vec3{0, -1, 0}
	case world.WorldFaceBottom:
		// Gravity points towards the bottom face (up in Y)
		return mgl32.Vec3{0, 1, 0}
	case world.WorldFaceLeft:
		// Gravity points towards the left face (right in X)
		return mgl32.Vec3{1, 0, 0}
	case world.WorldFaceRight:
		// Gravity points towards the right face (left in X)
		return mgl32.Vec3{-1, 0, 0}
	case world.WorldFaceFront:
		// Gravity points towards the front face (back in Z)
		return mgl32.Vec3{0, 0, -1}
	case world.WorldFaceBack:
		// Gravity points towards the back face (forward in Z)
		return mgl32.Vec3{0, 0, 1}
	default:
		// For WorldFaceNone, calculate direction towards planet center
		toCenter := planetCenter.Sub(worldPosition)
		if toCenter.LenSqr() > 0 {
			return toCenter.Normalize()
		}
		// Fallback to standard downward gravity
		return mgl32.Vec3{0, -1, 0}
	}
}

// getOrientationForFace returns the "up" orientation for a specific face
func (gs *GravitySystem) getOrientationForFace(face world.WorldFace) mgl32.Quat {
	switch face {
	case world.WorldFaceTop:
		// Standard orientation - up is +Y
		return mgl32.QuatIdent()
	case world.WorldFaceBottom:
		// Upside down - up is -Y (180° rotation around X)
		return mgl32.QuatRotate(math.Pi, mgl32.Vec3{1, 0, 0})
	case world.WorldFaceLeft:
		// Left face - up is +X (90° rotation around Z)
		return mgl32.QuatRotate(math.Pi/2, mgl32.Vec3{0, 0, 1})
	case world.WorldFaceRight:
		// Right face - up is -X (-90° rotation around Z)
		return mgl32.QuatRotate(-math.Pi/2, mgl32.Vec3{0, 0, 1})
	case world.WorldFaceFront:
		// Front face - up is +Z (-90° rotation around X)
		return mgl32.QuatRotate(-math.Pi/2, mgl32.Vec3{1, 0, 0})
	case world.WorldFaceBack:
		// Back face - up is -Z (90° rotation around X)
		return mgl32.QuatRotate(math.Pi/2, mgl32.Vec3{1, 0, 0})
	default:
		// Default orientation
		return mgl32.QuatIdent()
	}
}

// GetGravityForce calculates the gravity force to apply to an object
func (gs *GravitySystem) GetGravityForce(worldPosition mgl32.Vec3, mass float32) mgl32.Vec3 {
	gravityInfo := gs.CalculateGravityInfo(worldPosition)
	return gravityInfo.Direction.Mul(gravityInfo.Magnitude * mass)
}

// ApplyGravityToObject applies gravity force to a physics object (orientation managed by camera)
func (gs *GravitySystem) ApplyGravityToObject(obj *PhysicsObject) {
	if obj == nil {
		return
	}

	position := obj.GetPosition()
	gravityInfo := gs.CalculateGravityInfo(position)

	// Apply gravity force only - let camera manage orientation
	gravityForce := gravityInfo.Direction.Mul(gravityInfo.Magnitude * obj.GetMass())
	obj.ApplyForce(gravityForce)

	// Debug logging for player object
	if obj.GetID() == "player" {
		// Log every 180 frames (3 seconds at 60fps) to avoid spam
		debugFrameCounter++
		if debugFrameCounter%180 == 0 {
			log.Printf("Gravity Debug: Face=%s, Direction=(%.2f,%.2f,%.2f)",
				gravityInfo.Face, gravityInfo.Direction.X(), gravityInfo.Direction.Y(), gravityInfo.Direction.Z())
		}
	}
}
