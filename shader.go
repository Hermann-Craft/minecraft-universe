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

//////////////////////////////////////////////////////////
// Fonctions d'aide pour les shaders
//////////////////////////////////////////////////////////

func NewProgram(vertexShaderSource, fragmentShaderSource string) (uint32, error) {
	vertexShader, err := CompileShader(vertexShaderSource, gl.VERTEX_SHADER)
	if err != nil {
		return 0, err
	}
	fragmentShader, err := CompileShader(fragmentShaderSource, gl.FRAGMENT_SHADER)
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

func CompileShader(source string, shaderType uint32) (uint32, error) {
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
