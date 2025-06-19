package input

import (
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

// InputManager gère toutes les entrées utilisateur
type InputManager struct {
	window          *glfw.Window
	keyStates       map[glfw.Key]bool
	prevKeyStates   map[glfw.Key]bool // État des touches à la frame précédente
	mouseStates     map[glfw.MouseButton]bool
	prevMouseStates map[glfw.MouseButton]bool
	mousePos        mgl32.Vec2
	mouseDelta      mgl32.Vec2
	lastMousePos    mgl32.Vec2
	firstMove       bool // Pour gérer le premier mouvement de souris
	callbacks       map[string][]InputCallback
}

// InputCallback fonction de callback pour les événements d'entrée
type InputCallback func(event InputEvent)

// InputEvent représente un événement d'entrée
type InputEvent struct {
	Type      InputEventType
	Key       glfw.Key
	Button    glfw.MouseButton
	Position  mgl32.Vec2
	Delta     mgl32.Vec2
	Modifiers glfw.ModifierKey
}

// InputEventType type d'événement d'entrée
type InputEventType int

const (
	EventKeyPress InputEventType = iota
	EventKeyRelease
	EventKeyRepeat
	EventMousePress
	EventMouseRelease
	EventMouseMove
	EventScroll
)

// NewInputManager crée un nouveau gestionnaire d'entrées
func NewInputManager(window *glfw.Window) *InputManager {
	im := &InputManager{
		window:          window,
		keyStates:       make(map[glfw.Key]bool),
		prevKeyStates:   make(map[glfw.Key]bool),
		mouseStates:     make(map[glfw.MouseButton]bool),
		prevMouseStates: make(map[glfw.MouseButton]bool),
		callbacks:       make(map[string][]InputCallback),
		firstMove:       true,
	}

	im.setupCallbacks()
	return im
}

// setupCallbacks configure les callbacks GLFW
func (im *InputManager) setupCallbacks() {
	im.window.SetKeyCallback(im.keyCallback)
	im.window.SetMouseButtonCallback(im.mouseButtonCallback)
	im.window.SetCursorPosCallback(im.cursorPosCallback)
	im.window.SetScrollCallback(im.scrollCallback)
}

// keyCallback callback pour les événements clavier
func (im *InputManager) keyCallback(window *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	event := InputEvent{
		Key:       key,
		Modifiers: mods,
	}

	switch action {
	case glfw.Press:
		im.keyStates[key] = true
		event.Type = EventKeyPress
		im.triggerCallbacks("keyPress", event)
	case glfw.Release:
		im.keyStates[key] = false
		event.Type = EventKeyRelease
		im.triggerCallbacks("keyRelease", event)
	case glfw.Repeat:
		event.Type = EventKeyRepeat
		im.triggerCallbacks("keyRepeat", event)
	}
}

// mouseButtonCallback callback pour les événements souris
func (im *InputManager) mouseButtonCallback(window *glfw.Window, button glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
	event := InputEvent{
		Button:    button,
		Position:  im.mousePos,
		Modifiers: mods,
	}

	switch action {
	case glfw.Press:
		im.mouseStates[button] = true
		event.Type = EventMousePress
		im.triggerCallbacks("mousePress", event)
	case glfw.Release:
		im.mouseStates[button] = false
		event.Type = EventMouseRelease
		im.triggerCallbacks("mouseRelease", event)
	}
}

// cursorPosCallback callback pour le mouvement de la souris
func (im *InputManager) cursorPosCallback(window *glfw.Window, xpos, ypos float64) {
	currentPos := mgl32.Vec2{float32(xpos), float32(ypos)}

	if im.firstMove {
		im.lastMousePos = currentPos
		im.firstMove = false
	}

	im.mouseDelta = currentPos.Sub(im.lastMousePos)
	im.lastMousePos = currentPos
	im.mousePos = currentPos

	event := InputEvent{
		Type:     EventMouseMove,
		Position: im.mousePos,
		Delta:    im.mouseDelta,
	}

	im.triggerCallbacks("mouseMove", event)
}

// scrollCallback callback pour le scroll
func (im *InputManager) scrollCallback(window *glfw.Window, xoff, yoff float64) {
	event := InputEvent{
		Type:  EventScroll,
		Delta: mgl32.Vec2{float32(xoff), float32(yoff)},
	}

	im.triggerCallbacks("scroll", event)
}

// triggerCallbacks déclenche les callbacks pour un événement donné
func (im *InputManager) triggerCallbacks(eventType string, event InputEvent) {
	if callbacks, exists := im.callbacks[eventType]; exists {
		for _, callback := range callbacks {
			callback(event)
		}
	}
}

// OnKeyPress enregistre un callback pour les pressions de touches
func (im *InputManager) OnKeyPress(callback InputCallback) {
	im.callbacks["keyPress"] = append(im.callbacks["keyPress"], callback)
}

// OnKeyRelease enregistre un callback pour les relâchements de touches
func (im *InputManager) OnKeyRelease(callback InputCallback) {
	im.callbacks["keyRelease"] = append(im.callbacks["keyRelease"], callback)
}

// OnMousePress enregistre un callback pour les pressions de souris
func (im *InputManager) OnMousePress(callback InputCallback) {
	im.callbacks["mousePress"] = append(im.callbacks["mousePress"], callback)
}

// OnMouseRelease enregistre un callback pour les relâchements de souris
func (im *InputManager) OnMouseRelease(callback InputCallback) {
	im.callbacks["mouseRelease"] = append(im.callbacks["mouseRelease"], callback)
}

// OnMouseMove enregistre un callback pour les mouvements de souris
func (im *InputManager) OnMouseMove(callback InputCallback) {
	im.callbacks["mouseMove"] = append(im.callbacks["mouseMove"], callback)
}

// OnScroll enregistre un callback pour le scroll
func (im *InputManager) OnScroll(callback InputCallback) {
	im.callbacks["scroll"] = append(im.callbacks["scroll"], callback)
}

// IsKeyPressed vérifie si une touche est pressée
func (im *InputManager) IsKeyPressed(key glfw.Key) bool {
	return im.keyStates[key]
}

// IsKeyJustPressed vérifie si une touche vient d'être pressée (activée à cette frame)
func (im *InputManager) IsKeyJustPressed(key glfw.Key) bool {
	return im.keyStates[key] && !im.prevKeyStates[key]
}

// IsMousePressed vérifie si un bouton de souris est pressé
func (im *InputManager) IsMousePressed(button glfw.MouseButton) bool {
	return im.mouseStates[button]
}

// IsMouseJustPressed vérifie si un bouton de la souris vient d'être pressé
func (im *InputManager) IsMouseJustPressed(button glfw.MouseButton) bool {
	return im.mouseStates[button] && !im.prevMouseStates[button]
}

// GetMousePosition retourne la position actuelle de la souris
func (im *InputManager) GetMousePosition() mgl32.Vec2 {
	return im.mousePos
}

// GetMouseDelta retourne le delta de mouvement de la souris
func (im *InputManager) GetMouseDelta() mgl32.Vec2 {
	return im.mouseDelta
}

// Update met à jour l'état du gestionnaire d'entrées. Doit être appelé une fois par frame.
func (im *InputManager) Update() {
	// Réinitialiser le delta de la souris à chaque frame pour éviter les mouvements persistants
	im.mouseDelta = mgl32.Vec2{0, 0}

	// 1. Sauvegarder l'état précédent des touches et de la souris
	for k, v := range im.keyStates {
		im.prevKeyStates[k] = v
	}
	for k, v := range im.mouseStates {
		im.prevMouseStates[k] = v
	}

	// 2. Mettre à jour l'état actuel en interrogeant GLFW
	glfw.PollEvents()
}

// Cleanup nettoie les ressources
func (im *InputManager) Cleanup() {
	// Les callbacks GLFW sont automatiquement nettoyés quand la fenêtre est détruite
}
