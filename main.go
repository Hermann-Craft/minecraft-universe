package main

import (
	"runtime"
)

func init() {
	// Tous les appels OpenGL doivent se faire dans le thread principal.
	runtime.LockOSThread()
}

const FPS_LIMIT = 60

//////////////////////////////////////////////////////////
// Fonction principale
//////////////////////////////////////////////////////////

var UnicubeGame *Game = &Game{
	FpsLimit: FPS_LIMIT,
}

func main() {
	UnicubeGame.Init()
}
