package physics

import (
	"fmt"
	"sync"
	"time"

	"github.com/go-gl/mathgl/mgl32"
)

// PhysicsSimulation handles physics simulation for all objects
type PhysicsSimulation struct {
	// Physics constants
	gravity     mgl32.Vec3
	timeStep    float32
	maxVelocity float32

	// Simulation state
	isRunning bool
	lastTime  time.Time

	// Thread safety
	mu sync.RWMutex
}

// NewPhysicsSimulation creates a new physics simulation
func NewPhysicsSimulation() *PhysicsSimulation {
	return &PhysicsSimulation{
		gravity:     mgl32.Vec3{0, -9.81, 0}, // Earth gravity
		timeStep:    1.0 / 60.0,              // 60 FPS
		maxVelocity: 100.0,                   // Maximum velocity
		isRunning:   false,
		lastTime:    time.Now(),
	}
}

// SetGravity sets the gravity vector
func (ps *PhysicsSimulation) SetGravity(gravity mgl32.Vec3) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.gravity = gravity
}

// GetGravity returns the current gravity vector
func (ps *PhysicsSimulation) GetGravity() mgl32.Vec3 {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.gravity
}

// SetTimeStep sets the physics time step
func (ps *PhysicsSimulation) SetTimeStep(timeStep float32) error {
	if timeStep <= 0 {
		return fmt.Errorf("time step must be positive")
	}
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.timeStep = timeStep
	return nil
}

// GetTimeStep returns the current time step
func (ps *PhysicsSimulation) GetTimeStep() float32 {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.timeStep
}

// SetMaxVelocity sets the maximum velocity
func (ps *PhysicsSimulation) SetMaxVelocity(maxVelocity float32) error {
	if maxVelocity <= 0 {
		return fmt.Errorf("max velocity must be positive")
	}
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.maxVelocity = maxVelocity
	return nil
}

// GetMaxVelocity returns the maximum velocity
func (ps *PhysicsSimulation) GetMaxVelocity() float32 {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.maxVelocity
}

// Start starts the physics simulation
func (ps *PhysicsSimulation) Start() {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.isRunning = true
	ps.lastTime = time.Now()
}

// Stop stops the physics simulation
func (ps *PhysicsSimulation) Stop() {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.isRunning = false
}

// IsRunning returns whether the simulation is running
func (ps *PhysicsSimulation) IsRunning() bool {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.isRunning
}

// UpdateSimulation updates the physics simulation for all objects
func (ps *PhysicsSimulation) UpdateSimulation(objects []*PhysicsObject, deltaTime float32) {
	if !ps.IsRunning() {
		return
	}

	// Clamp delta time to prevent large jumps
	if deltaTime > ps.GetTimeStep()*2 {
		deltaTime = ps.GetTimeStep() * 2
	}

	// Update each dynamic object
	for _, obj := range objects {
		if obj.IsDynamic() && !obj.IsSleeping() {
			ps.UpdateObject(obj, deltaTime)
		}
	}
}

// UpdateObject updates a single physics object
func (ps *PhysicsSimulation) UpdateObject(obj *PhysicsObject, deltaTime float32) {
	if obj == nil {
		return
	}

	// Get current state
	position := obj.GetPosition()
	velocity := obj.GetVelocity()
	acceleration := obj.GetAcceleration()

	// Apply gravity if enabled
	if obj.IsGravityEnabled() {
		gravity := ps.GetGravity()
		acceleration = acceleration.Add(gravity)
	}

	// Update velocity using Euler integration
	newVelocity := velocity.Add(acceleration.Mul(deltaTime))

	// Apply velocity limits
	maxVel := ps.GetMaxVelocity()
	if newVelocity.Len() > maxVel {
		newVelocity = newVelocity.Normalize().Mul(maxVel)
	}

	// Update position using velocity
	newPosition := position.Add(newVelocity.Mul(deltaTime))

	// Apply friction
	friction := obj.friction
	if friction > 0 {
		// Simple friction model
		frictionForce := newVelocity.Mul(-friction * deltaTime)
		newVelocity = newVelocity.Add(frictionForce)

		// Stop very small velocities
		if newVelocity.Len() < 0.01 {
			newVelocity = mgl32.Vec3{0, 0, 0}
		}
	}

	// Update object state
	obj.SetVelocity(newVelocity)
	obj.SetPosition(newPosition)
	obj.SetAcceleration(acceleration)

	// Update bounding box
	obj.UpdateBoundingBox()

	// Check for sleep conditions
	ps.CheckSleepConditions(obj)
}

