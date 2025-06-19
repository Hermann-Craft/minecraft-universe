package physics

import (
	"fmt"
	"sync"
	"time"

	"github.com/go-gl/mathgl/mgl32"
)

// PhysicsManagerState represents the state of the physics manager
type PhysicsManagerState int

const (
	PhysicsManagerStateUninitialized PhysicsManagerState = iota
	PhysicsManagerStateInitialized
	PhysicsManagerStateRunning
	PhysicsManagerStatePaused
	PhysicsManagerStateError
)

// String returns the string representation of the physics manager state
func (s PhysicsManagerState) String() string {
	switch s {
	case PhysicsManagerStateUninitialized:
		return "Uninitialized"
	case PhysicsManagerStateInitialized:
		return "Initialized"
	case PhysicsManagerStateRunning:
		return "Running"
	case PhysicsManagerStatePaused:
		return "Paused"
	case PhysicsManagerStateError:
		return "Error"
	default:
		return "Unknown"
	}
}

// PhysicsManager coordinates all physics systems
type PhysicsManager struct {
	// Core state
	state PhysicsManagerState
	mu    sync.RWMutex

	// Physics systems
	simulation        *PhysicsSimulation
	collisionDetector *CollisionDetector
	raycastSystem     *RaycastSystem

	// Object management
	objects   map[string]*PhysicsObject
	objectsMu sync.RWMutex

	// Statistics
	stats struct {
		objectCount    int
		activeObjects  int
		collisionCount int
		lastUpdateTime time.Time
		updateCount    int64
	}

	// Error tracking
	lastError error
}

// NewPhysicsManager creates a new physics manager
func NewPhysicsManager() *PhysicsManager {
	return &PhysicsManager{
		state:             PhysicsManagerStateUninitialized,
		objects:           make(map[string]*PhysicsObject),
		simulation:        NewPhysicsSimulation(),
		collisionDetector: NewCollisionDetector(10.0), // 10 unit grid
		raycastSystem:     NewRaycastSystem(),
	}
}

// Initialize initializes the physics manager
func (pm *PhysicsManager) Initialize() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.state != PhysicsManagerStateUninitialized {
		return fmt.Errorf("physics manager already initialized")
	}

	// Initialize subsystems
	pm.simulation.Start()

	pm.state = PhysicsManagerStateInitialized
	pm.stats.lastUpdateTime = time.Now()

	return nil
}

// GetState returns the current state of the physics manager
func (pm *PhysicsManager) GetState() PhysicsManagerState {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.state
}

// Start starts the physics simulation
func (pm *PhysicsManager) Start() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.state != PhysicsManagerStateInitialized && pm.state != PhysicsManagerStatePaused {
		return fmt.Errorf("physics manager must be initialized or paused to start")
	}

	pm.state = PhysicsManagerStateRunning
	return nil
}

// Pause pauses the physics simulation
func (pm *PhysicsManager) Pause() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.state != PhysicsManagerStateRunning {
		return fmt.Errorf("physics manager must be running to pause")
	}

	pm.state = PhysicsManagerStatePaused
	return nil
}

// Stop stops the physics simulation
func (pm *PhysicsManager) Stop() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.simulation.Stop()
	pm.state = PhysicsManagerStateInitialized
	return nil
}

// IsRunning returns whether the physics manager is running
func (pm *PhysicsManager) IsRunning() bool {
	return pm.GetState() == PhysicsManagerStateRunning
}

// AddObject adds a physics object to the manager
func (pm *PhysicsManager) AddObject(obj *PhysicsObject) error {
	if obj == nil {
		return fmt.Errorf("physics object cannot be nil")
	}

	pm.objectsMu.Lock()
	defer pm.objectsMu.Unlock()

	objID := obj.GetID()
	if _, exists := pm.objects[objID]; exists {
		return fmt.Errorf("physics object with ID %s already exists", objID)
	}

	pm.objects[objID] = obj
	pm.collisionDetector.AddObject(obj)

	// Update statistics
	pm.stats.objectCount++
	if obj.IsDynamic() {
		pm.stats.activeObjects++
	}

	return nil
}

// RemoveObject removes a physics object from the manager
func (pm *PhysicsManager) RemoveObject(objID string) error {
	pm.objectsMu.Lock()
	defer pm.objectsMu.Unlock()

	obj, exists := pm.objects[objID]
	if !exists {
		return fmt.Errorf("physics object with ID %s not found", objID)
	}

	pm.collisionDetector.RemoveObject(obj)
	delete(pm.objects, objID)

	// Update statistics
	pm.stats.objectCount--
	if obj.IsDynamic() {
		pm.stats.activeObjects--
	}

	return nil
}

// GetObject returns a physics object by ID
func (pm *PhysicsManager) GetObject(objID string) (*PhysicsObject, error) {
	pm.objectsMu.RLock()
	defer pm.objectsMu.RUnlock()

	obj, exists := pm.objects[objID]
	if !exists {
		return nil, fmt.Errorf("physics object with ID %s not found", objID)
	}

	return obj, nil
}

