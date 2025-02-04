package main

import "github.com/go-gl/mathgl/mgl32"

type GalaxyPosition mgl32.Vec3

type GalaxyRotation mgl32.Quat

type Galaxy struct {
	Planets []*Planet

	// Technicals
}
