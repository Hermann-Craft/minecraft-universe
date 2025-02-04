package main

import "github.com/go-gl/gl/v4.1-core/gl"

const inventoryVertexShaderSource = `
#version 410 core
layout (location = 0) in vec2 vert;        // Position en 2D
layout (location = 1) in vec2 vertTexCoord;  // Coordonnées de texture

out vec2 fragTexCoord;

uniform mat4 projection; // Projection orthographique

void main(){
    fragTexCoord = vertTexCoord;
    gl_Position = projection * vec4(vert, 0.0, 1.0);
}
` + "\x00"

const inventoryFragmentShaderSource = `
#version 410 core
in vec2 fragTexCoord;
out vec4 outputColor;

uniform sampler2D inventoryTexture;

void main(){
    outputColor = texture(inventoryTexture, fragTexCoord);
}
` + "\x00"

var inventoryVAO, inventoryVBO uint32

func initInventoryUI() {
	// Définition d'un quad en coordonnées écran (exprimées en pixels)
	// Ici, le quad est défini pour un slot situé en (0, 0) avec une taille de 64x64 pixels.
	vertices := []float32{
		// Positions        // Coordonnées de texture
		0, 0, 0.0, 0.0,
		64, 0, 1.0, 0.0,
		64, 64, 1.0, 1.0,

		0, 0, 0.0, 0.0,
		64, 64, 1.0, 1.0,
		0, 64, 0.0, 1.0,
	}

	gl.GenVertexArrays(1, &inventoryVAO)
	gl.GenBuffers(1, &inventoryVBO)

	gl.BindVertexArray(inventoryVAO)

	gl.BindBuffer(gl.ARRAY_BUFFER, inventoryVBO)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*4, gl.Ptr(vertices), gl.STATIC_DRAW)

	// Position (location = 0)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, 4*4, gl.PtrOffset(0))
	// Coordonnées de texture (location = 1)
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointer(1, 2, gl.FLOAT, false, 4*4, gl.PtrOffset(2*4))

	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
	gl.BindVertexArray(0)
}
