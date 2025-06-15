package main

import (
	"fmt"
	"log"
	"sort"
	"sync"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
	goecs "github.com/oneforx/go-ecs"
)

type ChunkPriority struct {
	chunk    *Chunk
	distance float32
}

type Planet struct {
	Identifier goecs.Identifier
	Position   mgl32.Vec3
	Rotation   mgl32.Quat
	Chunks     [][][]*Chunk
	isInit     bool

	// Système de priorité pour le chargement des chunks
	chunkLoadQueue   chan ChunkPriority
	chunkLoadWorkers int
	chunkLoadMutex   sync.Mutex
	maxLoadDistance  float32
	chunksToLoad     []ChunkPriority
	stopWorkers      chan struct{}   // Canal pour arrêter les workers
	loadedChunks     map[*Chunk]bool // Map pour suivre les chunks déjà chargés
	window           *glfw.Window    // Référence à la fenêtre pour le contexte OpenGL
}

func (p *Planet) Init(identifier goecs.Identifier, position mgl32.Vec3, rotation mgl32.Quat) error {
	if p.isInit {
		return ErrPlanetAlreadyExist
	}

	log.Println("Initializing planet...")

	p.Identifier = identifier
	p.Position = position
	p.Rotation = rotation
	p.isInit = true
	p.loadedChunks = make(map[*Chunk]bool)

	// Initialiser le système de chargement des chunks
	p.chunkLoadQueue = make(chan ChunkPriority, 100)
	p.stopWorkers = make(chan struct{})
	p.chunkLoadWorkers = 4    // Nombre de workers pour le chargement des chunks
	p.maxLoadDistance = 500.0 // Distance maximale pour charger les chunks

	// Initialiser l'atlas de textures
	textureAtlas := GetTextureAtlas()
	if textureAtlas == nil {
		log.Println("Failed to get texture atlas")
		return fmt.Errorf("failed to get texture atlas")
	}

	log.Println("Initializing chunks grid...")

	// Initialiser la grille de chunks (3x3x3 pour commencer)
	p.Chunks = make([][][]*Chunk, 3)
	for x := range p.Chunks {
		p.Chunks[x] = make([][]*Chunk, 3)
		for y := range p.Chunks[x] {
			p.Chunks[x][y] = make([]*Chunk, 3)
			for z := range p.Chunks[x][y] {
				// Créer un nouveau chunk à la position (x,y,z)
				globalPos := mgl32.Vec3{
					float32(x-1) * 16, // -16, 0, 16
					float32(y-1) * 16,
					float32(z-1) * 16,
				}

				log.Printf("Creating chunk at position (%d, %d, %d)", x, y, z)

				// Créer le chunk avec la position globale
				chunk, err := NewChunk(
					globalPos,
					int64(x*1000+y*100+z), // Seed unique pour chaque chunk
				)
				if err != nil {
					return err
				}

				// Initialiser le chunk
				chunk.TextureAtlas = textureAtlas

				p.Chunks[x][y][z] = chunk
			}
		}
	}

	log.Println("Starting chunk load workers...")

	// Démarrer les workers de chargement
	for i := 0; i < p.chunkLoadWorkers; i++ {
		go p.chunkLoadWorker()
	}

	log.Println("Planet initialization complete")

	return nil
}

// SetWindow définit la fenêtre pour le contexte OpenGL
func (p *Planet) SetWindow(window *glfw.Window) {
	p.window = window
}

// chunkLoadWorker gère le chargement asynchrone des chunks
func (p *Planet) chunkLoadWorker() {
	for {
		select {
		case <-p.stopWorkers:
			return
		case chunkPriority := <-p.chunkLoadQueue:
			chunk := chunkPriority.chunk
			if chunk == nil {
				continue
			}

			// Vérifier si le chunk est déjà chargé
			p.chunkLoadMutex.Lock()
			if p.loadedChunks[chunk] {
				p.chunkLoadMutex.Unlock()
				continue
			}
			p.chunkLoadMutex.Unlock()

			// Générer les vertices du chunk (opération CPU uniquement)
			chunk.GenerateVertices()

			// Ajouter le chunk à la file d'attente pour la génération du mesh
			p.chunkLoadMutex.Lock()
			p.chunksToLoad = append(p.chunksToLoad, chunkPriority)
			p.chunkLoadMutex.Unlock()
		}
	}
}

