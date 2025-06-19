package camera

import (
	"fmt"

	"github.com/go-gl/mathgl/mgl32"
)

// FaceTransitionManager handles transitions between world faces
type FaceTransitionManager struct {
	// Face normal definitions for determining which face we're looking at
	faceNormals map[WorldFace]mgl32.Vec3
}

// NewFaceTransitionManager creates a new face transition manager
func NewFaceTransitionManager() *FaceTransitionManager {
	return &FaceTransitionManager{
		faceNormals: map[WorldFace]mgl32.Vec3{
			WorldFaceTop:    {0, 1, 0},
			WorldFaceBottom: {0, -1, 0},
			WorldFaceLeft:   {-1, 0, 0},
			WorldFaceRight:  {1, 0, 0},
			WorldFaceFront:  {0, 0, 1},
			WorldFaceBack:   {0, 0, -1},
		},
	}
}

// StartTransition initiates a transition to a new world face
func (ftm *FaceTransitionManager) StartTransition(camera *Camera, targetFace WorldFace) error {
	if camera.state == CameraStateTransitioning {
		return fmt.Errorf("camera is already transitioning")
	}

	if camera.currentFace == targetFace {
		return fmt.Errorf("camera is already on target face")
	}

	// Store current vectors as starting point
	startVectors := &CameraVectors{
		Front: camera.front,
		Up:    camera.up,
		Right: camera.right,
	}

	// Calculate new angles for the target face
	newYaw, newPitch := ftm.calculateTargetAngles(camera.currentFace, targetFace, camera.yaw, camera.pitch)

	// Update camera state
	camera.transitionState = &FaceTransitionState{
		isActive:     true,
		startTime:    0,
		duration:     60, // 1 second at 60fps
		startFace:    camera.currentFace,
		targetFace:   targetFace,
		startVectors: startVectors,
	}

	camera.currentFace = targetFace
	camera.yaw = newYaw
	camera.pitch = newPitch
	camera.state = CameraStateTransitioning

	return nil
}

// calculateTargetAngles determines the appropriate yaw and pitch for the target face
func (ftm *FaceTransitionManager) calculateTargetAngles(fromFace, toFace WorldFace, currentYaw, currentPitch float32) (yaw, pitch float32) {
	// Default pitch for all faces
	pitch = -15.0

	// Calculate yaw based on face transition rules
	switch {
	case fromFace == WorldFaceTop && toFace == WorldFaceFront:
		yaw = 180
	case fromFace == WorldFaceTop && toFace == WorldFaceLeft:
		yaw = 180
	case fromFace == WorldFaceFront && toFace == WorldFaceRight:
		yaw = -currentYaw
	case fromFace == WorldFaceFront && toFace == WorldFaceBottom:
		yaw = 90
	case fromFace == WorldFaceFront && toFace == WorldFaceTop:
		yaw = -90
	case fromFace == WorldFaceBack && toFace == WorldFaceRight:
		yaw = 90
	case fromFace == WorldFaceRight && toFace == WorldFaceBack:
		yaw = 90
	case fromFace == WorldFaceRight && toFace == WorldFaceFront:
		yaw = -currentYaw
	case fromFace == WorldFaceRight && toFace == WorldFaceBottom:
		yaw = 180
	case fromFace == WorldFaceBottom && toFace == WorldFaceBack:
		yaw = 0
	case fromFace == WorldFaceBottom && toFace == WorldFaceFront:
		yaw = 0
	case fromFace == WorldFaceBottom && toFace == WorldFaceLeft:
		yaw = 0
	case fromFace == WorldFaceBack && toFace == WorldFaceBottom:
		yaw = -90
	case fromFace == WorldFaceBack && toFace == WorldFaceTop:
		yaw = 90
	case fromFace == WorldFaceLeft && toFace == WorldFaceBottom:
		yaw = 0
	case fromFace == WorldFaceBottom && toFace == WorldFaceLeft:
		yaw = 0
	default:
		// For other transitions, maintain current yaw
		yaw = currentYaw
	}

	return yaw, pitch
}

// GetLookFace determines which face the camera is looking at based on its front vector
func (ftm *FaceTransitionManager) GetLookFace(camera *Camera) WorldFace {
	front := camera.front.Normalize()

	var bestFace WorldFace
	bestDot := float32(-1)

	for face, normal := range ftm.faceNormals {
		dot := front.Dot(normal)
		if dot > bestDot {
			bestDot = dot
			bestFace = face
		}
	}

	return bestFace
}

// GetFaceByPosition determines which face the camera is on based on its position
func (ftm *FaceTransitionManager) GetFaceByPosition(position mgl32.Vec3, currentFace WorldFace) WorldFace {
	if position.LenSqr() == 0 {
		return currentFace
	}

	direction := position.Normalize()

	var bestFace WorldFace
	bestDot := float32(-1)

	for face, normal := range ftm.faceNormals {
		dot := direction.Dot(normal)
		if dot > bestDot {
			bestDot = dot
			bestFace = face
		}
	}

	// Apply hysteresis to prevent flickering
	currentNormal := ftm.faceNormals[currentFace]
	currentDot := direction.Dot(currentNormal)

	if bestFace != currentFace && bestDot < currentDot {
		bestFace = currentFace
	}

	return bestFace
}

// IsValidTransition checks if a transition between two faces is valid
func (ftm *FaceTransitionManager) IsValidTransition(fromFace, toFace WorldFace) bool {
	// All face transitions are valid in this implementation
	// Could be extended to restrict certain transitions
	return fromFace != toFace
}

// GetTransitionDuration returns the duration for a specific face transition
func (ftm *FaceTransitionManager) GetTransitionDuration(fromFace, toFace WorldFace) int {
	// Could be customized based on transition type
	return 60 // 1 second at 60fps
}
