package main

import (
	"log"
	"time"

	"github.com/go-gl/gl/v4.6-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
	goecs "github.com/oneforx/go-ecs"
)

var CameraInstance *Camera

type Game struct {
	*Universe
	*Camera

	// Config
	FpsLimit uint16

	// Technicals
	window       *glfw.Window
	textRenderer *TextRenderer
}

func (game *Game) Init() {
	if err := glfw.Init(); err != nil {
		log.Fatalln("Erreur lors de l'initialisation de GLFW:", err)
	}
	defer glfw.Terminate()
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 6)
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

	// Activer les fonctionnalités OpenGL nécessaires
	gl.Enable(gl.DEPTH_TEST)
	gl.Enable(gl.CULL_FACE)
	gl.CullFace(gl.BACK)
	gl.FrontFace(gl.CCW)

	universe := &Universe{}

	universe.Init()

	galaxy := &Galaxy{}

	galaxy.Init(
		goecs.Identifier{Namespace: "core", Path: "default-galaxy"},
		mgl32.Vec3{0, 0, 0},
		mgl32.Quat{W: 0, V: mgl32.Vec3{0, 0, 0}},
	)

	planetSun := &Planet{}

	planetSun.Init(
		goecs.Identifier{Namespace: "core", Path: "sun-planet"},
		mgl32.Vec3{0, 0, 0},
		mgl32.Quat{W: 0, V: mgl32.Vec3{0, 0, 0}},
	)

	planetEarth := &Planet{}

	// scale := 0.0000000001 // 149597870700 m

	planetEarth.Init(
		goecs.Identifier{Namespace: "core", Path: "earth-planet"},
		mgl32.Vec3{0, 0, 0},
		mgl32.Quat{W: 0, V: mgl32.Vec3{0, 0, 0}},
	)

	// Passer la fenêtre à la planète
	planetEarth.window = game.window

	galaxy.AddPlanet(planetEarth)

	universe.AddGalaxy(*galaxy)

	// Initialiser la caméra
	CameraInstance = NewCamera(mgl32.Vec3{0, 10, 10})
	CameraInstance.Init(WorldFaceTop)
	CameraInstance.Zoom = 45.0
	CameraInstance.Pitch = -30.0
	CameraInstance.updateVectors()
	game.Camera = CameraInstance

	// Configurer les callbacks pour la caméra
	game.window.SetCursorPosCallback(game.Camera.HandlerCursorPosCallback)
	game.window.SetScrollCallback(func(w *glfw.Window, xoff float64, yoff float64) {
		game.Camera.Zoom -= float32(yoff)
		if game.Camera.Zoom < 1.0 {
			game.Camera.Zoom = 1.0
		}
		if game.Camera.Zoom > 45.0 {
			game.Camera.Zoom = 45.0
		}
	})

	frameDuration := time.Second / time.Duration(game.FpsLimit)
	previousTime := time.Now()
	startTime := previousTime

	for !game.window.ShouldClose() {
		currentTime := time.Now()
		elapsed := float64(currentTime.Sub(startTime).Seconds())
		deltaTime := float32(currentTime.Sub(previousTime).Seconds())
		previousTime = currentTime

		// UPDATE
		game.Camera.Update(game.window, deltaTime)
		planetEarth.Update(deltaTime)

		// RENDER
		// Nettoyage et rendu de la scène
		gl.ClearColor(0.1, 0.2, 0.3, 1.0)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		// Mettre à jour le frustum de la caméra
		projection := mgl32.Perspective(mgl32.DegToRad(game.Camera.Zoom), float32(800)/float32(600), 0.1, 1000.0)
		game.Camera.UpdateFrustum(projection)

		universe.Render(elapsed)

		// CLEAN
		game.window.SwapBuffers()
		glfw.PollEvents()

		// Calculer le temps écoulé pour cette frame
		elapsed = time.Since(currentTime).Seconds()
		// Calculer le temps à attendre pour compléter la frame
		sleepTime := frameDuration - time.Duration(elapsed*float64(time.Second))
		if sleepTime > 0 {
			time.Sleep(sleepTime)
		}
	}
}
