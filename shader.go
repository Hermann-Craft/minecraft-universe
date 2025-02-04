package main

import (
	"fmt"
	"os"

	"github.com/go-gl/gl/v4.6-core/gl"
)

type Shader struct {
	ID uint32
}

func LoadShader(name string) (*Shader, error) {
	shader := &Shader{}

	defaultFragSource, err := os.ReadFile(fmt.Sprintf("./shaders/%s.frag", name))
	if err != nil {
		return nil, err
	}

	defaultVertSource, err := os.ReadFile(fmt.Sprintf("./shaders/%s.vert", name))
	if err != nil {
		return nil, err
	}

	defaultFragSourceString := string(defaultFragSource)

	defaultVertSourceString := string(defaultVertSource)

	vertexShader := gl.CreateShader(gl.VERTEX_SHADER)
	vertexShaderSource, free := gl.Strs(defaultVertSourceString)
	gl.ShaderSource(vertexShader, 1, vertexShaderSource, nil)
	free()
	gl.CompileShader(vertexShader)

	fragmentShader := gl.CreateShader(gl.FRAGMENT_SHADER)
	fragmentShaderSource, free := gl.Strs(defaultFragSourceString)
	gl.ShaderSource(fragmentShader, 1, fragmentShaderSource, nil)
	free()
	gl.CompileShader(fragmentShader)

	shader.ID = gl.CreateProgram()

	gl.AttachShader(shader.ID, vertexShader)
	gl.AttachShader(shader.ID, fragmentShader)
	gl.LinkProgram(shader.ID)

	gl.DeleteShader(vertexShader)
	gl.DeleteShader(fragmentShader)

	return shader, nil
}

func (shader *Shader) Activate() {
	gl.UseProgram(shader.ID)
}

func (shader *Shader) Delete() {
	gl.DeleteProgram(shader.ID)
}
