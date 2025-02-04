package main

import (
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/go-gl/gl/v4.1-core/gl" // OpenGL 4.1
	"github.com/go-gl/glfw/v3.3/glfw"  // GLFW
	"github.com/go-gl/mathgl/mgl32"    // Maths (vecteurs, matrices)
)

func init() {
	// Tous les appels OpenGL doivent se faire dans le thread principal.
	runtime.LockOSThread()
}

//////////////////////////////////////////////////////////
// FRUSTUM CULLING : Définitions et fonctions (inchangées)
//////////////////////////////////////////////////////////

type Plane struct {
	Normal   mgl32.Vec3
	Distance float32
}

func extractFrustumPlanes(vp mgl32.Mat4) [6]Plane {
	var planes [6]Plane
	planes[0] = Plane{
		Normal:   mgl32.Vec3{vp[3] + vp[0], vp[7] + vp[4], vp[11] + vp[8]},
		Distance: vp[15] + vp[12],
	}
	planes[1] = Plane{
		Normal:   mgl32.Vec3{vp[3] - vp[0], vp[7] - vp[4], vp[11] - vp[8]},
		Distance: vp[15] - vp[12],
	}
	planes[2] = Plane{
		Normal:   mgl32.Vec3{vp[3] + vp[1], vp[7] + vp[5], vp[11] + vp[9]},
		Distance: vp[15] + vp[13],
	}
	planes[3] = Plane{
		Normal:   mgl32.Vec3{vp[3] - vp[1], vp[7] - vp[5], vp[11] - vp[9]},
		Distance: vp[15] - vp[13],
	}
	planes[4] = Plane{
		Normal:   mgl32.Vec3{vp[3] + vp[2], vp[7] + vp[6], vp[11] + vp[10]},
		Distance: vp[15] + vp[14],
	}
	planes[5] = Plane{
		Normal:   mgl32.Vec3{vp[3] - vp[2], vp[7] - vp[6], vp[11] - vp[10]},
		Distance: vp[15] - vp[14],
	}
	for i := 0; i < 6; i++ {
		norm := planes[i].Normal.Len()
		if norm != 0 {
			planes[i].Normal = planes[i].Normal.Mul(1.0 / norm)
			planes[i].Distance /= norm
		}
	}
	return planes
}

func cubeInFrustum(planes [6]Plane, center mgl32.Vec3, halfSize float32) bool {
	for _, plane := range planes {
		px := center.X()
		py := center.Y()
		pz := center.Z()
		if plane.Normal.X() >= 0 {
			px += halfSize
		} else {
			px -= halfSize
		}
		if plane.Normal.Y() >= 0 {
			py += halfSize
		} else {
			py -= halfSize
		}
		if plane.Normal.Z() >= 0 {
			pz += halfSize
		} else {
			pz -= halfSize
		}
		positiveVertex := mgl32.Vec3{px, py, pz}
		if plane.Normal.Dot(positiveVertex)+plane.Distance < 0 {
			return false
		}
	}
	return true
}

//////////////////////////////////////////////////////////
// Fonction principale
//////////////////////////////////////////////////////////

type Game struct {

	// Technicals
	window       *glfw.Window
	textRenderer *TextRenderer
}

func main() {
	game := &Game{}

	if err := glfw.Init(); err != nil {
		log.Fatalln("Erreur lors de l'initialisation de GLFW:", err)
	}
	defer glfw.Terminate()
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	var err error
	game.window, err = glfw.CreateWindow(800, 600, "Clone Minecraft - Phong Lighting", nil, nil)
	if err != nil {
		log.Fatalln("Impossible de créer la fenêtre:", err)
	}
	game.window.MakeContextCurrent()
	game.window.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
	if err := gl.Init(); err != nil {
		log.Fatalln("Erreur lors de l'initialisation d'OpenGL:", err)
	}
	log.Println("Version d'OpenGL:", gl.GoStr(gl.GetString(gl.VERSION)))
	gl.Enable(gl.DEPTH_TEST)

	// Shader principal pour la scène

	// Chargement de la texture et de l'UI (fonctions externes)

	// inventoryTexture, err := loadTexture("resources/texture.png")
	// if err != nil {
	// 	log.Fatalln(err)
	// }
	inventoryShaderProgram, err := newProgram(inventoryVertexShaderSource, inventoryFragmentShaderSource)
	if err != nil {
		log.Fatalln(err)
	}
	initInventoryUI()

	game.textRenderer = &TextRenderer{}
	game.textRenderer.Init()

	// Création d'un chunk (fonction externe)
	world := &World{}
	world.Init(game)

	frameDuration := time.Second / 30

	previousTime := time.Now()
	for !game.window.ShouldClose() {
		frameStart := time.Now()

		// Calcul du deltaTime basé sur la frame précédente.
		currentTime := time.Now()
		deltaTime := float32(currentTime.Sub(previousTime).Seconds())
		previousTime = currentTime

		// Mise à jour de la logique du jeu
		world.Update(deltaTime)

		// Nettoyage et rendu de la scène
		gl.ClearColor(0.1, 0.2, 0.3, 1.0)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		world.Render()

		game.window.SwapBuffers()
		glfw.PollEvents()

		// Calculer le temps écoulé pour cette frame
		elapsed := time.Since(frameStart)
		// Calculer le temps à attendre pour compléter la frame
		sleepTime := frameDuration - elapsed
		if sleepTime > 0 {
			time.Sleep(sleepTime)
		}
	}

	world.Clean()
	gl.DeleteProgram(inventoryShaderProgram)
	gl.DeleteProgram(game.textRenderer.textShaderProgram)
}

//////////////////////////////////////////////////////////
// Fonctions d'aide pour les shaders
//////////////////////////////////////////////////////////

func newProgram(vertexShaderSource, fragmentShaderSource string) (uint32, error) {
	vertexShader, err := compileShader(vertexShaderSource, gl.VERTEX_SHADER)
	if err != nil {
		return 0, err
	}
	fragmentShader, err := compileShader(fragmentShaderSource, gl.FRAGMENT_SHADER)
	if err != nil {
		return 0, err
	}
	program := gl.CreateProgram()
	gl.AttachShader(program, vertexShader)
	gl.AttachShader(program, fragmentShader)
	gl.LinkProgram(program)
	var status int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLength)
		logInfo := make([]byte, logLength+1)
		gl.GetProgramInfoLog(program, logLength, nil, &logInfo[0])
		return 0, fmt.Errorf("échec du linkage du programme : %s", logInfo)
	}
	gl.DeleteShader(vertexShader)
	gl.DeleteShader(fragmentShader)
	return program, nil
}

func compileShader(source string, shaderType uint32) (uint32, error) {
	shader := gl.CreateShader(shaderType)
	csources, free := gl.Strs(source)
	gl.ShaderSource(shader, 1, csources, nil)
	free()
	gl.CompileShader(shader)
	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)
		logInfo := make([]byte, logLength+1)
		gl.GetShaderInfoLog(shader, logLength, nil, &logInfo[0])
		return 0, fmt.Errorf("échec de la compilation du shader (type %v) : %s", shaderType, logInfo)
	}
	return shader, nil
}
