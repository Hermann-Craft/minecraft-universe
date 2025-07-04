package physics

import (
	"fmt"
	"sync"
	"time"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/hermann-craft/unicube/internal/core/geom"
)

// PhysicsState represents the explicit state of a physics object
type PhysicsState int

const (
	PhysicsStateUninitialized PhysicsState = iota
	PhysicsStateStatic
	PhysicsStateDynamic
	PhysicsStateKinematic
	PhysicsStateSleeping
	PhysicsStateError
)

// String returns the string representation of the physics state
func (s PhysicsState) String() string {
	switch s {
	case PhysicsStateUninitialized:
		return "Uninitialized"
	case PhysicsStateStatic:
		return "Static"
	case PhysicsStateDynamic:
		return "Dynamic"
	case PhysicsStateKinematic:
		return "Kinematic"
	case PhysicsStateSleeping:
		return "Sleeping"
	case PhysicsStateError:
		return "Error"
	default:
		return "Unknown"
	}
}

// PhysicsObject represents a physics object with explicit state management
type PhysicsObject struct {
	// Core state
	state PhysicsState
	id    string

	// Transform
	position mgl32.Vec3
	rotation mgl32.Quat
	scale    mgl32.Vec3

	// Physics properties
	velocity     mgl32.Vec3
	acceleration mgl32.Vec3
	mass         float32
	restitution  float32 // Bounciness
	friction     float32

	// Collision
	collider    Collider
	boundingBox geom.BoundingBox

	// Physics flags
	isGravityEnabled   bool
	isCollisionEnabled bool
	isSleeping         bool
	isGrounded         bool

	// Thread safety
	mu sync.RWMutex

	// Error tracking
	lastError  error
	lastUpdate time.Time
}

// NewPhysicsObject creates a new physics object
func NewPhysicsObject(id string) *PhysicsObject {
	return &PhysicsObject{
		state:              PhysicsStateUninitialized,
		id:                 id,
		position:           mgl32.Vec3{0, 0, 0},
		rotation:           mgl32.Quat{W: 1, V: mgl32.Vec3{0, 0, 0}},
		scale:              mgl32.Vec3{1, 1, 1},
		velocity:           mgl32.Vec3{0, 0, 0},
		acceleration:       mgl32.Vec3{0, 0, 0},
		mass:               1.0,
		restitution:        0.5,
		friction:           0.3,
		isGravityEnabled:   true,
		isCollisionEnabled: true,
		isSleeping:         false,
		isGrounded:         false,
		lastUpdate:         time.Now(),
	}
}

// GetID returns the physics object ID
func (po *PhysicsObject) GetID() string {
	return po.id
}

// GetState returns the current physics state
func (po *PhysicsObject) GetState() PhysicsState {
	po.mu.RLock()
	defer po.mu.RUnlock()
	return po.state
}

// SetState sets the physics state
func (po *PhysicsObject) SetState(state PhysicsState) {
	po.mu.Lock()
	defer po.mu.Unlock()
	po.state = state
	po.lastUpdate = time.Now()
}

// GetLastError returns the last error that occurred
func (po *PhysicsObject) GetLastError() error {
	po.mu.RLock()
	defer po.mu.RUnlock()
	return po.lastError
}

// SetLastError sets the last error
func (po *PhysicsObject) SetLastError(err error) {
	po.mu.Lock()
	defer po.mu.Unlock()
	po.lastError = err
	if err != nil {
		po.state = PhysicsStateError
	}
	po.lastUpdate = time.Now()
}

// GetPosition returns the current position
func (po *PhysicsObject) GetPosition() mgl32.Vec3 {
	po.mu.RLock()
	defer po.mu.RUnlock()
	return po.position
}

// SetPosition sets the position
func (po *PhysicsObject) SetPosition(position mgl32.Vec3) {
	po.mu.Lock()
	defer po.mu.Unlock()
	po.position = position
	// Mettre à jour la bounding box dès que la position change
	if po.collider != nil {
		po.boundingBox = po.collider.GetBoundingBox(po.position, po.rotation, po.scale)
	}
	po.lastUpdate = time.Now()
}

