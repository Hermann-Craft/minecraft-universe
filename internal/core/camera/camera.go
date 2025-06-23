package camera

import (
	"fmt"
	"math"

	"github.com/go-gl/mathgl/mgl32"
)

// CameraState represents the explicit state of the camera
type CameraState int

const (
	CameraStateIdle CameraState = iota
	CameraStateTransitioning
	CameraStateInitializing
)

// String returns the string representation of the camera state
func (s CameraState) String() string {
	switch s {
	case CameraStateIdle:
		return "Idle"
	case CameraStateTransitioning:
		return "Transitioning"
	case CameraStateInitializing:
		return "Initializing"
	default:
		return "Unknown"
	}
}

// WorldFace represents the face of the cubic world
type WorldFace int

const (
	WorldFaceTop WorldFace = iota
	WorldFaceBottom
	WorldFaceLeft
	WorldFaceRight
	WorldFaceFront
	WorldFaceBack
)

// String returns the string representation of the world face
func (f WorldFace) String() string {
	switch f {
	case WorldFaceTop:
		return "Top"
	case WorldFaceBottom:
		return "Bottom"
	case WorldFaceLeft:
		return "Left"
	case WorldFaceRight:
		return "Right"
	case WorldFaceFront:
		return "Front"
	case WorldFaceBack:
		return "Back"
	default:
		return "Unknown"
	}
}

// Camera represents a 3D camera with cubic world face support
type Camera struct {
	// Core state
	state       CameraState
	position    mgl32.Vec3
	currentFace WorldFace

	// Canonical angles (in degrees)
	yaw   float32
	pitch float32

	// Derived vectors (after face transformation)
	front mgl32.Vec3
	up    mgl32.Vec3
	right mgl32.Vec3

	// Movement settings
	moveSpeed float32
	mouseSens float32

	// Face transition state
	transitionState *FaceTransitionState

	// Frustum for culling
	frustum *Frustum
}

// FaceTransitionState manages face transition animations
type FaceTransitionState struct {
	isActive     bool
	startTime    int
	duration     int
	startFace    WorldFace
	targetFace   WorldFace
	startVectors *CameraVectors
}

// CameraVectors represents the camera's orientation vectors
type CameraVectors struct {
	Front mgl32.Vec3
	Up    mgl32.Vec3
	Right mgl32.Vec3
}

// NewCamera creates a new camera with default settings
func NewCamera(position mgl32.Vec3) *Camera {
	return &Camera{
		state:       CameraStateInitializing,
		position:    position,
		currentFace: WorldFaceTop,
		yaw:         90.0,  // Look along positive X axis
		pitch:       -15.0, // Slight downward angle
		moveSpeed:   5.0,
		mouseSens:   0.1,
		transitionState: &FaceTransitionState{
			isActive: false,
			duration: 60, // 60 frames at 60fps = 1 second
		},
		frustum: NewFrustum(),
	}
}

// Init initializes the camera for a specific world face
func (c *Camera) Init(face WorldFace) error {
	if c.state != CameraStateInitializing {
		return fmt.Errorf("camera can only be initialized once")
	}

	c.currentFace = face
	c.updateVectors()
	c.state = CameraStateIdle

	return nil
}

// GetState returns the current camera state
func (c *Camera) GetState() CameraState {
	return c.state
}

// GetPosition returns the camera position
func (c *Camera) GetPosition() mgl32.Vec3 {
	return c.position
}

// SetPosition sets the camera position
func (c *Camera) SetPosition(pos mgl32.Vec3) {
	c.position = pos
}

// GetCurrentFace returns the current world face
func (c *Camera) GetCurrentFace() WorldFace {
	return c.currentFace
}

// SetFace sets the current world face (can be called after initialization)
func (c *Camera) SetFace(face WorldFace) {
	if c.currentFace != face {
		c.currentFace = face
		c.updateVectors() // Recalculate vectors with new face transformation
	}
}

// GetVectors returns the current camera vectors
func (c *Camera) GetVectors() *CameraVectors {
	return &CameraVectors{
		Front: c.front,
		Up:    c.up,
		Right: c.right,
	}
}

// GetViewMatrix returns the view matrix for rendering
func (c *Camera) GetViewMatrix() mgl32.Mat4 {
	target := c.position.Add(c.front)

	return mgl32.LookAtV(c.position, target, c.up)
}

// GetFrustum returns the camera's frustum for culling
func (c *Camera) GetFrustum() *Frustum {
	return c.frustum
}

// UpdateFrustum updates the frustum with the given projection matrix
func (c *Camera) UpdateFrustum(projection mgl32.Mat4) {
	c.frustum.Update(c.GetViewMatrix(), projection)
}

// IsTransitioning returns true if the camera is currently transitioning between faces
func (c *Camera) IsTransitioning() bool {
	return c.state == CameraStateTransitioning
}

