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

	// Resource Management
	blockRegistry *world.BlockRegistry
	textureAtlas  render.TextureAtlas
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

	// Récupérer la vélocité actuelle UNIQUEMENT pour sa composante verticale.
	currentVelY := game.cameraPhysics.GetVelocity().Y()

	// Calculer la direction de mouvement basée sur les touches
	moveDir := mgl32.Vec3{0, 0, 0}
	vectors := game.camera.GetVectors()

	if game.window.GetKey(glfw.KeyW) == glfw.Press {
		moveDir = moveDir.Add(vectors.Front)
	}
	if game.window.GetKey(glfw.KeyS) == glfw.Press {
		moveDir = moveDir.Sub(vectors.Front)
	}
	if game.window.GetKey(glfw.KeyA) == glfw.Press {
		moveDir = moveDir.Sub(vectors.Right)
	}
	if game.window.GetKey(glfw.KeyD) == glfw.Press {
		moveDir = moveDir.Add(vectors.Right)
	}

	// Appliquer la nouvelle vélocité horizontale tout en PRÉSERVANT la vélocité verticale
	// calculée par le moteur physique dans la frame précédente.
	if moveDir.LenSqr() > 0 {
		moveDir = moveDir.Normalize()
		newHorizontalVel := moveDir.Mul(currentMoveSpeed)
		finalVel := mgl32.Vec3{newHorizontalVel.X(), currentVelY, newHorizontalVel.Z()}
		game.cameraPhysics.SetVelocity(finalVel)
	}
	// Si aucune touche n'est pressée, la vélocité horizontale diminuera grâce à la friction
	// et la vélocité verticale sera entièrement gérée par la gravité.
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

func (game *Game) handlePlayerActions() {
	// --- JUMP LOGIC ---
	// On saute seulement si on est au sol et qu'on ne monte pas déjà
	if game.input.IsKeyPressed(glfw.KeySpace) && game.cameraPhysics.IsGrounded() {
		jumpImpulse := mgl32.Vec3{0, 6, 0} // Jump impulse in m/s
		game.cameraPhysics.ApplyImpulse(jumpImpulse)
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
