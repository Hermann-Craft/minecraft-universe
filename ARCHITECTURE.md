# Architecture du Projet - Minecraft Clone Go

## 🏗️ Vue d'ensemble

Ce projet suit les **12 principes du game engineering** et utilise une architecture modulaire basée sur la séparation des responsabilités (principe **Triarc**).

## 📁 Structure des Packages

```
minecraft-clone-go/
├── cmd/
│   └── server/                    # Point d'entrée principal
├── internal/                      # Code privé de l'application
│   ├── core/                      # Logique métier pure
│   │   ├── world/                 # Gestion du monde (Planet, Chunk, Block)
│   │   ├── physics/               # Physique et collisions
│   │   └── game/                  # Logique de jeu
│   ├── render/                    # Couche de rendu (Triarc)
│   │   ├── shaders/               # Gestion des shaders
│   │   ├── textures/              # Gestion des textures
│   │   └── meshes/                # Gestion des meshes
│   ├── input/                     # Gestion des entrées (Triarc)
│   └── utils/                     # Utilitaires
├── pkg/                           # Packages publics réutilisables
│   ├── math/                      # Mathématiques 3D
│   └── ecs/                       # Système ECS
└── resources/                     # Ressources (textures, shaders)
```

## 🎯 Principes Architecturaux

### 1. Triarc - Séparation Input-Simulation-Render

- **Input Layer** (`internal/input/`) : Gestion des entrées utilisateur
- **Simulation Layer** (`internal/core/`) : Logique métier et simulation
- **Render Layer** (`internal/render/`) : Rendu graphique

### 2. Explicit Scope - Configuration Centralisée

Toute configuration passe par `internal/utils/config.go` :
- Variables d'environnement
- Validation automatique
- Valeurs par défaut

### 3. Single-Authority - Responsabilités Uniques

Chaque composant a une responsabilité unique et bien définie :
- `InputManager` : Gestion des entrées
- `Renderer` : Gestion du rendu
- `Config` : Gestion de la configuration

## 🔧 Composants Principaux

### Configuration (`internal/utils/config.go`)

```go
type Config struct {
    Window     WindowConfig
    Graphics   GraphicsConfig
    World      WorldConfig
    Performance PerformanceConfig
}
```

**Variables d'environnement supportées :**
- `WINDOW_WIDTH`, `WINDOW_HEIGHT`
- `WORLD_CHUNK_SIZE`, `WORLD_CHUNK_HEIGHT`
- `PERF_CHUNK_WORKERS`, `PERF_MAX_CHUNK_DISTANCE`

### Gestionnaire d'Entrées (`internal/input/input_manager.go`)

```go
type InputManager struct {
    window      *glfw.Window
    keyStates   map[glfw.Key]bool
    mouseStates map[glfw.MouseButton]bool
    callbacks   map[string][]InputCallback
}
```

**Fonctionnalités :**
- Gestion des événements clavier/souris
- Système de callbacks
- États des touches/boutons

### Renderer (`internal/render/renderer.go`)

```go
type Renderer struct {
    window      *glfw.Window
    shaders     map[string]*shaderImpl
    meshes      *meshManagerImpl
    textures    *textureManagerImpl
    stats       *RenderStats
}
```

**Fonctionnalités :**
- Gestion des shaders, meshes et textures
- Statistiques de rendu
- Thread-safe

## 🚀 Utilisation

### Compilation

```bash
# Compiler le serveur
go build ./cmd/server

# Exécuter
./server
```

### Configuration

```bash
# Utiliser des variables d'environnement
export WINDOW_WIDTH=1024
export WINDOW_HEIGHT=768
export WORLD_CHUNK_SIZE=32
./server
```

### Tests

```bash
# Tester la configuration
go test ./internal/utils

# Tester tout le projet
go test ./...
```

## 📊 Métriques et Performance

Le renderer fournit des statistiques en temps réel :
- Nombre d'appels de rendu
- Nombre de triangles
- FPS
- Temps de frame

## 🔄 Évolutions Futures

### Phase 2 : Refactorisation des Composants Critiques
- [ ] Système Chunk modulaire
- [ ] Système Planet avec états explicites
- [ ] Système Camera découplé

### Phase 3 : Optimisations Performance
- [ ] Frustum culling
- [ ] Système de cache intelligent
- [ ] Rendu par batch

### Phase 4 : Tests et Documentation
- [ ] Tests unitaires complets
- [ ] Tests d'intégration
- [ ] Benchmarks

## 🛠️ Développement

### Ajouter un nouveau composant

1. **Créer l'interface** dans le package approprié
2. **Implémenter le composant** avec gestion d'erreurs
3. **Ajouter les tests** unitaires
4. **Documenter** l'API

### Respecter les principes

- **Triarc** : Séparer Input/Simulation/Render
- **Explicit Scope** : Toute configuration visible
- **Single-Authority** : Une responsabilité par composant
- **Fail Fast** : Gestion d'erreurs rigoureuse

## 📝 Notes de Migration

Cette architecture remplace l'ancienne structure monolithique :
- `main.go` → `cmd/server/main.go`
- `game.go` → Architecture modulaire
- `chunk.go` → Sera refactorisé en Phase 2
- `planet.go` → Sera refactorisé en Phase 2 
