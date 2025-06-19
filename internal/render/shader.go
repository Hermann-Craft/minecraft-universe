package render

import (
	"fmt"
	"io/ioutil"

	"github.com/go-gl/gl/v4.6-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

// shaderImpl implémentation du Shader
type shaderImpl struct {
	ID uint32
}

// LoadShaderFromFiles charge un shader depuis des fichiers
func LoadShaderFromFiles(vertexPath, fragmentPath string) (*shaderImpl, error) {
	vertexSource, err := ioutil.ReadFile(vertexPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read vertex shader: %w", err)
	}
	fragmentSource, err := ioutil.ReadFile(fragmentPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read fragment shader: %w", err)
	}

	vertexShader, err := compileShader(string(vertexSource), gl.VERTEX_SHADER)
	if err != nil {
		return nil, fmt.Errorf("failed to compile vertex shader: %w", err)
	}
	defer gl.DeleteShader(vertexShader)

	fragmentShader, err := compileShader(string(fragmentSource), gl.FRAGMENT_SHADER)
	if err != nil {
		return nil, fmt.Errorf("failed to compile fragment shader: %w", err)
	}
	defer gl.DeleteShader(fragmentShader)

	program := gl.CreateProgram()
	gl.AttachShader(program, vertexShader)
	gl.AttachShader(program, fragmentShader)
	gl.LinkProgram(program)

	var status int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLength)
		log := make([]byte, logLength)
		gl.GetProgramInfoLog(program, logLength, nil, &log[0])
		gl.DeleteProgram(program)
		return nil, fmt.Errorf("failed to link shader program: %s", string(log))
	}

	return &shaderImpl{ID: program}, nil
}

// Activate active le shader
func (s *shaderImpl) Activate() {
	gl.UseProgram(s.ID)
}

// SetMat4 définit une matrice 4x4
func (s *shaderImpl) SetMat4(name string, value mgl32.Mat4) {
	location := gl.GetUniformLocation(s.ID, gl.Str(name+"\x00"))
	if location != -1 {
		gl.UniformMatrix4fv(location, 1, false, &value[0])
	}
}

// SetVec3 définit un vecteur 3D
func (s *shaderImpl) SetVec3(name string, value mgl32.Vec3) {
	location := gl.GetUniformLocation(s.ID, gl.Str(name+"\x00"))
	if location != -1 {
		gl.Uniform3f(location, value.X(), value.Y(), value.Z())
	}
}

// SetFloat définit un float
func (s *shaderImpl) SetFloat(name string, value float32) {
	location := gl.GetUniformLocation(s.ID, gl.Str(name+"\x00"))
	if location != -1 {
		gl.Uniform1f(location, value)
	}
}

// SetInt définit un int
func (s *shaderImpl) SetInt(name string, value int32) {
	location := gl.GetUniformLocation(s.ID, gl.Str(name+"\x00"))
	if location != -1 {
		gl.Uniform1i(location, value)
	}
}

// GetID retourne l'ID du shader
func (s *shaderImpl) GetID() uint32 {
	return s.ID
}

// Cleanup nettoie les ressources
func (s *shaderImpl) Cleanup() {
	if s.ID != 0 {
		gl.DeleteProgram(s.ID)
		s.ID = 0
	}
}

// compileShader compile un shader GLSL depuis une source
func compileShader(source string, shaderType uint32) (uint32, error) {
	shader := gl.CreateShader(shaderType)
	sourceStrs, free := gl.Strs(source + "\x00")
	defer free()
	gl.ShaderSource(shader, 1, sourceStrs, nil)
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)
		log := make([]byte, logLength)
		gl.GetShaderInfoLog(shader, logLength, nil, &log[0])
		gl.DeleteShader(shader)
		return 0, fmt.Errorf("failed to compile shader: %s", string(log))
	}
	return shader, nil
}
