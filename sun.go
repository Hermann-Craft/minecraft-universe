package main

import (
	"fmt"

	"github.com/go-gl/gl/v4.6-core/gl"
	"github.com/go-gl/mathgl/mgl32"
	goecs "github.com/oneforx/go-ecs"
)

type Sun struct {
	*goecs.Identifier
	UniversePosition *mgl32.Vec3
	UniverseRotation *mgl32.Quat

	// Propriétés du soleil
	Radius    float32
	Color     mgl32.Vec3
	Intensity float32

	// Technicals
	isInit bool
}

func (sun *Sun) Init(identifier goecs.Identifier, position mgl32.Vec3, rotation mgl32.Quat) error {
	sun.Identifier = &identifier
	sun.UniversePosition = &position
	sun.UniverseRotation = &rotation

	// Initialiser les propriétés du soleil
	sun.Radius = 100.0                     // Rayon du soleil
	sun.Color = mgl32.Vec3{1.0, 0.95, 0.8} // Couleur jaune-orangé
	sun.Intensity = 1.0                    // Intensité de la lumière

	sun.isInit = true
	return nil
}

func (sun *Sun) IsInit() bool {
	return sun.isInit
}

func (sun *Sun) Render(currentTime float64, galaxy *Galaxy) {
	if !sun.isInit {
		return
	}

	// Mettre à jour la position du soleil dans les shaders de tous les chunks
	for _, planet := range galaxy.Planets {
		for _, chunkRow := range planet.Chunks {
			for _, chunkCol := range chunkRow {
				for _, chunk := range chunkCol {
					if chunk != nil && chunk.ShaderProgram != 0 {
						gl.UseProgram(chunk.ShaderProgram)

						// Mettre à jour la position du soleil
						sunPosLoc := gl.GetUniformLocation(chunk.ShaderProgram, gl.Str("sunPos\x00"))
						if sunPosLoc != -1 {
							gl.Uniform3f(sunPosLoc, sun.UniversePosition.X(), sun.UniversePosition.Y(), sun.UniversePosition.Z())
						}

						// Mettre à jour la couleur du soleil
						sunColorLoc := gl.GetUniformLocation(chunk.ShaderProgram, gl.Str("sunColor\x00"))
						if sunColorLoc != -1 {
							gl.Uniform3f(sunColorLoc, sun.Color.X(), sun.Color.Y(), sun.Color.Z())
						}

						// Mettre à jour l'intensité du soleil
						sunAmbientLoc := gl.GetUniformLocation(chunk.ShaderProgram, gl.Str("sunAmbient\x00"))
						if sunAmbientLoc != -1 {
							gl.Uniform1f(sunAmbientLoc, sun.Intensity)
						}
					}
				}
			}
		}
	}
}

var ErrSunNotInitialized = fmt.Errorf("the sun is not initialized")
