package main

import (
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"

	"github.com/hermann-craft/unicube/internal/core/camera"
	"github.com/hermann-craft/unicube/internal/core/geom"
	"github.com/hermann-craft/unicube/internal/core/physics"
	"github.com/hermann-craft/unicube/internal/core/world"
	"github.com/hermann-craft/unicube/internal/input"
	"github.com/hermann-craft/unicube/internal/render"
	"github.com/hermann-craft/unicube/internal/utils"
	goecs "github.com/oneforx/go-ecs"
)

// Game represents the main game with the new architecture
type Game struct {
	config         *utils.Config
	window         *glfw.Window
	input          *input.InputManager
	renderer       *render.Renderer
	camera         *camera.Camera
	movementCtrl   *camera.MovementController
	physicsManager *physics.PhysicsManager
	cameraPhysics  *physics.PhysicsObject
	planet         *world.Planet
	worldCollision *physics.WorldCollisionSystem
	running        bool
	cursorCaptured bool

	// Gravity
	planetaryGravityField *physics.PlanetaryGravityField
	gravitySystem         *physics.GravitySystem

	// Camera face management
	faceTransitionManager *camera.FaceTransitionManager

	// Resource Management
	blockRegistry *world.BlockRegistry
	textureAtlas  render.TextureAtlas
	textRenderer  *render.TextRenderer
}

// NewGame creates a new instance of the game
func NewGame(config *utils.Config) *Game {
	return &Game{
		config:         config,
		running:        false,
		cursorCaptured: true, // Le curseur est capturé par défaut au démarrage
	}
}

