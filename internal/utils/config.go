package utils

import (
	"fmt"
	"os"
	"strconv"
)

// Config représente la configuration globale de l'application
type Config struct {
	Window      WindowConfig
	Graphics    GraphicsConfig
	World       WorldConfig
	Performance PerformanceConfig
}

// WindowConfig configuration de la fenêtre
type WindowConfig struct {
	Width      int
	Height     int
	Title      string
	Fullscreen bool
	Resizable  bool
}

// GraphicsConfig configuration graphique
type GraphicsConfig struct {
	VSync       bool
	MSAA        int
	MaxFPS      int
	OpenGLMajor int
	OpenGLMinor int
}

// WorldConfig configuration du monde
type WorldConfig struct {
	ChunkSize   int
	ChunkHeight int
	PlanetSize  [3]int
	Seed        int64
}

// PerformanceConfig configuration des performances
type PerformanceConfig struct {
	MaxChunkLoadDistance float32
	ChunkLoadWorkers     int
	EnableFrustumCulling bool
	EnableMeshCaching    bool
}

// LoadConfig charge la configuration depuis les variables d'environnement ou utilise les valeurs par défaut
func LoadConfig() *Config {
	return &Config{
		Window: WindowConfig{
			Width:      getEnvInt("WINDOW_WIDTH", 1920),
			Height:     getEnvInt("WINDOW_HEIGHT", 1080),
			Title:      getEnvString("WINDOW_TITLE", "Minecraft Universe - OpenGL 3.3 Compatible"),
			Fullscreen: getEnvBool("WINDOW_FULLSCREEN", false),
			Resizable:  getEnvBool("WINDOW_RESIZABLE", true),
		},
		Graphics: GraphicsConfig{
			VSync:       getEnvBool("WINDOW_VSYNC", true),
			MSAA:        getEnvInt("GRAPHICS_MSAA", 4),
			MaxFPS:      getEnvInt("GRAPHICS_MAX_FPS", 60),
			OpenGLMajor: getEnvInt("GRAPHICS_OPENGL_MAJOR", 3),
			OpenGLMinor: getEnvInt("GRAPHICS_OPENGL_MINOR", 3),
		},
		World: WorldConfig{
			ChunkSize:   getEnvInt("WORLD_CHUNK_SIZE", 16),
			ChunkHeight: getEnvInt("WORLD_CHUNK_HEIGHT", 32),
			PlanetSize: [3]int{
				getEnvInt("WORLD_PLANET_SIZE_X", 3),
				getEnvInt("WORLD_PLANET_SIZE_Y", 3),
				getEnvInt("WORLD_PLANET_SIZE_Z", 3),
			},
			Seed: getEnvInt64("WORLD_SEED", 42),
		},
		Performance: PerformanceConfig{
			MaxChunkLoadDistance: getEnvFloat32("PERF_MAX_CHUNK_DISTANCE", 500.0),
			ChunkLoadWorkers:     getEnvInt("PERF_CHUNK_WORKERS", 4),
			EnableFrustumCulling: getEnvBool("PERF_FRUSTUM_CULLING", true),
			EnableMeshCaching:    getEnvBool("PERF_MESH_CACHING", true),
		},
	}
}

// getEnvString récupère une variable d'environnement string avec une valeur par défaut
func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt récupère une variable d'environnement int avec une valeur par défaut
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvInt64 récupère une variable d'environnement int64 avec une valeur par défaut
func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvFloat32 récupère une variable d'environnement float32 avec une valeur par défaut
func getEnvFloat32(key string, defaultValue float32) float32 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 32); err == nil {
			return float32(floatValue)
		}
	}
	return defaultValue
}

// getEnvBool récupère une variable d'environnement bool avec une valeur par défaut
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// Validate vérifie que la configuration est valide
func (c *Config) Validate() error {
	if c.Window.Width <= 0 || c.Window.Height <= 0 {
		return fmt.Errorf("invalid window dimensions: %dx%d", c.Window.Width, c.Window.Height)
	}

	if c.World.ChunkSize <= 0 || c.World.ChunkHeight <= 0 {
		return fmt.Errorf("invalid chunk dimensions: %dx%d", c.World.ChunkSize, c.World.ChunkHeight)
	}

	if c.Performance.ChunkLoadWorkers <= 0 {
		return fmt.Errorf("invalid number of chunk workers: %d", c.Performance.ChunkLoadWorkers)
	}

	return nil
}
