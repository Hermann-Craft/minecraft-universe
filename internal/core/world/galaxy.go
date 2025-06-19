package world

import (
	"fmt"
	"sync"

	"github.com/go-gl/mathgl/mgl32"
	goecs "github.com/oneforx/go-ecs"
)

// Galaxy représente une galaxie contenant plusieurs planètes
// (thread-safe, scalable, future-proof)
type Galaxy struct {
	ID       goecs.Identifier
	Position mgl32.Vec3
	Rotation mgl32.Quat

	planets map[string]*Planet
	mu      sync.RWMutex
}

// NewGalaxy crée une nouvelle galaxie
func NewGalaxy(id goecs.Identifier, position mgl32.Vec3, rotation mgl32.Quat) *Galaxy {
	return &Galaxy{
		ID:       id,
		Position: position,
		Rotation: rotation,
		planets:  make(map[string]*Planet),
	}
}

// AddPlanet ajoute une planète à la galaxie
func (g *Galaxy) AddPlanet(planet *Planet) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if planet == nil {
		return fmt.Errorf("planet is nil")
	}
	if _, exists := g.planets[planet.ID.String()]; exists {
		return fmt.Errorf("planet already exists in galaxy")
	}
	g.planets[planet.ID.String()] = planet
	return nil
}

// GetPlanet retourne une planète par son ID
func (g *Galaxy) GetPlanet(id goecs.Identifier) (*Planet, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	p, ok := g.planets[id.String()]
	return p, ok
}

// GetPlanets retourne la liste des planètes
func (g *Galaxy) GetPlanets() []*Planet {
	g.mu.RLock()
	defer g.mu.RUnlock()
	planets := make([]*Planet, 0, len(g.planets))
	for _, p := range g.planets {
		planets = append(planets, p)
	}
	return planets
}