// Init initializes the game
func (game *Game) Init() error {
	// Initialize GLFW
	if err := glfw.Init(); err != nil {
		return err
	}

	// Configure GLFW
	glfw.WindowHint(glfw.ContextVersionMajor, game.config.Graphics.OpenGLMajor)
	glfw.WindowHint(glfw.ContextVersionMinor, game.config.Graphics.OpenGLMinor)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	if game.config.Window.Resizable {
		glfw.WindowHint(glfw.Resizable, glfw.True)
	} else {
		glfw.WindowHint(glfw.Resizable, glfw.False)
	}

	// Create the window
	var err error
	game.window, err = glfw.CreateWindow(
		game.config.Window.Width,
		game.config.Window.Height,
		game.config.Window.Title,
		nil, nil,
	)
	if err != nil {
		glfw.Terminate()
		return err
	}
	game.window.MakeContextCurrent()
	game.window.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)

	// Initialize the input manager
	game.input = input.NewInputManager(game.window)

	// Configure the callback for mouse movement via the InputManager
	game.input.OnMouseMove(game.handleMouseMovement)

	// Initialize the renderer
	game.renderer, err = render.NewRenderer(game.window)
	if err != nil {
		game.window.Destroy()
		glfw.Terminate()
		return err
	}

	// In Init(), after initializing the renderer:
	err = game.renderer.LoadShader("basic", "shaders/basic.vert", "shaders/default.frag")
	if err != nil {
		game.window.Destroy()
		glfw.Terminate()
		return fmt.Errorf("failed to load basic shader: %w", err)
	}

	// Update text renderer screen size
	w, h := game.window.GetSize()
	render.SetScreenDimensions(w, h)

	// Initialize text renderer for HUD
	game.textRenderer = &render.TextRenderer{}
	if err := game.textRenderer.Init(); err != nil {
		game.window.Destroy()
		glfw.Terminate()
		return fmt.Errorf("failed to initialize text renderer: %w", err)
	}

	// === RESOURCE LOADING ===
	// 1. Load block definitions from the resource pack
	game.blockRegistry = world.NewBlockRegistry()
	game.blockRegistry.RegisterBlockModel(world.BlockTypeBedrock, "minecraft:block/bedrock")
	// world.Air does not need a model

	// 1b. Load all models and textures from the resource packs
	resourcePacks := []string{"resourcepacks/Ashen_16x", "resources/vanilla"}
	textureFiles, err := game.blockRegistry.LoadResourcePacks(resourcePacks)
	if err != nil {
		return fmt.Errorf("failed to load resource packs: %w", err)
	}

	// 2. Create the texture atlas from the required textures
	game.textureAtlas, err = render.NewTextureAtlas(resourcePacks, textureFiles)
	if err != nil {
		return fmt.Errorf("failed to create texture atlas: %w", err)
	}
	// ========================

	// Initialize the main planet first (before camera)
	game.planet = world.NewPlanet(
		goecs.Identifier{Namespace: "core", Path: "earth-planet"},
		mgl32.Vec3{0, 0, 0},
		mgl32.Vec3{4, 1, 4},
		world.ChunkSize{Width: 16, Height: 32, Depth: 16},
		42,
	)
	if err := game.planet.InitializeChunks(game.blockRegistry, game.textureAtlas); err != nil {
		game.window.Destroy()
		glfw.Terminate()
		return err
	}

	// Initialize the gravity field for the planet
	game.planetaryGravityField = physics.NewPlanetaryGravityField(game.planet)
	game.planetaryGravityField.Initialize()

	// Initialize the dynamic gravity system
	game.gravitySystem = physics.NewGravitySystem(game.planet, game.planetaryGravityField)

	// Calculate safe spawn position after the planet is populated
	log.Println("Calculating safe spawn position...")
	spawnPos, err := game.planet.GetSafeSpawnPosition()
	if err != nil {
		log.Printf("Warning: Could not calculate safe spawn position: %v. Using default.", err)
		spawnPos = mgl32.Vec3{32, 35, 32} // Fallback position
	}
	log.Printf("Player will spawn at: (%.1f, %.1f, %.1f)", spawnPos.X(), spawnPos.Y(), spawnPos.Z())

	// Initialize the camera with the calculated spawn position
	game.camera = camera.NewCamera(spawnPos)
	if err := game.camera.Init(camera.WorldFaceTop); err != nil {
		game.window.Destroy()
		glfw.Terminate()
		return err
	}

	// Initialize the camera movement controller
	game.movementCtrl = camera.NewMovementController(game.camera)

	// Initialize face transition manager
	game.faceTransitionManager = camera.NewFaceTransitionManager()

	// Initialize physics and collision with the main planet
	game.physicsManager = physics.NewPhysicsManager()
	if err := game.physicsManager.Initialize(); err != nil {
		game.window.Destroy()
		glfw.Terminate()
		return err
	}
	if err := game.physicsManager.Start(); err != nil {
		game.window.Destroy()
		glfw.Terminate()
		return err
	}
	game.worldCollision = physics.NewWorldCollisionSystem(game.physicsManager, game.planet)

	// Connect the gravity system to the world collision system
	game.worldCollision.SetGravitySystem(game.gravitySystem)

	// NEW: Create a PhysicsObject for the player/camera with the safe spawn position
	playerPhysics := physics.NewPhysicsObject("player")
	playerPhysics.SetPosition(spawnPos)                                          // Use calculated spawn position
	playerPhysics.SetMass(80.0)                                                  // Typical mass of a human
	playerPhysics.SetCollider(physics.NewBoxCollider(mgl32.Vec3{0.6, 1.8, 0.6})) // Capsule approx.
	playerPhysics.SetState(physics.PhysicsStateDynamic)
	if err := game.physicsManager.AddObject(playerPhysics); err != nil {
		game.window.Destroy()
		glfw.Terminate()
		return err
	}
	// Store the reference for fast access
	game.cameraPhysics = playerPhysics

	// IMPORTANT: Initialize physics rotation to match camera's initial face
	game.synchronizePhysicsRotationWithCameraFace(game.camera.GetCurrentFace())

	game.running = true
	return nil
}