// Update met à jour l'état de la planète
func (p *Planet) Update(deltaTime float32) {
	// Générer les meshes des chunks en attente (dans le thread principal)
	if len(p.chunksToLoad) > 0 {
		p.chunkLoadMutex.Lock()
		chunk := p.chunksToLoad[0].chunk
		p.chunksToLoad = p.chunksToLoad[1:]
		p.chunkLoadMutex.Unlock()

		if chunk != nil {
			// S'assurer que nous sommes dans le bon contexte OpenGL
			p.window.MakeContextCurrent()

			// Générer le mesh (opération OpenGL)
			chunk.GenerateMesh()

			if chunk.vao != 0 && chunk.vbo != 0 {
				p.chunkLoadMutex.Lock()
				p.loadedChunks[chunk] = true
				p.chunkLoadMutex.Unlock()
			}
		}
	}
}

// Render rend la planète
func (p *Planet) Render(currentTime float64) {
	if !p.isInit {
		return
	}

	// Calculer la distance entre la caméra et la planète
	distance := CameraInstance.Position.Sub(p.Position).Len()

	// Si la planète est trop loin, ne pas la rendre
	if distance > 1000.0 {
		return
	}

	// Mettre à jour la liste des chunks à charger
	p.chunkLoadMutex.Lock()
	for _, chunkRow := range p.Chunks {
		for _, chunkCol := range chunkRow {
			for _, chunk := range chunkCol {
				if chunk == nil {
					continue
				}
				// Ne pas ajouter les chunks déjà chargés
				if p.loadedChunks[chunk] {
					continue
				}
				chunkDistance := CameraInstance.Position.Sub(*chunk.Position).Len()
				if chunkDistance <= p.maxLoadDistance {
					p.chunksToLoad = append(p.chunksToLoad, ChunkPriority{
						chunk:    chunk,
						distance: float32(chunkDistance),
					})
				}
			}
		}
	}
	p.chunkLoadMutex.Unlock()

	// Mettre à jour la file d'attente de chargement
	p.updateChunkLoadQueue()

	// Rendre les chunks de la planète
	for _, chunkRow := range p.Chunks {
		for _, chunkCol := range chunkRow {
			for _, chunk := range chunkCol {
				if chunk != nil && p.loadedChunks[chunk] {
					chunk.Render(currentTime)
				}
			}
		}
	}
}

// Cleanup nettoie les ressources de la planète
func (p *Planet) Cleanup() {
	// Arrêter les workers
	close(p.stopWorkers)

	// Nettoyer les chunks
	for x := range p.Chunks {
		for y := range p.Chunks[x] {
			for z := range p.Chunks[x][y] {
				if p.Chunks[x][y][z] != nil {
					p.Chunks[x][y][z].Cleanup()
				}
			}
		}
	}
}

func (p *Planet) IsInit() bool {
	return p.isInit
}

var ErrPlanetNotInitialized = fmt.Errorf("the planet is not initialized")
var ErrPlanetAlreadyExist = fmt.Errorf("a planet with the same identifier already exist")

// updateChunkLoadQueue met à jour la file d'attente de chargement des chunks
func (p *Planet) updateChunkLoadQueue() {
	p.chunkLoadMutex.Lock()
	defer p.chunkLoadMutex.Unlock()

	// Trier les chunks par distance
	sort.Slice(p.chunksToLoad, func(i, j int) bool {
		return p.chunksToLoad[i].distance < p.chunksToLoad[j].distance
	})

	// Mettre à jour la file d'attente
	for _, priority := range p.chunksToLoad {
		// Vérifier si le chunk est déjà chargé
		if p.loadedChunks[priority.chunk] {
			continue
		}

		select {
		case p.chunkLoadQueue <- priority:
			// Chunk ajouté à la file d'attente
		default:
			// File d'attente pleine, ignorer ce chunk
		}
	}

	// Vider la liste des chunks à charger
	p.chunksToLoad = p.chunksToLoad[:0]
}