// CheckSleepConditions checks if an object should go to sleep
func (ps *PhysicsSimulation) CheckSleepConditions(obj *PhysicsObject) {
	if obj.IsStatic() {
		return
	}

	velocity := obj.GetVelocity()
	acceleration := obj.GetAcceleration()

	// Object should sleep if velocity and acceleration are very small
	velocityThreshold := float32(0.01)
	accelerationThreshold := float32(0.01)

	if velocity.Len() < velocityThreshold && acceleration.Len() < accelerationThreshold {
		obj.SetSleeping(true)
	} else {
		obj.SetSleeping(false)
	}
}

// ApplyForce applies a force to a physics object
func (ps *PhysicsSimulation) ApplyForce(obj *PhysicsObject, force mgl32.Vec3) error {
	if obj == nil {
		return fmt.Errorf("physics object cannot be nil")
	}

	if obj.IsStatic() {
		return fmt.Errorf("cannot apply force to static object")
	}

	mass := obj.GetMass()
	if mass <= 0 {
		return fmt.Errorf("object mass must be positive")
	}

	// F = ma, so a = F/m
	acceleration := force.Mul(1.0 / mass)
	currentAccel := obj.GetAcceleration()
	obj.SetAcceleration(currentAccel.Add(acceleration))

	return nil
}

// ApplyImpulse applies an impulse to a physics object
func (ps *PhysicsSimulation) ApplyImpulse(obj *PhysicsObject, impulse mgl32.Vec3) error {
	if obj == nil {
		return fmt.Errorf("physics object cannot be nil")
	}

	if obj.IsStatic() {
		return fmt.Errorf("cannot apply impulse to static object")
	}

	mass := obj.GetMass()
	if mass <= 0 {
		return fmt.Errorf("object mass must be positive")
	}

	// Impulse = mv, so v = impulse/m
	velocityChange := impulse.Mul(1.0 / mass)
	currentVel := obj.GetVelocity()
	obj.SetVelocity(currentVel.Add(velocityChange))

	return nil
}

// ResolveCollision resolves a collision between two objects
func (ps *PhysicsSimulation) ResolveCollision(collision CollisionInfo) error {
	obj1 := collision.Object1
	obj2 := collision.Object2

	if obj1 == nil || obj2 == nil {
		return fmt.Errorf("collision objects cannot be nil")
	}

	// Don't resolve collisions between static objects
	if obj1.IsStatic() && obj2.IsStatic() {
		return nil
	}

	normal := collision.Normal
	penetration := collision.Penetration

	// Separate objects
	if !obj1.IsStatic() {
		separation := normal.Mul(penetration * 0.5)
		newPos := obj1.GetPosition().Sub(separation)
		obj1.SetPosition(newPos)
	}

	if !obj2.IsStatic() {
		separation := normal.Mul(penetration * 0.5)
		newPos := obj2.GetPosition().Add(separation)
		obj2.SetPosition(newPos)
	}

	// Resolve velocities
	if !obj1.IsStatic() && !obj2.IsStatic() {
		// Elastic collision
		vel1 := obj1.GetVelocity()
		vel2 := obj2.GetVelocity()
		mass1 := obj1.GetMass()
		mass2 := obj2.GetMass()

		// Calculate relative velocity
		relativeVel := vel1.Sub(vel2)
		velocityAlongNormal := relativeVel.Dot(normal)

		// Don't resolve if objects are moving apart
		if velocityAlongNormal > 0 {
			return nil
		}

		// Calculate restitution
		restitution := (obj1.restitution + obj2.restitution) * 0.5

		// Calculate impulse
		j := -(1 + restitution) * velocityAlongNormal
		j /= 1/mass1 + 1/mass2

		impulse := normal.Mul(j)

		// Apply impulse
		newVel1 := vel1.Add(impulse.Mul(1.0 / mass1))
		newVel2 := vel2.Sub(impulse.Mul(1.0 / mass2))

		obj1.SetVelocity(newVel1)
		obj2.SetVelocity(newVel2)
	} else if !obj1.IsStatic() {
		// obj1 is dynamic, obj2 is static
		vel1 := obj1.GetVelocity()
		velocityAlongNormal := vel1.Dot(normal)

		if velocityAlongNormal < 0 {
			// Reflect velocity
			reflection := normal.Mul(2 * velocityAlongNormal)
			newVel := vel1.Sub(reflection)
			obj1.SetVelocity(newVel)
		}
	} else if !obj2.IsStatic() {
		// obj2 is dynamic, obj1 is static
		vel2 := obj2.GetVelocity()
		velocityAlongNormal := vel2.Dot(normal)

		if velocityAlongNormal < 0 {
			// Reflect velocity
			reflection := normal.Mul(2 * velocityAlongNormal)
			newVel := vel2.Sub(reflection)
			obj2.SetVelocity(newVel)
		}
	}

	return nil
}