// Run lance la boucle principale du jeu
func (game *Game) Run() {
	log.Println("Game loop started")
	frameDuration := time.Second / time.Duration(game.config.Graphics.MaxFPS)
	previousTime := time.Now()

	for game.running && !game.window.ShouldClose() {
		currentTime := time.Now()
		deltaTime := float32(currentTime.Sub(previousTime).Seconds())
		previousTime = currentTime

		// Gérer l'état du curseur
		if game.cursorCaptured {
			if game.window.GetInputMode(glfw.CursorMode) != glfw.CursorDisabled {
				game.window.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
			}
		} else {
			if game.window.GetInputMode(glfw.CursorMode) != glfw.CursorNormal {
				game.window.SetInputMode(glfw.CursorMode, glfw.CursorNormal)
			}
		}

		// Mettre à jour les entrées (ce qui inclut PollEvents)
		game.input.Update()

		// Mettre à jour la logique du jeu
		game.Update(deltaTime)

		// Rendu
		game.Render()

		// Limiter le FPS
		elapsed := time.Since(currentTime)
		sleepTime := frameDuration - elapsed
		if sleepTime > 0 {
			time.Sleep(sleepTime)
		}
	}
	log.Println("Game loop ended")
}

// Update met à jour la logique du jeu
func (game *Game) Update(deltaTime float32) {
	// Mise à jour des vecteurs de la caméra au début pour refléter les mouvements de la souris
	game.camera.UpdateVectors()

	// Mettre à jour la physique et les collisions
	if game.worldCollision != nil {
		game.worldCollision.Update(deltaTime)
	}

	// Synchroniser la position de la caméra avec l'objet physique APRES la physique
	if game.cameraPhysics != nil {
		game.camera.SetPosition(game.cameraPhysics.GetPosition())

		// Update camera face based on gravity system
		game.updateCameraFace()

		// Synchronize camera vectors after face changes (ensure consistency with physics object)
		game.camera.UpdateVectors()
	}

	// Gérer les entrées utilisateur pour le mouvement et les actions
	game.updateCamera(deltaTime) // Mouvement
	game.handlePlayerActions()   // Saut, minage, etc.

	// Update the planet
	game.planet.Update(float64(deltaTime))
}