// GetAllObjects returns all physics objects
func (pm *PhysicsManager) GetAllObjects() []*PhysicsObject {
	pm.objectsMu.RLock()
	defer pm.objectsMu.RUnlock()

	objects := make([]*PhysicsObject, 0, len(pm.objects))
	for _, obj := range pm.objects {
		objects = append(objects, obj)
	}

	return objects
}

// GetDynamicObjects returns all dynamic physics objects
func (pm *PhysicsManager) GetDynamicObjects() []*PhysicsObject {
	pm.objectsMu.RLock()
	defer pm.objectsMu.RUnlock()

	var objects []*PhysicsObject
	for _, obj := range pm.objects {
		if obj.IsDynamic() {
			objects = append(objects, obj)
		}
	}

	return objects
}

// Update updates the physics simulation
func (pm *PhysicsManager) Update(deltaTime float32) error {
	if !pm.IsRunning() {
		return nil
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Get all objects
	objects := pm.GetAllObjects()

	// Update physics simulation
	pm.simulation.UpdateSimulation(objects, deltaTime)

	// Detect and resolve collisions
	collisions := pm.collisionDetector.DetectCollisions()
	pm.stats.collisionCount = len(collisions)

	for _, collision := range collisions {
		if err := pm.simulation.ResolveCollision(collision); err != nil {
			pm.lastError = err
			return err
		}
	}

	// Update statistics
	pm.stats.updateCount++
	pm.stats.lastUpdateTime = time.Now()

	return nil
}

// Raycast performs a raycast using the raycast system
func (pm *PhysicsManager) Raycast(origin, direction mgl32.Vec3) RaycastResult {
	objects := pm.GetAllObjects()
	return pm.raycastSystem.Raycast(origin, direction, objects)
}

// RaycastAll performs a raycast and returns all hits
func (pm *PhysicsManager) RaycastAll(origin, direction mgl32.Vec3) []RaycastHit {
	objects := pm.GetAllObjects()
	return pm.raycastSystem.RaycastAll(origin, direction, objects)
}

// SphereCast performs a sphere cast
func (pm *PhysicsManager) SphereCast(origin, direction mgl32.Vec3, radius float32) RaycastResult {
	objects := pm.GetAllObjects()
	return pm.raycastSystem.SphereCast(origin, direction, radius, objects)
}

// LineOfSight checks if there's a clear line of sight between two points
func (pm *PhysicsManager) LineOfSight(start, end mgl32.Vec3) bool {
	objects := pm.GetAllObjects()
	return pm.raycastSystem.LineOfSight(start, end, objects)
}

// GetClosestObject finds the closest physics object to a point
func (pm *PhysicsManager) GetClosestObject(point mgl32.Vec3) (*PhysicsObject, float32) {
	objects := pm.GetAllObjects()
	return pm.raycastSystem.GetClosestObject(point, objects)
}

// ApplyForce applies a force to a physics object
func (pm *PhysicsManager) ApplyForce(objID string, force mgl32.Vec3) error {
	obj, err := pm.GetObject(objID)
	if err != nil {
		return err
	}

	return pm.simulation.ApplyForce(obj, force)
}

// ApplyImpulse applies an impulse to a physics object
func (pm *PhysicsManager) ApplyImpulse(objID string, impulse mgl32.Vec3) error {
	obj, err := pm.GetObject(objID)
	if err != nil {
		return err
	}

	return pm.simulation.ApplyImpulse(obj, impulse)
}

// SetGravity sets the gravity for the physics simulation
func (pm *PhysicsManager) SetGravity(gravity mgl32.Vec3) {
	pm.simulation.SetGravity(gravity)
}

// GetGravity returns the current gravity
func (pm *PhysicsManager) GetGravity() mgl32.Vec3 {
	return pm.simulation.GetGravity()
}

// GetStatistics returns physics statistics
func (pm *PhysicsManager) GetStatistics() map[string]interface{} {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	return map[string]interface{}{
		"state":          pm.state.String(),
		"objectCount":    pm.stats.objectCount,
		"activeObjects":  pm.stats.activeObjects,
		"collisionCount": pm.stats.collisionCount,
		"updateCount":    pm.stats.updateCount,
		"lastUpdateTime": pm.stats.lastUpdateTime,
		"isRunning":      pm.IsRunning(),
	}
}

// GetLastError returns the last error that occurred
func (pm *PhysicsManager) GetLastError() error {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.lastError
}

// ClearError clears the last error
func (pm *PhysicsManager) ClearError() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.lastError = nil
}

// Cleanup cleans up the physics manager
func (pm *PhysicsManager) Cleanup() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Stop simulation
	pm.simulation.Stop()

	// Clear objects
	pm.objectsMu.Lock()
	pm.objects = make(map[string]*PhysicsObject)
	pm.objectsMu.Unlock()

	// Reset statistics
	pm.stats.objectCount = 0
	pm.stats.activeObjects = 0
	pm.stats.collisionCount = 0
	pm.stats.updateCount = 0

	pm.state = PhysicsManagerStateUninitialized
	pm.lastError = nil

	return nil
}
