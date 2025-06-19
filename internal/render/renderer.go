package render

import (
	"fmt"
	"log"
	"sync"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/hermann-craft/unicube/internal/core/geom"
)

// Renderer gère le rendu de la scène
type Renderer struct {
	window     *glfw.Window
	shaders    map[string]*shaderImpl
	meshes     *meshManagerImpl
	textures   *textureManagerImpl
	stats      *RenderStats
	clearColor mgl32.Vec4
	mu         sync.RWMutex
}

// RenderStats statistiques de rendu
type RenderStats struct {
	DrawCalls      int
	Triangles      int
	Vertices       int
	FPS            float64
	FrameTime      float64
	ChunksRendered int
}

// Renderable interface pour les objets rendables
type Renderable interface {
	Render(renderer *Renderer, view, projection mgl32.Mat4)
	GetPosition() mgl32.Vec3
	GetBoundingBox() geom.BoundingBox
	IsVisible() bool
}

// NewRenderer crée un nouveau renderer
func NewRenderer(window *glfw.Window) (*Renderer, error) {
	if window == nil {
		return nil, fmt.Errorf("window cannot be nil")
	}

	renderer := &Renderer{
		window:     window,
		shaders:    make(map[string]*shaderImpl),
		clearColor: mgl32.Vec4{0.1, 0.2, 0.3, 1.0},
		stats:      &RenderStats{},
	}

	// Initialiser les gestionnaires
	renderer.meshes = NewMeshManager()
	renderer.textures = NewTextureManager()

	// Configurer OpenGL
	if err := renderer.setupOpenGL(); err != nil {
		return nil, fmt.Errorf("failed to setup OpenGL: %w", err)
	}

	return renderer, nil
}

// setupOpenGL configure OpenGL
func (renderer *Renderer) setupOpenGL() error {
	renderer.window.MakeContextCurrent()

	// Initialiser OpenGL
	if err := gl.Init(); err != nil {
		return fmt.Errorf("failed to initialize OpenGL: %w", err)
	}

	// Logs détaillés pour la compatibilité
	glVersion := gl.GoStr(gl.GetString(gl.VERSION))
	glRenderer := gl.GoStr(gl.GetString(gl.RENDERER))
	glVendor := gl.GoStr(gl.GetString(gl.VENDOR))
	glslVersion := gl.GoStr(gl.GetString(gl.SHADING_LANGUAGE_VERSION))

	log.Printf("OpenGL Version: %s", glVersion)
	log.Printf("OpenGL Renderer: %s", glRenderer)
	log.Printf("OpenGL Vendor: %s", glVendor)
	log.Printf("GLSL Version: %s", glslVersion)

	// Vérification de la compatibilité OpenGL 3.3
	var majorVersion, minorVersion int32
	gl.GetIntegerv(gl.MAJOR_VERSION, &majorVersion)
	gl.GetIntegerv(gl.MINOR_VERSION, &minorVersion)
	log.Printf("OpenGL Context Version: %d.%d", majorVersion, minorVersion)

	if majorVersion < 3 || (majorVersion == 3 && minorVersion < 3) {
		log.Printf("WARNING: OpenGL 3.3+ required, but only %d.%d available", majorVersion, minorVersion)
		return fmt.Errorf("OpenGL 3.3 or higher required, but only %d.%d available", majorVersion, minorVersion)
	}

	log.Println("✅ OpenGL 3.3+ compatibility confirmed")

	// Activer les fonctionnalités
	gl.Enable(gl.DEPTH_TEST)
	gl.Enable(gl.CULL_FACE)
	gl.CullFace(gl.BACK)
	gl.FrontFace(gl.CCW)

	// Activer le blending pour la transparence
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	return nil
}

// LoadShader charge un shader
func (renderer *Renderer) LoadShader(name, vertexPath, fragmentPath string) error {
	renderer.mu.Lock()
	defer renderer.mu.Unlock()

	shader, err := LoadShaderFromFiles(vertexPath, fragmentPath)
	if err != nil {
		return fmt.Errorf("failed to load shader %s: %w", name, err)
	}

	renderer.shaders[name] = shader
	return nil
}

// GetShader retourne un shader par nom
func (renderer *Renderer) GetShader(name string) (*shaderImpl, bool) {
	renderer.mu.RLock()
	defer renderer.mu.RUnlock()

	shader, exists := renderer.shaders[name]
	return shader, exists
}

// SetClearColor définit la couleur de fond
func (renderer *Renderer) SetClearColor(color mgl32.Vec4) {
	renderer.mu.Lock()
	defer renderer.mu.Unlock()
	renderer.clearColor = color
}

// BeginFrame commence une frame de rendu
func (renderer *Renderer) BeginFrame() {
	// Nettoyer les statistiques
	renderer.stats.DrawCalls = 0
	renderer.stats.Triangles = 0
	renderer.stats.Vertices = 0
	renderer.stats.ChunksRendered = 0

	// Nettoyer les buffers
	gl.ClearColor(renderer.clearColor.X(), renderer.clearColor.Y(), renderer.clearColor.Z(), renderer.clearColor.W())
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
}

// EndFrame termine une frame de rendu
func (renderer *Renderer) EndFrame() {
	// Échanger les buffers
	renderer.window.SwapBuffers()
}

// RenderScene rend une scène complète
func (renderer *Renderer) RenderScene(scene Scene, camera Camera, projection mgl32.Mat4) {
	view := camera.GetViewMatrix()

	// Rendu des objets visibles
	for _, renderable := range scene.GetRenderables() {
		if renderable.IsVisible() {
			renderable.Render(renderer, view, projection)
			renderer.stats.DrawCalls++
		}
	}
}

// GetStats retourne les statistiques de rendu
func (renderer *Renderer) GetStats() RenderStats {
	renderer.mu.RLock()
	defer renderer.mu.RUnlock()
	return *renderer.stats
}

// GetMeshManager retourne le gestionnaire de meshes
func (renderer *Renderer) GetMeshManager() *meshManagerImpl {
	return renderer.meshes
}

// GetTextureManager retourne le gestionnaire de textures
func (renderer *Renderer) GetTextureManager() *textureManagerImpl {
	return renderer.textures
}

// Cleanup nettoie les ressources
func (renderer *Renderer) Cleanup() {
	renderer.mu.Lock()
	defer renderer.mu.Unlock()

	// Nettoyer les shaders
	for _, shader := range renderer.shaders {
		shader.Cleanup()
	}

	// Nettoyer les gestionnaires
	if renderer.meshes != nil {
		renderer.meshes.CleanupAll()
	}
	if renderer.textures != nil {
		renderer.textures.Cleanup()
	}
}

// Scene interface pour une scène rendable
type Scene interface {
	GetRenderables() []Renderable
	Update(deltaTime float32)
}

// Camera interface pour une caméra
type Camera interface {
	GetViewMatrix() mgl32.Mat4
	GetPosition() mgl32.Vec3
	GetFront() mgl32.Vec3
	GetFrustum() *Frustum
}
