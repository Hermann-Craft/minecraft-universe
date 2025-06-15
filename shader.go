package main

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/go-gl/gl/v4.6-core/gl"
)

type Shader struct {
	ID uint32
}

var (
	shaderCache     = make(map[string]*Shader)
	shaderCacheLock sync.RWMutex
)

// checkGLError vérifie s'il y a une erreur OpenGL et la retourne si c'est le cas
func checkGLError() error {
	err := gl.GetError()
	if err != gl.NO_ERROR {
		return fmt.Errorf("OpenGL error: %v", err)
	}
	return nil
}

func LoadShader(name string) (*Shader, error) {
	// Vérifier d'abord dans le cache
	shaderCacheLock.RLock()
	if shader, exists := shaderCache[name]; exists {
		shaderCacheLock.RUnlock()
		return shader, nil
	}
	shaderCacheLock.RUnlock()

	// Si pas dans le cache, charger le shader
	shaderCacheLock.Lock()
	defer shaderCacheLock.Unlock()

	// Vérifier à nouveau (un autre thread pourrait l'avoir chargé entre temps)
	if shader, exists := shaderCache[name]; exists {
		return shader, nil
	}

	log.Printf("Loading shader: %s", name)
	shader := &Shader{}

	defaultFragSource, err := os.ReadFile(fmt.Sprintf("./shaders/%s.frag", name))
	if err != nil {
		log.Printf("Failed to read fragment shader file: %v", err)
		return nil, fmt.Errorf("failed to read fragment shader file: %v", err)
	}
	log.Printf("Successfully read fragment shader file")

	defaultVertSource, err := os.ReadFile(fmt.Sprintf("./shaders/%s.vert", name))
	if err != nil {
		log.Printf("Failed to read vertex shader file: %v", err)
		return nil, fmt.Errorf("failed to read vertex shader file: %v", err)
	}
	log.Printf("Successfully read vertex shader file")

	defaultFragSourceString := string(defaultFragSource)
	defaultVertSourceString := string(defaultVertSource)

	// Compiler le vertex shader
	vertexShader := gl.CreateShader(gl.VERTEX_SHADER)
	if err := checkGLError(); err != nil {
		return nil, fmt.Errorf("failed to create vertex shader: %v", err)
	}

	vertexShaderSource, free := gl.Strs(defaultVertSourceString)
	gl.ShaderSource(vertexShader, 1, vertexShaderSource, nil)
	free()
	if err := checkGLError(); err != nil {
		gl.DeleteShader(vertexShader)
		return nil, fmt.Errorf("failed to set vertex shader source: %v", err)
	}

	gl.CompileShader(vertexShader)
	if err := checkGLError(); err != nil {
		gl.DeleteShader(vertexShader)
		return nil, fmt.Errorf("failed to compile vertex shader: %v", err)
	}

	// Vérifier la compilation du vertex shader
	var success int32
	gl.GetShaderiv(vertexShader, gl.COMPILE_STATUS, &success)
	if success == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(vertexShader, gl.INFO_LOG_LENGTH, &logLength)
		logInfo := make([]byte, logLength)
		gl.GetShaderInfoLog(vertexShader, logLength, nil, &logInfo[0])
		gl.DeleteShader(vertexShader)
		log.Printf("Failed to compile vertex shader: %s", logInfo)
		return nil, fmt.Errorf("failed to compile vertex shader: %s", logInfo)
	}
	log.Printf("Successfully compiled vertex shader")

	// Compiler le fragment shader
	fragmentShader := gl.CreateShader(gl.FRAGMENT_SHADER)
	if err := checkGLError(); err != nil {
		gl.DeleteShader(vertexShader)
		return nil, fmt.Errorf("failed to create fragment shader: %v", err)
	}

	fragmentShaderSource, free := gl.Strs(defaultFragSourceString)
	gl.ShaderSource(fragmentShader, 1, fragmentShaderSource, nil)
	free()
	if err := checkGLError(); err != nil {
		gl.DeleteShader(vertexShader)
		gl.DeleteShader(fragmentShader)
		return nil, fmt.Errorf("failed to set fragment shader source: %v", err)
	}

	gl.CompileShader(fragmentShader)
	if err := checkGLError(); err != nil {
		gl.DeleteShader(vertexShader)
		gl.DeleteShader(fragmentShader)
		return nil, fmt.Errorf("failed to compile fragment shader: %v", err)
	}

	// Vérifier la compilation du fragment shader
	gl.GetShaderiv(fragmentShader, gl.COMPILE_STATUS, &success)
	if success == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(fragmentShader, gl.INFO_LOG_LENGTH, &logLength)
		logInfo := make([]byte, logLength+1)
		gl.GetShaderInfoLog(fragmentShader, logLength, nil, &logInfo[0])
		gl.DeleteShader(vertexShader)
		gl.DeleteShader(fragmentShader)
		log.Printf("Failed to compile fragment shader: %s", logInfo)
		return nil, fmt.Errorf("failed to compile fragment shader: %s", logInfo)
	}
	log.Printf("Successfully compiled fragment shader")

	// Créer et lier le programme
	shader.ID = gl.CreateProgram()
	if err := checkGLError(); err != nil {
		gl.DeleteShader(vertexShader)
		gl.DeleteShader(fragmentShader)
		return nil, fmt.Errorf("failed to create shader program: %v", err)
	}

	gl.AttachShader(shader.ID, vertexShader)
	if err := checkGLError(); err != nil {
		gl.DeleteShader(vertexShader)
		gl.DeleteShader(fragmentShader)
		gl.DeleteProgram(shader.ID)
		return nil, fmt.Errorf("failed to attach vertex shader: %v", err)
	}

	gl.AttachShader(shader.ID, fragmentShader)
	if err := checkGLError(); err != nil {
		gl.DeleteShader(vertexShader)
		gl.DeleteShader(fragmentShader)
		gl.DeleteProgram(shader.ID)
		return nil, fmt.Errorf("failed to attach fragment shader: %v", err)
	}

	gl.LinkProgram(shader.ID)
	if err := checkGLError(); err != nil {
		gl.DeleteShader(vertexShader)
		gl.DeleteShader(fragmentShader)
		gl.DeleteProgram(shader.ID)
		return nil, fmt.Errorf("failed to link shader program: %v", err)
	}

	// Vérifier la liaison du programme
	gl.GetProgramiv(shader.ID, gl.LINK_STATUS, &success)
	if success == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(shader.ID, gl.INFO_LOG_LENGTH, &logLength)
		logInfo := make([]byte, logLength)
		gl.GetShaderInfoLog(shader.ID, logLength, nil, &logInfo[0])
		gl.DeleteShader(vertexShader)
		gl.DeleteShader(fragmentShader)
		gl.DeleteProgram(shader.ID)
		log.Printf("Failed to link shader program: %s", logInfo)
		return nil, fmt.Errorf("failed to link shader program: %s", logInfo)
	}

	// Nettoyer les shaders individuels car ils sont maintenant liés au programme
	gl.DeleteShader(vertexShader)
	gl.DeleteShader(fragmentShader)

	// Ajouter au cache
	shaderCache[name] = shader
	log.Printf("Successfully linked shader program")

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

// CleanupShaders nettoie tous les shaders du cache
func CleanupShaders() {
	shaderCacheLock.Lock()
	defer shaderCacheLock.Unlock()

	for _, shader := range shaderCache {
		gl.DeleteProgram(shader.ID)
	}
	shaderCache = make(map[string]*Shader)
}