// UpdateVectors calculates the camera vectors based on canonical angles and face transformation
func (c *Camera) UpdateVectors() {
	// 1. Calculate canonical vectors from yaw and pitch
	yawRad := mgl32.DegToRad(c.yaw)
	pitchRad := mgl32.DegToRad(c.pitch)

	frontCanon := mgl32.Vec3{
		float32(math.Cos(float64(yawRad)) * math.Cos(float64(pitchRad))),
		float32(math.Sin(float64(pitchRad))),
		float32(math.Sin(float64(yawRad)) * math.Cos(float64(pitchRad))),
	}.Normalize()

	upCanon := mgl32.Vec3{0, 1, 0}
	rightCanon := frontCanon.Cross(upCanon).Normalize()

	// 2. Apply face transformation
	faceTransform := c.getFaceTransform()

	// Convert to Vec4 for transformation
	front4 := mgl32.Vec4{frontCanon.X(), frontCanon.Y(), frontCanon.Z(), 0}
	up4 := mgl32.Vec4{upCanon.X(), upCanon.Y(), upCanon.Z(), 0}
	right4 := mgl32.Vec4{rightCanon.X(), rightCanon.Y(), rightCanon.Z(), 0}

	// Apply transformation
	transformedFront4 := faceTransform.Mul4x1(front4)
	transformedUp4 := faceTransform.Mul4x1(up4)
	transformedRight4 := faceTransform.Mul4x1(right4)

	// Convert back to Vec3
	newFront := mgl32.Vec3{transformedFront4.X(), transformedFront4.Y(), transformedFront4.Z()}.Normalize()
	newUp := mgl32.Vec3{transformedUp4.X(), transformedUp4.Y(), transformedUp4.Z()}.Normalize()
	newRight := mgl32.Vec3{transformedRight4.X(), transformedRight4.Y(), transformedRight4.Z()}.Normalize()

	// Ensure orthonormality
	newRight = newFront.Cross(newUp).Normalize()
	newUp = newRight.Cross(newFront).Normalize()

	// 3. Handle transition interpolation if needed
	if c.state == CameraStateTransitioning && c.transitionState.isActive {
		alpha := float32(c.transitionState.startTime) / float32(c.transitionState.duration)
		if alpha > 1.0 {
			alpha = 1.0
		}

		startVectors := c.transitionState.startVectors
		c.front = startVectors.Front.Mul(1 - alpha).Add(newFront.Mul(alpha)).Normalize()
		c.up = startVectors.Up.Mul(1 - alpha).Add(newUp.Mul(alpha)).Normalize()
		c.right = startVectors.Right.Mul(1 - alpha).Add(newRight.Mul(alpha)).Normalize()

		c.transitionState.startTime++

		// Check if transition is complete
		if c.transitionState.startTime >= c.transitionState.duration {
			c.front = newFront
			c.up = newUp
			c.right = newRight
			c.state = CameraStateIdle
			c.transitionState.isActive = false
		}
	} else {
		// Direct update
		c.front = newFront
		c.up = newUp
		c.right = newRight
	}
}

// updateVectors is a private alias for UpdateVectors for backward compatibility
func (c *Camera) updateVectors() {
	c.UpdateVectors()
}

// getFaceTransform returns the transformation matrix for the current world face
// These transformations ensure that:
// - Front vector stays horizontal to the face plane
// - Up vector points away from the center of the cube
// - The vectors are consistent with the gravity system
func (c *Camera) getFaceTransform() mgl32.Mat4 {
	switch c.currentFace {
	case WorldFaceTop:
		// No transformation needed - canonical orientation
		return mgl32.Ident4()
	case WorldFaceBottom:
		// Flip upside down (180° around X axis)
		return mgl32.HomogRotate3DX(mgl32.DegToRad(180))
	case WorldFaceLeft:
		// Rotate 90° around Z axis (roll left)
		return mgl32.HomogRotate3DZ(mgl32.DegToRad(90))
	case WorldFaceRight:
		// Rotate -90° around Z axis (roll right)
		return mgl32.HomogRotate3DZ(mgl32.DegToRad(-90))
	case WorldFaceFront:
		return mgl32.HomogRotate3DX(mgl32.DegToRad(90)).Mul4(mgl32.HomogRotate3DY(mgl32.DegToRad(90)))
	case WorldFaceBack:
		return mgl32.HomogRotate3DX(mgl32.DegToRad(-90)).Mul4(mgl32.HomogRotate3DY(mgl32.DegToRad(-90)))
	default:
		return mgl32.Ident4()
	}
}

// SetAngles sets the canonical yaw and pitch angles
func (c *Camera) SetAngles(yaw, pitch float32) {
	c.yaw = yaw
	c.pitch = pitch

	// Clamp pitch to avoid gimbal lock
	if c.pitch > 89.0 {
		c.pitch = 89.0
	}
	if c.pitch < -89.0 {
		c.pitch = -89.0
	}

	c.updateVectors()
}

// GetAngles returns the current canonical angles
func (c *Camera) GetAngles() (yaw, pitch float32) {
	return c.yaw, c.pitch
}

// SetMovementSettings sets the camera movement parameters
func (c *Camera) SetMovementSettings(moveSpeed, mouseSens float32) {
	c.moveSpeed = moveSpeed
	c.mouseSens = mouseSens
}

// GetMovementSettings returns the current movement settings
func (c *Camera) GetMovementSettings() (moveSpeed, mouseSens float32) {
	return c.moveSpeed, c.mouseSens
}

// GetFront returns the camera's front vector
func (c *Camera) GetFront() mgl32.Vec3 {
	return c.front
}
