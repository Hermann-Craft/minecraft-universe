package world

// WorldFace représente une face du monde cubique
type WorldFace int

const (
	WorldFaceTop WorldFace = iota
	WorldFaceBottom
	WorldFaceLeft
	WorldFaceRight
	WorldFaceFront
	WorldFaceBack
)

// String retourne le nom de la face
func (wf WorldFace) String() string {
	switch wf {
	case WorldFaceTop:
		return "top"
	case WorldFaceBottom:
		return "bottom"
	case WorldFaceLeft:
		return "left"
	case WorldFaceRight:
		return "right"
	case WorldFaceFront:
		return "front"
	case WorldFaceBack:
		return "back"
	default:
		return "unknown"
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