// updateCamera met à jour la caméra basée sur les entrées
func (game *Game) updateCamera(deltaTime float32) {
	// Gère le basculement de la capture du curseur
	if game.input.IsKeyJustPressed(glfw.KeyEscape) {
		game.cursorCaptured = !game.cursorCaptured
	}

	// Re-capture le curseur si on clique et qu'il n'est pas capturé
	if !game.cursorCaptured && game.input.IsMousePressed(glfw.MouseButton1) {
		game.cursorCaptured = true
	}

	// Ne pas traiter le mouvement de la caméra si le curseur n'est pas capturé
	if !game.cursorCaptured {
		return
	}

	const moveSpeed = 5.0 // Vitesse de base
	sprintMultiplier := 2.0
	currentMoveSpeed := float32(moveSpeed)

	if game.input.IsKeyPressed(glfw.KeyLeftShift) {
		currentMoveSpeed *= float32(sprintMultiplier)
	}

	// Calculer la direction de mouvement basée sur les touches et l'orientation de l'objet physique
	moveDir := mgl32.Vec3{0, 0, 0}

	// Get camera vectors for movement direction
	vectors := game.camera.GetVectors()

	// Calculate local movement direction first
	localMoveDir := mgl32.Vec3{0, 0, 0}
	var keysPressed []string

	if game.window.GetKey(glfw.KeyW) == glfw.Press {
		localMoveDir = localMoveDir.Add(vectors.Front)
		keysPressed = append(keysPressed, "W")
	}
	if game.window.GetKey(glfw.KeyS) == glfw.Press {
		localMoveDir = localMoveDir.Sub(vectors.Front)
		keysPressed = append(keysPressed, "S")
	}
	if game.window.GetKey(glfw.KeyA) == glfw.Press {
		localMoveDir = localMoveDir.Sub(vectors.Right)
		keysPressed = append(keysPressed, "A")
	}
	if game.window.GetKey(glfw.KeyD) == glfw.Press {
		localMoveDir = localMoveDir.Add(vectors.Right)
		keysPressed = append(keysPressed, "D")
	}

	// Debug movement for Front face specifically
	currentPos := game.cameraPhysics.GetPosition()
	currentFace := game.planetaryGravityField.DetectFaceForPoint(currentPos)
	if len(keysPressed) > 0 && currentFace.String() == "Front" {
		log.Printf("FRONT FACE Movement Debug: Keys=%v, Front=(%.2f,%.2f,%.2f), Right=(%.2f,%.2f,%.2f), localMoveDir=(%.2f,%.2f,%.2f)",
			keysPressed,
			vectors.Front.X(), vectors.Front.Y(), vectors.Front.Z(),
			vectors.Right.X(), vectors.Right.Y(), vectors.Right.Z(),
			localMoveDir.X(), localMoveDir.Y(), localMoveDir.Z())
	}

	// Project movement onto the face plane to eliminate unwanted vertical movement
	if localMoveDir.LenSqr() > 0 {
		// Get current face to determine gravity direction
		currentPos := game.cameraPhysics.GetPosition()
		gravityInfo := game.gravitySystem.CalculateGravityInfo(currentPos)

		// Debug: Log detailed movement calculation
		log.Printf("Movement Debug BEFORE projection: Face=%s, localMoveDir=(%.2f,%.2f,%.2f), gravityDir=(%.2f,%.2f,%.2f)",
			gravityInfo.Face, localMoveDir.X(), localMoveDir.Y(), localMoveDir.Z(),
			gravityInfo.Direction.X(), gravityInfo.Direction.Y(), gravityInfo.Direction.Z())

		// Project movement direction onto the face plane (perpendicular to gravity)
		gravityDir := gravityInfo.Direction
		dot := localMoveDir.Dot(gravityDir)
		projectedMovement := localMoveDir.Sub(gravityDir.Mul(dot))

		log.Printf("Movement Debug AFTER projection: dot=%.2f, projectedMovement=(%.2f,%.2f,%.2f)",
			dot, projectedMovement.X(), projectedMovement.Y(), projectedMovement.Z())

		if projectedMovement.LenSqr() > 0 {
			moveDir = projectedMovement
		}
	}

	// Appliquer la nouvelle vélocité horizontale tout en PRÉSERVANT la vélocité verticale
	// calculée par le moteur physique dans la frame précédente.
	if moveDir.LenSqr() > 0 {
		moveDir = moveDir.Normalize()
		newHorizontalVel := moveDir.Mul(currentMoveSpeed)

		// Get current "vertical" velocity along the gravity vector
		currentVel := game.cameraPhysics.GetVelocity()
		gravityDir := game.gravitySystem.CalculateGravityInfo(game.cameraPhysics.GetPosition()).Direction
		verticalVelComponent := gravityDir.Mul(currentVel.Dot(gravityDir))

		// The new velocity is the combination of the desired horizontal move and the existing vertical velocity.
		finalVel := newHorizontalVel.Add(verticalVelComponent)
		game.cameraPhysics.SetVelocity(finalVel)

		// Debug movement occasionally to avoid spam
		if int(finalVel.X()*100)%100 == 0 { // Log when velocity changes significantly
			log.Printf("Movement Debug: Keys pressed, moveDir=(%.2f,%.2f,%.2f), finalVel=(%.2f,%.2f,%.2f)",
				moveDir.X(), moveDir.Y(), moveDir.Z(), finalVel.X(), finalVel.Y(), finalVel.Z())
		}
	} else {
		// If no keys are pressed, apply friction to the "horizontal" plane
		// while preserving the "vertical" velocity.
		currentVel := game.cameraPhysics.GetVelocity()
		gravityDir := game.gravitySystem.CalculateGravityInfo(game.cameraPhysics.GetPosition()).Direction
		verticalVelComponent := gravityDir.Mul(currentVel.Dot(gravityDir))
		horizontalVelComponent := currentVel.Sub(verticalVelComponent)

		// Apply friction only to the horizontal part
		horizontalVelComponent = horizontalVelComponent.Mul(0.8)

		finalVel := horizontalVelComponent.Add(verticalVelComponent)
		game.cameraPhysics.SetVelocity(finalVel)
	}
	// La vélocité verticale sera entièrement gérée par la gravité.
}

