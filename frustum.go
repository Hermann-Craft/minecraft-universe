package main

import (
	"github.com/go-gl/mathgl/mgl32"
)

type Frustum struct {
	planes [6]mgl32.Vec4
}

func NewFrustum() *Frustum {
	return &Frustum{}
}

// Update met à jour le frustum en fonction de la matrice de vue et de projection
func (f *Frustum) Update(view, projection mgl32.Mat4) {
	// Combiner les matrices de vue et de projection
	clip := projection.Mul4(view)

	// Extraire les plans du frustum
	// Plan droit
	f.planes[0] = mgl32.Vec4{
		clip[3] - clip[0],
		clip[7] - clip[4],
		clip[11] - clip[8],
		clip[15] - clip[12],
	}.Normalize()

	// Plan gauche
	f.planes[1] = mgl32.Vec4{
		clip[3] + clip[0],
		clip[7] + clip[4],
		clip[11] + clip[8],
		clip[15] + clip[12],
	}.Normalize()

	// Plan bas
	f.planes[2] = mgl32.Vec4{
		clip[3] + clip[1],
		clip[7] + clip[5],
		clip[11] + clip[9],
		clip[15] + clip[13],
	}.Normalize()

	// Plan haut
	f.planes[3] = mgl32.Vec4{
		clip[3] - clip[1],
		clip[7] - clip[5],
		clip[11] - clip[9],
		clip[15] - clip[13],
	}.Normalize()

	// Plan proche
	f.planes[4] = mgl32.Vec4{
		clip[3] + clip[2],
		clip[7] + clip[6],
		clip[11] + clip[10],
		clip[15] + clip[14],
	}.Normalize()

	// Plan lointain
	f.planes[5] = mgl32.Vec4{
		clip[3] - clip[2],
		clip[7] - clip[6],
		clip[11] - clip[10],
		clip[15] - clip[14],
	}.Normalize()
}

// IsBoxInFrustum vérifie si une boîte est dans le frustum
func (f *Frustum) IsBoxInFrustum(min, max mgl32.Vec3) bool {
	// Vérifier chaque plan du frustum
	for i := 0; i < 6; i++ {
		// Calculer le point le plus éloigné du plan
		p := mgl32.Vec3{
			min.X(),
			min.Y(),
			min.Z(),
		}
		if f.planes[i].X() >= 0 {
			p[0] = max.X()
		}
		if f.planes[i].Y() >= 0 {
			p[1] = max.Y()
		}
		if f.planes[i].Z() >= 0 {
			p[2] = max.Z()
		}

		// Si le point le plus éloigné est en dehors du plan, la boîte est en dehors du frustum
		if f.planes[i].X()*p.X()+f.planes[i].Y()*p.Y()+f.planes[i].Z()*p.Z()+f.planes[i].W() < 0 {
			return false
		}
	}
	return true
}
