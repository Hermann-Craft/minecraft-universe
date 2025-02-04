package main

import (
	"log"
	"math"
	"time"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

type Moon struct {
	Position   mgl32.Vec3
	Radius     float32
	Speed      float32
	Color      mgl32.Vec4
	Projection mgl32.Mat4

	vao uint32
	vbo uint32

	shaderProgram uint32
}

func (moon *Moon) Init() {
	// Ajustement des paramètres pour la lune
	moon.Radius = float32(150.0) // distance de la lune au centre (valeur modifiable)
	moon.Speed = float32(0.01)   // vitesse de rotation réduite par rapport au soleil
	// Couleur typique d'une lune (gris clair, avec une légère teinte bleutée)
	moon.Color = mgl32.Vec4{0, 0, 1, 1}
	moon.Projection = mgl32.Perspective(mgl32.DegToRad(45.0), 800.0/600.0, 0.1, 1000.0)

	gl.GenVertexArrays(1, &moon.vao)
	gl.GenBuffers(1, &moon.vbo)
	gl.BindVertexArray(moon.vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, moon.vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(moonCubeVertices)*4, gl.Ptr(moonCubeVertices), gl.STATIC_DRAW)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 3*4, gl.PtrOffset(0))
	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
	gl.BindVertexArray(0)

	var err error
	moon.shaderProgram, err = newProgram(moonVertexShaderSource, moonFragmentShaderSource)
	if err != nil {
		log.Fatalln(err)
	}
}

func (moon *Moon) Update(world *World) {
	// Utilisation du temps écoulé et de la vitesse du soleil
	elapsed := float32(time.Since(world.StartTime).Seconds())
	angle := elapsed*world.Sun.Speed + float32(3*math.Pi/2)

	// Calcul de la position orbitale relative au centre (calculée autour de (0,0,0))
	orbitX := moon.Radius * float32(math.Cos(float64(angle)))
	orbitY := moon.Radius * float32(math.Sin(float64(angle)))

	// Positionnement de la lune par rapport au centre du monde
	moon.Position = mgl32.Vec3{
		world.CenterPosition.X() + orbitX,
		world.CenterPosition.Y() + orbitY,
		world.CenterPosition.Z(),
	}
}

func (moon *Moon) Render(world *World) {
	gl.UseProgram(moon.shaderProgram)
	moonModel := mgl32.Translate3D(moon.Position.X(), moon.Position.Y(), moon.Position.Z()).
		Mul4(mgl32.Scale3D(2.0, 2.0, 2.0))
	moonModelLoc := gl.GetUniformLocation(moon.shaderProgram, gl.Str("model\x00"))
	moonViewLoc := gl.GetUniformLocation(moon.shaderProgram, gl.Str("view\x00"))
	moonProjLoc := gl.GetUniformLocation(moon.shaderProgram, gl.Str("projection\x00"))
	gl.UniformMatrix4fv(moonModelLoc, 1, false, &moonModel[0])
	view := world.Camera.GetViewMatrix()
	gl.UniformMatrix4fv(moonViewLoc, 1, false, &view[0])
	gl.UniformMatrix4fv(moonProjLoc, 1, false, &moon.Projection[0])

	moonColorUniformLoc := gl.GetUniformLocation(moon.shaderProgram, gl.Str("moonColor\x00"))
	gl.Uniform4f(moonColorUniformLoc, moon.Color.X(), moon.Color.Y(), moon.Color.Z(), moon.Color.W())
	gl.BindVertexArray(moon.vao)
	gl.DrawArrays(gl.TRIANGLES, 0, int32(len(moonCubeVertices)/3))
	gl.BindVertexArray(0)
}

func (moon *Moon) Clean() {
	gl.DeleteProgram(moon.shaderProgram)
}

var moonCubeVertices = []float32{
	// positions d'un cube centré à l'origine
	-0.5, -0.5, -0.5,
	0.5, -0.5, -0.5,
	0.5, 0.5, -0.5,
	0.5, 0.5, -0.5,
	-0.5, 0.5, -0.5,
	-0.5, -0.5, -0.5,

	-0.5, -0.5, 0.5,
	0.5, -0.5, 0.5,
	0.5, 0.5, 0.5,
	0.5, 0.5, 0.5,
	-0.5, 0.5, 0.5,
	-0.5, -0.5, 0.5,

	-0.5, 0.5, 0.5,
	-0.5, 0.5, -0.5,
	-0.5, -0.5, -0.5,
	-0.5, -0.5, -0.5,
	-0.5, -0.5, 0.5,
	-0.5, 0.5, 0.5,

	0.5, 0.5, 0.5,
	0.5, 0.5, -0.5,
	0.5, -0.5, -0.5,
	0.5, -0.5, -0.5,
	0.5, -0.5, 0.5,
	0.5, 0.5, 0.5,

	-0.5, -0.5, -0.5,
	0.5, -0.5, -0.5,
	0.5, -0.5, 0.5,
	0.5, -0.5, 0.5,
	-0.5, -0.5, 0.5,
	-0.5, -0.5, -0.5,

	-0.5, 0.5, -0.5,
	0.5, 0.5, -0.5,
	0.5, 0.5, 0.5,
	0.5, 0.5, 0.5,
	-0.5, 0.5, 0.5,
	-0.5, 0.5, -0.5,
}

//////////////////////////////////////////////////////////
// SHADERS POUR LA LUNE (cube)
//////////////////////////////////////////////////////////

const moonVertexShaderSource = `
#version 410 core
layout (location = 0) in vec3 aPos;
uniform mat4 model;
uniform mat4 view;
uniform mat4 projection;
void main() {
    gl_Position = projection * view * model * vec4(aPos, 1.0);
}
` + "\x00"

const moonFragmentShaderSource = `
#version 410 core
out vec4 FragColor;
uniform vec4 moonColor;
void main() {
    FragColor = moonColor;
}
` + "\x00"
