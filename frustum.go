package main

import "github.com/go-gl/mathgl/mgl32"

type Plane struct {
	Normal   mgl32.Vec3
	Distance float32
}

func extractFrustumPlanes(vp mgl32.Mat4) [6]Plane {
	var planes [6]Plane
	planes[0] = Plane{
		Normal:   mgl32.Vec3{vp[3] + vp[0], vp[7] + vp[4], vp[11] + vp[8]},
		Distance: vp[15] + vp[12],
	}
	planes[1] = Plane{
		Normal:   mgl32.Vec3{vp[3] - vp[0], vp[7] - vp[4], vp[11] - vp[8]},
		Distance: vp[15] - vp[12],
	}
	planes[2] = Plane{
		Normal:   mgl32.Vec3{vp[3] + vp[1], vp[7] + vp[5], vp[11] + vp[9]},
		Distance: vp[15] + vp[13],
	}
	planes[3] = Plane{
		Normal:   mgl32.Vec3{vp[3] - vp[1], vp[7] - vp[5], vp[11] - vp[9]},
		Distance: vp[15] - vp[13],
	}
	planes[4] = Plane{
		Normal:   mgl32.Vec3{vp[3] + vp[2], vp[7] + vp[6], vp[11] + vp[10]},
		Distance: vp[15] + vp[14],
	}
	planes[5] = Plane{
		Normal:   mgl32.Vec3{vp[3] - vp[2], vp[7] - vp[6], vp[11] - vp[10]},
		Distance: vp[15] - vp[14],
	}
	for i := 0; i < 6; i++ {
		norm := planes[i].Normal.Len()
		if norm != 0 {
			planes[i].Normal = planes[i].Normal.Mul(1.0 / norm)
			planes[i].Distance /= norm
		}
	}
	return planes
}

func cubeInFrustum(planes [6]Plane, center mgl32.Vec3, halfSize float32) bool {
	for _, plane := range planes {
		px := center.X()
		py := center.Y()
		pz := center.Z()
		if plane.Normal.X() >= 0 {
			px += halfSize
		} else {
			px -= halfSize
		}
		if plane.Normal.Y() >= 0 {
			py += halfSize
		} else {
			py -= halfSize
		}
		if plane.Normal.Z() >= 0 {
			pz += halfSize
		} else {
			pz -= halfSize
		}
		positiveVertex := mgl32.Vec3{px, py, pz}
		if plane.Normal.Dot(positiveVertex)+plane.Distance < 0 {
			return false
		}
	}
	return true
}
