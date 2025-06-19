package utils

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	config := LoadConfig()

	// Vérifier les valeurs par défaut
	if config.Window.Width != 800 {
		t.Errorf("Expected window width 800, got %d", config.Window.Width)
	}

	if config.Window.Height != 600 {
		t.Errorf("Expected window height 600, got %d", config.Window.Height)
	}

	if config.World.ChunkSize != 16 {
		t.Errorf("Expected chunk size 16, got %d", config.World.ChunkSize)
	}

	if config.Performance.ChunkLoadWorkers != 4 {
		t.Errorf("Expected chunk workers 4, got %d", config.Performance.ChunkLoadWorkers)
	}
}

func TestLoadConfigFromEnv(t *testing.T) {
	// Définir des variables d'environnement
	os.Setenv("WINDOW_WIDTH", "1024")
	os.Setenv("WINDOW_HEIGHT", "768")
	os.Setenv("WORLD_CHUNK_SIZE", "32")
	os.Setenv("PERF_CHUNK_WORKERS", "8")

	config := LoadConfig()

	// Vérifier que les valeurs d'environnement sont prises en compte
	if config.Window.Width != 1024 {
		t.Errorf("Expected window width 1024, got %d", config.Window.Width)
	}

	if config.Window.Height != 768 {
		t.Errorf("Expected window height 768, got %d", config.Window.Height)
	}

	if config.World.ChunkSize != 32 {
		t.Errorf("Expected chunk size 32, got %d", config.World.ChunkSize)
	}

	if config.Performance.ChunkLoadWorkers != 8 {
		t.Errorf("Expected chunk workers 8, got %d", config.Performance.ChunkLoadWorkers)
	}

	// Nettoyer les variables d'environnement
	os.Unsetenv("WINDOW_WIDTH")
	os.Unsetenv("WINDOW_HEIGHT")
	os.Unsetenv("WORLD_CHUNK_SIZE")
	os.Unsetenv("PERF_CHUNK_WORKERS")
}

func TestConfigValidation(t *testing.T) {
	config := LoadConfig()

	// La configuration par défaut devrait être valide
	if err := config.Validate(); err != nil {
		t.Errorf("Default config should be valid: %v", err)
	}

	// Tester une configuration invalide
	invalidConfig := &Config{
		Window: WindowConfig{
			Width:  -1, // Invalide
			Height: 600,
		},
	}

	if err := invalidConfig.Validate(); err == nil {
		t.Error("Invalid config should return error")
	}
}
