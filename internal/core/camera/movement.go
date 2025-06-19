package camera

// MovementDirection represents a movement direction
type MovementDirection int

const (
	MovementForward MovementDirection = iota
	MovementBackward
	MovementLeft
	MovementRight
	MovementUp
	MovementDown
)

// String returns the string representation of the movement direction
func (d MovementDirection) String() string {
	switch d {
	case MovementForward:
		return "Forward"
	case MovementBackward:
		return "Backward"
	case MovementLeft:
		return "Left"
	case MovementRight:
		return "Right"
	case MovementUp:
		return "Up"
	case MovementDown:
		return "Down"
	default:
		return "Unknown"
	}
}

// MovementController handles camera movement
type MovementController struct {
	camera *Camera
}

// NewMovementController creates a new movement controller for a camera
func NewMovementController(camera *Camera) *MovementController {
	return &MovementController{
		camera: camera,
	}
}

// Move moves the camera in the specified direction
func (mc *MovementController) Move(direction MovementDirection, deltaTime float32) {
	if mc.camera.state == CameraStateTransitioning {
		// Don't allow movement during face transitions
		return
	}

	velocity := mc.camera.moveSpeed * deltaTime
	vectors := mc.camera.GetVectors()

	switch direction {
	case MovementForward:
		mc.camera.position = mc.camera.position.Add(vectors.Front.Mul(velocity))
	case MovementBackward:
		mc.camera.position = mc.camera.position.Sub(vectors.Front.Mul(velocity))
	case MovementLeft:
		mc.camera.position = mc.camera.position.Sub(vectors.Right.Mul(velocity * mc.getHorizontalMultiplier()))
	case MovementRight:
		mc.camera.position = mc.camera.position.Add(vectors.Right.Mul(velocity * mc.getHorizontalMultiplier()))
	case MovementUp:
		mc.camera.position = mc.camera.position.Add(vectors.Up.Mul(velocity))
	case MovementDown:
		mc.camera.position = mc.camera.position.Sub(vectors.Up.Mul(velocity))
	}
}

// getHorizontalMultiplier returns a multiplier for horizontal movement based on the current face
func (mc *MovementController) getHorizontalMultiplier() float32 {
	switch mc.camera.currentFace {
	case WorldFaceBack:
		return 1.0
	default:
		return 1.0
	}
}

// ProcessMouseMovement handles mouse movement to rotate the camera.
// It's expected that UpdateVectors will be called once per frame after this.
func (mc *MovementController) ProcessMouseMovement(xoffset, yoffset float32, constrainPitch bool) {
	if mc.camera.state == CameraStateTransitioning {
		return
	}

	yaw, pitch := mc.camera.GetAngles()
	_, mouseSens := mc.camera.GetMovementSettings()

	yaw += xoffset * mouseSens
	pitch -= yoffset * mouseSens

	if constrainPitch {
		if pitch > 89.0 {
			pitch = 89.0
		}
		if pitch < -89.0 {
			pitch = -89.0
		}
	}

	mc.camera.SetAngles(yaw, pitch)
}

// SetMovementSpeed sets the camera movement speed
func (mc *MovementController) SetMovementSpeed(speed float32) {
	mc.camera.moveSpeed = speed
}

// GetMovementSpeed returns the current movement speed
func (mc *MovementController) GetMovementSpeed() float32 {
	return mc.camera.moveSpeed
}

// SetMouseSensitivity sets the mouse sensitivity
func (mc *MovementController) SetMouseSensitivity(sensitivity float32) {
	mc.camera.mouseSens = sensitivity
}

// GetMouseSensitivity returns the current mouse sensitivity
func (mc *MovementController) GetMouseSensitivity() float32 {
	return mc.camera.mouseSens
}
