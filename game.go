package main

import (
	"log"
	"time"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
)

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

	frameDuration := time.Second / time.Duration(game.FpsLimit)

	previousTime := time.Now()
	for !game.window.ShouldClose() {
		currentTime := time.Now()
		_ = float32(currentTime.Sub(previousTime).Seconds())
		previousTime = currentTime

		// UPDATE

		// RENDER
		// Nettoyage et rendu de la scène
		gl.ClearColor(0.1, 0.2, 0.3, 1.0)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		// CLEAN
		game.window.SwapBuffers()
		glfw.PollEvents()

		// Calculer le temps écoulé pour cette frame
		elapsed := time.Since(currentTime)
		// Calculer le temps à attendre pour compléter la frame
		sleepTime := frameDuration - elapsed
		if sleepTime > 0 {
			time.Sleep(sleepTime)
		}
	}

}
