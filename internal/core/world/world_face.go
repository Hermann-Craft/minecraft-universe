package world

// WorldFace représente une face du monde cubique
type WorldFace int

const (
	WorldFaceNone WorldFace = iota // Represents no specific face
	WorldFaceTop
	WorldFaceBottom
	WorldFaceLeft
	WorldFaceRight
	WorldFaceFront
	WorldFaceBack
)

// String returns the string representation of a WorldFace.
func (f WorldFace) String() string {
	switch f {
	case WorldFaceNone:
		return "None"
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

// GetDirection retourne la direction de la face
func (wf WorldFace) GetDirection() (dx, dy, dz int) {
	switch wf {
	case WorldFaceTop:
		return 0, 1, 0
	case WorldFaceBottom:
		return 0, -1, 0
	case WorldFaceLeft:
		return -1, 0, 0
	case WorldFaceRight:
		return 1, 0, 0
	case WorldFaceFront:
		return 0, 0, 1
	case WorldFaceBack:
		return 0, 0, -1
	default:
		return 0, 0, 0
	}
}

// GetOpposite retourne la face opposée
func (wf WorldFace) GetOpposite() WorldFace {
	switch wf {
	case WorldFaceTop:
		return WorldFaceBottom
	case WorldFaceBottom:
		return WorldFaceTop
	case WorldFaceLeft:
		return WorldFaceRight
	case WorldFaceRight:
		return WorldFaceLeft
	case WorldFaceFront:
		return WorldFaceBack
	case WorldFaceBack:
		return WorldFaceFront
	default:
		return wf
	}
}
