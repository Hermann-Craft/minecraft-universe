package main

import (
	"encoding/binary"

	"github.com/go-gl/gl/v4.6-core/gl"
)

type VertexBuffer struct {
	ID uint32
}

func NewVertexBuffer(vertices []float32) *VertexBuffer {
	newVertexBuffer := &VertexBuffer{}
	gl.GenBuffers(1, &newVertexBuffer.ID)
	gl.BindBuffer(gl.ARRAY_BUFFER, newVertexBuffer.ID)
	gl.BufferData(gl.ARRAY_BUFFER, binary.Size(vertices), gl.Ptr(vertices), gl.STATIC_DRAW)
	return newVertexBuffer
}

func (vertexBuffer *VertexBuffer) Bind() {
	gl.BindBuffer(gl.ARRAY_BUFFER, vertexBuffer.ID)
}

func (vertexBuffer *VertexBuffer) Unbind() {
	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
}

func (vertexBuffer *VertexBuffer) Delete() {
	gl.DeleteBuffers(1, &vertexBuffer.ID)
}