// GetRotation returns the current rotation
func (po *PhysicsObject) GetRotation() mgl32.Quat {
	po.mu.RLock()
	defer po.mu.RUnlock()
	return po.rotation
}

// SetRotation sets the rotation
func (po *PhysicsObject) SetRotation(rotation mgl32.Quat) {
	po.mu.Lock()
	defer po.mu.Unlock()
	po.rotation = rotation
	if po.collider != nil {
		po.boundingBox = po.collider.GetBoundingBox(po.position, po.rotation, po.scale)
	}
	po.lastUpdate = time.Now()
}

// GetScale returns the current scale
func (po *PhysicsObject) GetScale() mgl32.Vec3 {
	po.mu.RLock()
	defer po.mu.RUnlock()
	return po.scale
}

// GetVelocity returns the current velocity
func (po *PhysicsObject) GetVelocity() mgl32.Vec3 {
	po.mu.RLock()
	defer po.mu.RUnlock()
	return po.velocity
}

// SetVelocity sets the velocity
func (po *PhysicsObject) SetVelocity(velocity mgl32.Vec3) {
	po.mu.Lock()
	defer po.mu.Unlock()
	po.velocity = velocity
	po.lastUpdate = time.Now()
}

// GetAcceleration returns the current acceleration
func (po *PhysicsObject) GetAcceleration() mgl32.Vec3 {
	po.mu.RLock()
	defer po.mu.RUnlock()
	return po.acceleration
}

// SetAcceleration sets the acceleration
func (po *PhysicsObject) SetAcceleration(acceleration mgl32.Vec3) {
	po.mu.Lock()
	defer po.mu.Unlock()
	po.acceleration = acceleration
	po.lastUpdate = time.Now()
}

// GetMass returns the mass
func (po *PhysicsObject) GetMass() float32 {
	po.mu.RLock()
	defer po.mu.RUnlock()
	return po.mass
}

// SetMass sets the mass
func (po *PhysicsObject) SetMass(mass float32) error {
	if mass <= 0 {
		return fmt.Errorf("mass must be positive")
	}
	po.mu.Lock()
	defer po.mu.Unlock()
	po.mass = mass
	po.lastUpdate = time.Now()
	return nil
}

// GetCollider returns the collider
func (po *PhysicsObject) GetCollider() Collider {
	po.mu.RLock()
	defer po.mu.RUnlock()
	return po.collider
}

// SetCollider sets the collider
func (po *PhysicsObject) SetCollider(collider Collider) {
	po.mu.Lock()
	defer po.mu.Unlock()
	po.collider = collider
	po.boundingBox = collider.GetBoundingBox(po.position, po.rotation, po.scale)
	po.lastUpdate = time.Now()
}

// GetBoundingBox returns the current bounding box
func (po *PhysicsObject) GetBoundingBox() geom.BoundingBox {
	po.mu.RLock()
	defer po.mu.RUnlock()
	return po.boundingBox
}

// UpdateBoundingBox updates the bounding box based on current transform
func (po *PhysicsObject) UpdateBoundingBox() {
	po.mu.Lock()
	defer po.mu.Unlock()
	if po.collider != nil {
		po.boundingBox = po.collider.GetBoundingBox(po.position, po.rotation, po.scale)
	}
	po.lastUpdate = time.Now()
}

// IsGravityEnabled returns whether gravity is enabled
func (po *PhysicsObject) IsGravityEnabled() bool {
	po.mu.RLock()
	defer po.mu.RUnlock()
	return po.isGravityEnabled
}

// SetGravityEnabled sets whether gravity is enabled
func (po *PhysicsObject) SetGravityEnabled(enabled bool) {
	po.mu.Lock()
	defer po.mu.Unlock()
	po.isGravityEnabled = enabled
	po.lastUpdate = time.Now()
}