// Render effectue le rendu
func (game *Game) Render() {
	game.renderer.BeginFrame()

	// Mettre à jour le frustum de la caméra
	projection := mgl32.Perspective(
		mgl32.DegToRad(45.0),
		float32(game.config.Window.Width)/float32(game.config.Window.Height),
		0.1,
		1000.0,
	)
	game.camera.UpdateFrustum(projection)

	// Rendu de la planète
	if game.planet != nil {
		currentTime := glfw.GetTime()
		game.planet.Render(currentTime, game.renderer, game.camera, projection)
	}

	// Render HUD text showing current world face and debug info
	if game.textRenderer != nil && game.cameraPhysics != nil {
		pos := game.cameraPhysics.GetPosition()
		face := game.planetaryGravityField.DetectFaceForPoint(pos)

		text := fmt.Sprintf("World Face: %s", face)
		game.textRenderer.RenderText(text, 10, 25, 1.0)

		// Camera face info
		cameraText := fmt.Sprintf("Camera Face: %s", game.camera.GetCurrentFace())
		game.textRenderer.RenderText(cameraText, 10, 50, 1.0)

		// Camera vectors debug
		vectors := game.camera.GetVectors()
		vectorText := fmt.Sprintf("Front: (%.2f,%.2f,%.2f)", vectors.Front.X(), vectors.Front.Y(), vectors.Front.Z())
		game.textRenderer.RenderText(vectorText, 10, 75, 1.0)
	}

	game.renderer.EndFrame()
}

// handleMouseMovement gère le mouvement de la souris pour la rotation de la caméra
func (game *Game) handleMouseMovement(event input.InputEvent) {
	// Ne pas traiter le mouvement de la souris si le curseur n'est pas capturé
	if !game.cursorCaptured {
		return
	}
	game.movementCtrl.ProcessMouseMovement(event.Delta.X(), event.Delta.Y(), true)
}

// Cleanup nettoie les ressources
func (game *Game) Cleanup() {
	log.Println("Cleaning up game resources...")
	game.running = false

	if game.renderer != nil {
		game.renderer.Cleanup()
	}

	if game.input != nil {
		game.input.Cleanup()
	}

	if game.window != nil {
		game.window.Destroy()
	}

	if game.physicsManager != nil {
		game.physicsManager.Stop()
	}

	glfw.Terminate()
}

// Helper functions
func float32Abs(f float32) float32 {
	if f < 0 {
		return -f
	}
	return f
}

func sign(f float32) float32 {
	if f > 0 {
		return 1
	} else if f < 0 {
		return -1
	}
	return 0
}

func describeDirection(vec mgl32.Vec3) string {
	// Find the dominant axis
	absX, absY, absZ := float32Abs(vec.X()), float32Abs(vec.Y()), float32Abs(vec.Z())

	if absX > absY && absX > absZ {
		if vec.X() > 0 {
			return "+X (Right)"
		} else {
			return "-X (Left)"
		}
	} else if absY > absX && absY > absZ {
		if vec.Y() > 0 {
			return "+Y (Up)"
		} else {
			return "-Y (Down)"
		}
	} else {
		if vec.Z() > 0 {
			return "+Z (Forward)"
		} else {
			return "-Z (Backward)"
		}
	}
}

