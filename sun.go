package main

import (
	"log"
	"math"
	"time"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

type Sun struct {
	Position   mgl32.Vec3
	Radius     float32
	Speed      float32
	Color      mgl32.Vec4
	Projection mgl32.Mat4

	vao uint32
	vbo uint32

	shaderProgram uint32
}

func (sun *Sun) Init() {
	sun.Radius = float32(150.0) // distance du soleil au centre
	sun.Speed = float32(0.01)   // vitesse de rotation
	sun.Color = mgl32.Vec4{1, 1, 0, 1}
	sun.Projection = mgl32.Perspective(mgl32.DegToRad(45.0), 800.0/600.0, 0.1, 1000.0)

	gl.GenVertexArrays(1, &sun.vao)
	gl.GenBuffers(1, &sun.vbo)
	gl.BindVertexArray(sun.vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, sun.vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(sunCubeVertices)*4, gl.Ptr(sunCubeVertices), gl.STATIC_DRAW)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 3*4, gl.PtrOffset(0))
	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
	gl.BindVertexArray(0)

	var err error
	sun.shaderProgram, err = NewProgram(sunVertexShaderSource, sunFragmentShaderSource)
	if err != nil {
		log.Fatalln(err)
	}

}

func (sun *Sun) Update(world *World) {
	// Calcul du temps écoulé et de l'angle de rotation
	elapsed := float32(time.Since(world.StartTime).Seconds())
	angle := elapsed*sun.Speed + float32(math.Pi/2)

	// Calcul de la position orbitale relative au centre (calculée autour de (0,0,0))
	orbitX := sun.Radius * float32(math.Cos(float64(angle)))
	orbitY := sun.Radius * float32(math.Sin(float64(angle)))

	// On peut appliquer un offset vertical (par exemple, pour lever ou baisser l'orbite)

	// On translate la position orbitale par la position centrale du monde
	sun.Position = mgl32.Vec3{
		world.CenterPosition.X() + orbitX,
		world.CenterPosition.Y() + orbitY,
		world.CenterPosition.Z(), // On garde la même coordonnée en Z
	}
}

func (sun *Sun) Render(world *World) {
	gl.UseProgram(sun.shaderProgram)
	sunModel := mgl32.Translate3D(sun.Position.X(), sun.Position.Y(), sun.Position.Z()).
		Mul4(mgl32.Scale3D(2.0, 2.0, 2.0))
	sunModelLoc := gl.GetUniformLocation(sun.shaderProgram, gl.Str("model\x00"))
	sunViewLoc := gl.GetUniformLocation(sun.shaderProgram, gl.Str("view\x00"))
	sunProjLoc := gl.GetUniformLocation(sun.shaderProgram, gl.Str("projection\x00"))
	gl.UniformMatrix4fv(sunModelLoc, 1, false, &sunModel[0])
	view := world.Camera.GetViewMatrix()
	gl.UniformMatrix4fv(sunViewLoc, 1, false, &view[0])

	gl.UniformMatrix4fv(sunProjLoc, 1, false, &sun.Projection[0])
	sunColorUniformLoc := gl.GetUniformLocation(sun.shaderProgram, gl.Str("sunColor\x00"))
	gl.Uniform4f(sunColorUniformLoc, sun.Color.X(), sun.Color.Y(), sun.Color.Z(), sun.Color.W())
	gl.BindVertexArray(sun.vao)
	gl.DrawArrays(gl.TRIANGLES, 0, int32(len(sunCubeVertices)/3))
	gl.BindVertexArray(0)
}

func (sun *Sun) Clean() {
	gl.DeleteProgram(sun.shaderProgram)
}

var sunCubeVertices = []float32{
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
// SHADERS POUR LE SOLEIL (cube)
//////////////////////////////////////////////////////////

const sunVertexShaderSource = `
#version 410 core
layout (location = 0) in vec3 aPos;
uniform mat4 model;
uniform mat4 view;
uniform mat4 projection;
void main() {
    gl_Position = projection * view * model * vec4(aPos, 1.0);
}
` + "\x00"

const sunFragmentShaderSource = `
#version 410 core
out vec4 FragColor;
uniform vec4 sunColor;
void main() {
    FragColor = sunColor;
}
` + "\x00"