// IsCollisionEnabled returns whether collision detection is enabled
func (po *PhysicsObject) IsCollisionEnabled() bool {
	po.mu.RLock()
	defer po.mu.RUnlock()
	return po.isCollisionEnabled
}

// SetCollisionEnabled sets whether collision detection is enabled
func (po *PhysicsObject) SetCollisionEnabled(enabled bool) {
	po.mu.Lock()
	defer po.mu.Unlock()
	po.isCollisionEnabled = enabled
	po.lastUpdate = time.Now()
}

// IsSleeping returns whether the object is sleeping
func (po *PhysicsObject) IsSleeping() bool {
	po.mu.RLock()
	defer po.mu.RUnlock()
	return po.isSleeping
}

// SetSleeping sets whether the object is sleeping
func (po *PhysicsObject) SetSleeping(sleeping bool) {
	po.mu.Lock()
	defer po.mu.Unlock()
	po.isSleeping = sleeping
	if sleeping {
		po.state = PhysicsStateSleeping
		po.velocity = mgl32.Vec3{0, 0, 0}
		po.acceleration = mgl32.Vec3{0, 0, 0}
	} else if po.state == PhysicsStateSleeping {
		po.state = PhysicsStateDynamic // Or whatever state it was before
	}
	po.lastUpdate = time.Now()
}

// IsGrounded returns true if the object is on the ground
func (po *PhysicsObject) IsGrounded() bool {
	po.mu.RLock()
	defer po.mu.RUnlock()
	return po.isGrounded
}

// SetGrounded sets the grounded state of the object
func (po *PhysicsObject) SetGrounded(grounded bool) {
	po.mu.Lock()
	defer po.mu.Unlock()
	po.isGrounded = grounded
	po.lastUpdate = time.Now()
}

// GetLastUpdate returns the last time the object was updated
func (po *PhysicsObject) GetLastUpdate() time.Time {
	po.mu.RLock()
	defer po.mu.RUnlock()
	return po.lastUpdate
}

// IsStatic returns whether the object is static
func (po *PhysicsObject) IsStatic() bool {
	return po.GetState() == PhysicsStateStatic
}

// IsDynamic returns whether the object is dynamic
func (po *PhysicsObject) IsDynamic() bool {
	po.mu.RLock()
	defer po.mu.RUnlock()
	return po.state == PhysicsStateDynamic
}

// ApplyForce applies a force to the object
func (po *PhysicsObject) ApplyForce(force mgl32.Vec3) {
	po.mu.Lock()
	defer po.mu.Unlock()
	if po.mass > 0 {
		po.acceleration = po.acceleration.Add(force.Mul(1.0 / po.mass))
	}
	po.lastUpdate = time.Now()
}

// Update performs a physics integration step
func (po *PhysicsObject) Update(deltaTime float32) {
	po.mu.Lock()
	defer po.mu.Unlock()

	// Don't update static or sleeping objects
	if po.state == PhysicsStateStatic || po.state == PhysicsStateSleeping {
		return
	}

	// Apply acceleration to velocity
	po.velocity = po.velocity.Add(po.acceleration.Mul(deltaTime))

	// Apply velocity to position
	po.position = po.position.Add(po.velocity.Mul(deltaTime))

	// Reset acceleration for the next frame
	po.acceleration = mgl32.Vec3{0, 0, 0}

	// Update bounding box
	if po.collider != nil {
		po.boundingBox = po.collider.GetBoundingBox(po.position, po.rotation, po.scale)
	}

	po.lastUpdate = time.Now()
}

// ApplyImpulse adds an instantaneous change to velocity.
func (po *PhysicsObject) ApplyImpulse(impulse mgl32.Vec3) {
	po.mu.Lock()
	defer po.mu.Unlock()
	// dv = F * dt / m. For an impulse, we consider this as a direct velocity change.
	// We'll treat the impulse as a direct change in velocity for simplicity.
	po.velocity = po.velocity.Add(impulse)
	po.lastUpdate = time.Now()
}