func (game *Game) handlePlayerActions() {
	// --- JUMP LOGIC ---
	// On saute seulement si on est au sol et qu'on ne monte pas déjà
	if game.input.IsKeyJustPressed(glfw.KeySpace) {
		isGrounded := game.cameraPhysics.IsGrounded()

		log.Printf("Jump Debug: Space pressed, isGrounded=%t", isGrounded)

		if isGrounded {
			// Get current gravity direction to jump upwards
			currentPos := game.cameraPhysics.GetPosition()
			gravityInfo := game.gravitySystem.CalculateGravityInfo(currentPos)
			jumpStrength := float32(6.0) // Jump impulse in m/s
			jumpImpulse := gravityInfo.Direction.Mul(-jumpStrength)

			// Add impulse to current velocity instead of just setting it
			currentVel := game.cameraPhysics.GetVelocity()
			newVel := currentVel.Add(jumpImpulse)
			game.cameraPhysics.SetVelocity(newVel)

			log.Printf("Jump ACTION: Face=%s, isGrounded=%t. Applying impulse (%.2f,%.2f,%.2f). New vel: (%.2f,%.2f,%.2f)",
				gravityInfo.Face, isGrounded,
				jumpImpulse.X(), jumpImpulse.Y(), jumpImpulse.Z(),
				newVel.X(), newVel.Y(), newVel.Z())
		}
	}

	// --- MINING (LEFT CLICK) ---
	if game.input.IsMouseJustPressed(glfw.MouseButton1) {
		origin := game.camera.GetPosition()
		direction := game.camera.GetFront()
		chunk, bx, by, bz, _, hit := game.worldCollision.RaycastBlock(origin, direction, 8.0)
		if hit && chunk != nil {
			// Calculate world position manually
			blockPos := mgl32.Vec3{
				float32(int(chunk.Position.X()) + bx),
				float32(int(chunk.Position.Y()) + by),
				float32(int(chunk.Position.Z()) + bz),
			}
			if err := game.worldCollision.RemoveBlock(blockPos); err != nil {
				log.Printf("Error removing block: %v", err)
			}
		}
	}

	// --- PLACING (RIGHT CLICK, tuned) ---
	if game.input.IsMouseJustPressed(glfw.MouseButton2) {
		origin := game.camera.GetPosition()
		direction := game.camera.GetFront()
		chunk, bx, by, bz, hitPos, hit := game.worldCollision.RaycastBlock(origin, direction, 8.0)
		if hit && chunk != nil {
			// Compute normal of the hit face (approximate by checking which axis has the largest difference)
			hitBlockCenter := mgl32.Vec3{
				float32(int(chunk.Position.X())+bx) + 0.5,
				float32(int(chunk.Position.Y())+by) + 0.5,
				float32(int(chunk.Position.Z())+bz) + 0.5,
			}
			delta := hitPos.Sub(hitBlockCenter)
			absDelta := mgl32.Vec3{float32Abs(delta.X()), float32Abs(delta.Y()), float32Abs(delta.Z())}
			var normal mgl32.Vec3
			if absDelta.X() > absDelta.Y() && absDelta.X() > absDelta.Z() {
				normal = mgl32.Vec3{sign(delta.X()), 0, 0}
			} else if absDelta.Y() > absDelta.X() && absDelta.Y() > absDelta.Z() {
				normal = mgl32.Vec3{0, sign(delta.Y()), 0}
			} else {
				normal = mgl32.Vec3{0, 0, sign(delta.Z())}
			}
			// Place block at adjacent position
			placePos := mgl32.Vec3{
				float32(int(chunk.Position.X()) + bx),
				float32(int(chunk.Position.Y()) + by),
				float32(int(chunk.Position.Z()) + bz),
			}.Add(normal)
			// Prevent placing inside player
			playerBox := game.cameraPhysics.GetBoundingBox()
			blockBox := geom.NewBoundingBox(
				placePos,
				placePos.Add(mgl32.Vec3{1, 1, 1}),
			)
			if !playerBox.Intersects(blockBox) {
				_ = game.worldCollision.PlaceBlock(placePos, world.BlockTypeStone)
			}
		}
	}
}

