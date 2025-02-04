package main

import (
	"math"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

//////////////////////////////////////////////////////////
// Structure de la caméra
//////////////////////////////////////////////////////////

type Camera struct {
	Position         mgl32.Vec3 // Position dans le monde
	Yaw              float32    // Angle horizontal (en degrés) dans le repère canonique
	Pitch            float32    // Angle vertical (en degrés) dans le repère canonique
	CurrentWorldFace WorldFace  // La face du monde sur laquelle le joueur se trouve
	MoveSpeed        float32
	MouseSens        float32
	FitWorld         bool
	firstMouse       bool
	lastX, lastY     float64
	// Vecteurs dérivés (après transformation)
	Front mgl32.Vec3
	Up    mgl32.Vec3
	Right mgl32.Vec3

	oldFront mgl32.Vec3
	oldUp    mgl32.Vec3
	oldRight mgl32.Vec3

	IsSwappingFace      bool
	CurrentSwappingTime int
	MaxSwappingTime     int

	// Pour un traitement global, on conserve la référence au monde (pour par exemple accéder à CenterPosition)
	World *World
}

func NewCamera(position mgl32.Vec3) *Camera {
	return &Camera{
		Position:            position,
		MoveSpeed:           5.0,
		MouseSens:           0.1,
		firstMouse:          true,
		IsSwappingFace:      false,
		FitWorld:            true,
		CurrentSwappingTime: 0,
		MaxSwappingTime:     60,
		// Initialisation des angles canoniques par défaut (pour la face Top)
		Yaw:   90,  // Dans le repère canonique, 90° signifie que l'on regarde dans une direction "standard"
		Pitch: -15, // On regarde légèrement vers le bas
	}
}

// Init configure la caméra en fonction du monde et de la face sur laquelle le joueur spawn.
func (cam *Camera) Init(world *World, worldFace WorldFace) {
	cam.World = world
	cam.CurrentWorldFace = worldFace

	// On peut ici éventuellement ajuster les angles canoniques en fonction de la face
	// Si vous le souhaitez, vous pouvez laisser les angles canoniques "Top" et
	// adapter via la transformation.
	// Par exemple, pour WorldFaceBottom, vous pourriez conserver Yaw=90, Pitch=-15
	// et laisser la transformation se charger de "retourner" la vue.
	// Ici, nous partons du principe que le repère canonique correspond à WorldFaceTop.

	cam.updateVectors()
}

//////////////////////////////////////////////////////////
// Transformation selon la face du monde
//////////////////////////////////////////////////////////

// getFaceTransform retourne une matrice de transformation (4x4) qui mappe le repère canonique
// (comme pour WorldFaceTop, c'est-à-dire, Up=(0,1,0)) vers le repère approprié pour la face actuelle.
func (cam *Camera) getFaceTransform() mgl32.Mat4 {
	switch cam.CurrentWorldFace {
	case WorldFaceTop:
		return mgl32.Ident4()
	case WorldFaceBottom:
		// Par exemple, une rotation de 180° autour de l'axe X permet d'inverser verticalement.
		return mgl32.HomogRotate3DX(mgl32.DegToRad(180))
	case WorldFaceLeft:
		// Pour la face Left, on peut appliquer une rotation de +90° autour de l'axe Z.
		return mgl32.HomogRotate3DZ(mgl32.DegToRad(90))
	case WorldFaceRight:
		// Pour la face Right, appliquer -90° autour de l'axe Z.
		return mgl32.HomogRotate3DZ(mgl32.DegToRad(-90))
	case WorldFaceFront:
		// Pour Front, on compose :
		// 1. Une rotation de -90° autour de Y pour tourner la vue vers l'intérieur.
		// 2. Une rotation de +90° autour de X pour orienter l'axe Up de façon à ce que le sol soit sous vos pieds.
		return mgl32.HomogRotate3DX(mgl32.DegToRad(90)).Mul4(mgl32.HomogRotate3DY(mgl32.DegToRad(90)))
	case WorldFaceBack:
		// Pour la face Back, on compose :
		// 1. Une rotation de -90° autour de Y.
		// 2. Une rotation de 90° autour de X.
		return mgl32.HomogRotate3DX(mgl32.DegToRad(-90)).Mul4(mgl32.HomogRotate3DY(mgl32.DegToRad(-90)))
	default:
		return mgl32.Ident4()
	}
}

//////////////////////////////////////////////////////////
// Mise à jour des vecteurs de la caméra
//////////////////////////////////////////////////////////

// updateVectors calcule le vecteur "front" canonique à partir de Yaw et Pitch,
// puis l'adapte via la transformation définie par la face du monde.
func (cam *Camera) updateVectors() {
	// 1. Calcul dans le repère canonique (comme pour WorldFaceTop)
	yawRad := mgl32.DegToRad(cam.Yaw)
	pitchRad := mgl32.DegToRad(cam.Pitch)
	frontCanon := mgl32.Vec3{
		float32(math.Cos(float64(yawRad)) * math.Cos(float64(pitchRad))),
		float32(math.Sin(float64(pitchRad))),
		float32(math.Sin(float64(yawRad)) * math.Cos(float64(pitchRad))),
	}.Normalize()
	upCanon := mgl32.Vec3{0, 1, 0}
	rightCanon := frontCanon.Cross(upCanon).Normalize()

	// 2. Récupérer la matrice de transformation selon la face du monde
	faceTransform := cam.getFaceTransform()

	// Convertir les vecteurs canoniques en Vec4 (w=0 pour les directions)
	front4 := mgl32.Vec4{frontCanon.X(), frontCanon.Y(), frontCanon.Z(), 0}
	up4 := mgl32.Vec4{upCanon.X(), upCanon.Y(), upCanon.Z(), 0}
	right4 := mgl32.Vec4{rightCanon.X(), rightCanon.Y(), rightCanon.Z(), 0}

	// Appliquer la transformation
	transformedFront4 := faceTransform.Mul4x1(front4)
	transformedUp4 := faceTransform.Mul4x1(up4)
	transformedRight4 := faceTransform.Mul4x1(right4)

	// Convertir les résultats en Vec3 (nouvelles directions "cibles")
	newFront := mgl32.Vec3{transformedFront4.X(), transformedFront4.Y(), transformedFront4.Z()}.Normalize()
	newUp := mgl32.Vec3{transformedUp4.X(), transformedUp4.Y(), transformedUp4.Z()}.Normalize()
	newRight := mgl32.Vec3{transformedRight4.X(), transformedRight4.Y(), transformedRight4.Z()}.Normalize()

	// Optionnel : recalculer pour garantir l'orthonormalité
	newRight = newFront.Cross(newUp).Normalize()
	newUp = newRight.Cross(newFront).Normalize()

	// 3. Si une transition est en cours, interpoler entre les anciens vecteurs et les nouveaux cibles
	if cam.IsSwappingFace {
		// Calculer le facteur d'interpolation (entre 0 et 1)
		alpha := float32(cam.CurrentSwappingTime) / float32(cam.MaxSwappingTime)
		// Interpoler linéairement et normaliser
		cam.Front = cam.oldFront.Mul(1 - alpha).Add(newFront.Mul(alpha)).Normalize()
		cam.Up = cam.oldUp.Mul(1 - alpha).Add(newUp.Mul(alpha)).Normalize()
		cam.Right = cam.oldRight.Mul(1 - alpha).Add(newRight.Mul(alpha)).Normalize()

		// Incrémenter le temps de transition
		cam.CurrentSwappingTime++
		if cam.CurrentSwappingTime >= cam.MaxSwappingTime {
			// Fin de la transition : adopte complètement les nouveaux vecteurs
			cam.Front = newFront
			cam.Up = newUp
			cam.Right = newRight
			cam.IsSwappingFace = false
			cam.CurrentSwappingTime = 0
		}
	} else {
		// Pas de transition en cours, mise à jour directe avec interpolation "rapide" (si désiré)
		// Vous pouvez ici conserver l'interpolation douce (alpha fixe) ou l'appliquer directement :
		alpha := float32(1)
		cam.Front = cam.Front.Mul(1 - alpha).Add(newFront.Mul(alpha)).Normalize()
		cam.Up = cam.Up.Mul(1 - alpha).Add(newUp.Mul(alpha)).Normalize()
		cam.Right = cam.Right.Mul(1 - alpha).Add(newRight.Mul(alpha)).Normalize()
	}
}

func (camera *Camera) SwapVectors(oldFace WorldFace, newFace WorldFace) {
	if camera.IsSwappingFace {

	}
}

//////////////////////////////////////////////////////////
// Gestion des entrées
//////////////////////////////////////////////////////////

// ProcessMouseMovement met à jour les angles canoniques selon les offsets de la souris.
// Ces angles sont ensuite transformés dans updateVectors().
func (cam *Camera) ProcessMouseMovement(xoffset, yoffset float64) {
	// Ici, on prend directement xoffset et yoffset (appliquant MouseSens)
	cam.Yaw += float32(xoffset) * cam.MouseSens
	cam.Pitch += float32(yoffset) * cam.MouseSens

	// Clamp du pitch pour éviter le gimbal lock
	if cam.Pitch > 89.0 {
		cam.Pitch = 89.0
	}
	if cam.Pitch < -89.0 {
		cam.Pitch = -89.0
	}
}

// HandlerCursorPosCallback gère le déplacement de la souris.
func (cam *Camera) HandlerCursorPosCallback(w *glfw.Window, xpos, ypos float64) {
	if w.GetInputMode(glfw.CursorMode) == glfw.CursorNormal {
		return
	}
	if cam.firstMouse {
		cam.lastX = xpos
		cam.lastY = ypos
		cam.firstMouse = false
	}
	xoffset := xpos - cam.lastX
	yoffset := cam.lastY - ypos // inversion Y
	cam.lastX = xpos
	cam.lastY = ypos
	cam.ProcessMouseMovement(xoffset, yoffset)
}

//////////////////////////////////////////////////////////
// Autres méthodes de la caméra
//////////////////////////////////////////////////////////

func (cam *Camera) GetViewMatrix() mgl32.Mat4 {
	return mgl32.LookAtV(cam.Position, cam.Position.Add(cam.Front), cam.Up)
}

func (cam *Camera) horizontalMovementMultiplier() float32 {
	switch cam.CurrentWorldFace {
	case WorldFaceBack:
		return 1
	default:
		return 1
	}
}

func (cam *Camera) ProcessKeyboard(direction string, deltaTime float32) {
	velocity := cam.MoveSpeed * deltaTime
	horizMultiplier := cam.horizontalMovementMultiplier()

	switch direction {
	case "FORWARD":
		cam.Position = cam.Position.Add(cam.Front.Mul(velocity))
	case "BACKWARD":
		cam.Position = cam.Position.Sub(cam.Front.Mul(velocity))
	case "LEFT":
		cam.Position = cam.Position.Sub(cam.Right.Mul(velocity * horizMultiplier))
	case "RIGHT":
		cam.Position = cam.Position.Add(cam.Right.Mul(velocity * horizMultiplier))
	}
}

func (cam *Camera) Update(window *glfw.Window, deltaTime float32) {
	if window.GetKey(glfw.KeyEscape) == glfw.Press {
		window.SetInputMode(glfw.CursorMode, glfw.CursorNormal)
	}
	if window.GetMouseButton(glfw.MouseButton1) == glfw.Press {
		window.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
	}
	if window.GetKey(glfw.KeyW) == glfw.Press {
		cam.ProcessKeyboard("FORWARD", deltaTime)
	}
	if window.GetKey(glfw.KeyS) == glfw.Press {
		cam.ProcessKeyboard("BACKWARD", deltaTime)
	}
	if window.GetKey(glfw.KeyA) == glfw.Press {
		cam.ProcessKeyboard("LEFT", deltaTime)
	}
	if window.GetKey(glfw.KeyD) == glfw.Press {
		cam.ProcessKeyboard("RIGHT", deltaTime)
	}

	if window.GetKey(glfw.KeyC) == glfw.Press {
		cam.FitWorld = !cam.FitWorld
	}
	if window.GetKey(glfw.KeyLeftShift) == glfw.Press {
		cam.MoveSpeed = 100
	} else {
		cam.MoveSpeed = 5
	}

	if cam.FitWorld {
		newFace := cam.GetDynamicCurrentWorldFaceByPosition(float32(len(cam.World.Chunks) * 16))
		if newFace != cam.CurrentWorldFace {
			cam.StartFaceSwap(newFace)
		}
	}

	cam.updateVectors()
}

func (cam *Camera) StartFaceSwap(newFace WorldFace) {
	// Si un swap est déjà en cours, on ne fait rien
	if cam.IsSwappingFace {
		return
	}
	// Stocker les vecteurs actuels comme vecteurs de départ
	cam.oldFront = cam.Front
	cam.oldUp = cam.Up
	cam.oldRight = cam.Right

	if cam.CurrentWorldFace == WorldFaceTop && newFace == WorldFaceFront {
		cam.Yaw = 180
	}

	if cam.CurrentWorldFace == WorldFaceTop && newFace == WorldFaceLeft {
		cam.Yaw = 180
	}

	if cam.CurrentWorldFace == WorldFaceFront && newFace == WorldFaceRight {
		cam.Yaw = -cam.Yaw
	}

	if cam.CurrentWorldFace == WorldFaceFront && newFace == WorldFaceBottom {
		cam.Yaw = 90
	}

	if cam.CurrentWorldFace == WorldFaceFront && newFace == WorldFaceTop {
		cam.Yaw = -90
	}

	if cam.CurrentWorldFace == WorldFaceBack && newFace == WorldFaceRight {
		cam.Yaw = 90
	}

	if cam.CurrentWorldFace == WorldFaceRight && newFace == WorldFaceBack {
		cam.Yaw = 90
	}

	if cam.CurrentWorldFace == WorldFaceRight && newFace == WorldFaceFront {
		cam.Yaw = -cam.Yaw
	}

	if cam.CurrentWorldFace == WorldFaceRight && newFace == WorldFaceBottom {
		cam.Yaw = 180
	}

	if cam.CurrentWorldFace == WorldFaceBottom && newFace == WorldFaceBack {
		cam.Yaw = 0
	}

	if cam.CurrentWorldFace == WorldFaceBottom && newFace == WorldFaceFront {
		cam.Yaw = 0
	}

	if cam.CurrentWorldFace == WorldFaceBottom && newFace == WorldFaceLeft {
		cam.Yaw = 0
	}

	if cam.CurrentWorldFace == WorldFaceBack && newFace == WorldFaceBottom {
		cam.Yaw = -90
	}

	if cam.CurrentWorldFace == WorldFaceBack && newFace == WorldFaceTop {
		cam.Yaw = 90
	}

	if cam.CurrentWorldFace == WorldFaceLeft && newFace == WorldFaceBottom {
		cam.Yaw = 0
	}

	if cam.CurrentWorldFace == WorldFaceBottom && newFace == WorldFaceLeft {
		cam.Yaw = 0
	}

	cam.Pitch = -15

	// Définir la nouvelle face cible
	cam.CurrentWorldFace = newFace

	// Démarrer la transition
	cam.IsSwappingFace = true
	cam.CurrentSwappingTime = 0
}

func (cam *Camera) GetLookFace() WorldFace {
	// On travaille avec le vecteur Front, déjà normalisé.
	d := cam.Front.Normalize()

	candidate := cam.CurrentWorldFace // valeur par défaut, qui sera remplacée
	bestDot := float32(-1)
	for i := 0; i < len(cubeFaceNormals); i++ {
		dot := d.Dot(cubeFaceNormals[i].normal)
		if dot > bestDot {
			bestDot = dot
			candidate = cubeFaceNormals[i].face
		}
	}
	return candidate
}

func (cam *Camera) GetDynamicCurrentWorldFaceByPosition(_ float32) WorldFace {
	pos := cam.Position

	// Utiliser LenSqr pour éviter un appel à sqrt (sauf si nécessaire)
	if pos.LenSqr() == 0 {
		return cam.CurrentWorldFace
	}
	// Normaliser la direction depuis le centre
	d := pos.Normalize()

	// Recherche du candidat en parcourant le tableau global cubeFaceNormals
	candidate := cam.CurrentWorldFace
	bestDot := float32(-1)

	for i := 0; i < len(cubeFaceNormals); i++ {
		dot := d.Dot(cubeFaceNormals[i].normal)
		if dot > bestDot {
			bestDot = dot
			candidate = cubeFaceNormals[i].face
		}
	}

	// Récupérer la normale correspondant à la face actuellement active.
	var currentNormal mgl32.Vec3
	switch cam.CurrentWorldFace {
	case WorldFaceTop:
		currentNormal = mgl32.Vec3{0, 1, 0}
	case WorldFaceBottom:
		currentNormal = mgl32.Vec3{0, -1, 0}
	case WorldFaceRight:
		currentNormal = mgl32.Vec3{1, 0, 0}
	case WorldFaceLeft:
		currentNormal = mgl32.Vec3{-1, 0, 0}
	case WorldFaceFront:
		currentNormal = mgl32.Vec3{0, 0, 1}
	case WorldFaceBack:
		currentNormal = mgl32.Vec3{0, 0, -1}
	}

	currentDot := d.Dot(currentNormal)

	// Désactiver le log en production ou conditionner son exécution
	// log.Printf("Candidate %s CurrentDot: %v BestDot: %v\n", candidate.ToString(), currentDot, bestDot)

	// Seuil d'hystérésis : ne changer que si la différence est suffisamment marquée
	if candidate != cam.CurrentWorldFace && bestDot < currentDot {
		candidate = cam.CurrentWorldFace
	}

	return candidate
}