// updateCameraFace synchronizes the camera's face with the gravity system using FaceTransitionManager
func (game *Game) updateCameraFace() {
	if game.cameraPhysics == nil || game.gravitySystem == nil || game.faceTransitionManager == nil {
		return
	}

	// Get the current face from the gravity system
	currentPos := game.cameraPhysics.GetPosition()
	gravityInfo := game.gravitySystem.CalculateGravityInfo(currentPos)

	// Convert world.WorldFace to camera.WorldFace
	var cameraFace camera.WorldFace
	switch gravityInfo.Face {
	case world.WorldFaceTop:
		cameraFace = camera.WorldFaceTop
	case world.WorldFaceBottom:
		cameraFace = camera.WorldFaceBottom
	case world.WorldFaceLeft:
		cameraFace = camera.WorldFaceLeft
	case world.WorldFaceRight:
		cameraFace = camera.WorldFaceRight
	case world.WorldFaceFront:
		cameraFace = camera.WorldFaceFront
	case world.WorldFaceBack:
		cameraFace = camera.WorldFaceBack
	default:
		return // Don't change face for WorldFaceNone
	}

	// Apply face change to camera using FaceTransitionManager if needed
	if game.camera.GetCurrentFace() != cameraFace && !game.camera.IsTransitioning() {
		log.Printf("Camera face changing from %s to %s", game.camera.GetCurrentFace(), cameraFace)

		// IMPORTANT: Synchronize physics object rotation FIRST
		game.synchronizePhysicsRotationWithCameraFace(cameraFace)

		if err := game.faceTransitionManager.StartTransition(game.camera, cameraFace); err != nil {
			log.Printf("Failed to start face transition: %v", err)
			// Fallback to direct setting
			game.camera.SetFace(cameraFace)
		}
	}
}

// synchronizePhysicsRotationWithCameraFace applies the same face transformation to the physics object
func (game *Game) synchronizePhysicsRotationWithCameraFace(cameraFace camera.WorldFace) {
	if game.cameraPhysics == nil {
		return
	}

	// Get the face transformation matrix (same logic as camera.getFaceTransform)
	var faceTransform mgl32.Mat4
	switch cameraFace {
	case camera.WorldFaceTop:
		// No transformation needed - canonical orientation
		faceTransform = mgl32.Ident4()
	case camera.WorldFaceBottom:
		// Flip upside down (180° around X axis)
		faceTransform = mgl32.HomogRotate3DX(mgl32.DegToRad(180))
	case camera.WorldFaceLeft:
		// Rotate 90° around Z axis (roll left)
		faceTransform = mgl32.HomogRotate3DZ(mgl32.DegToRad(90))
	case camera.WorldFaceRight:
		// Rotate -90° around Z axis (roll right)
		faceTransform = mgl32.HomogRotate3DZ(mgl32.DegToRad(-90))
	case camera.WorldFaceFront:
		faceTransform = mgl32.HomogRotate3DX(mgl32.DegToRad(90)).Mul4(mgl32.HomogRotate3DY(mgl32.DegToRad(90)))
	case camera.WorldFaceBack:
		faceTransform = mgl32.HomogRotate3DX(mgl32.DegToRad(-90)).Mul4(mgl32.HomogRotate3DY(mgl32.DegToRad(-90)))
	default:
		faceTransform = mgl32.Ident4()
	}

	// Convert the transformation matrix to a quaternion
	physicsRotation := mgl32.Mat4ToQuat(faceTransform)

	// Apply the rotation to the physics object
	game.cameraPhysics.SetRotation(physicsRotation)

	log.Printf("Physics rotation synchronized for face %s: quat=(%.2f,%.2f,%.2f,%.2f)",
		cameraFace, physicsRotation.W, physicsRotation.V.X(), physicsRotation.V.Y(), physicsRotation.V.Z())
}

func init() {
	// Tous les appels OpenGL doivent se faire dans le thread principal
	runtime.LockOSThread()
}

func main() {
	// Charger la configuration
	config := utils.LoadConfig()

	// Valider la configuration
	if err := config.Validate(); err != nil {
		log.Fatalf("Configuration invalide: %v", err)
	}

	// Créer et initialiser le jeu
	game := NewGame(config)
	if err := game.Init(); err != nil {
		log.Fatalf("Erreur lors de l'initialisation: %v", err)
	}
	defer game.Cleanup()

	// Lancer le jeu
	game.Run()
}
