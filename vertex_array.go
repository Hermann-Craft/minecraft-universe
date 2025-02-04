package main

import (
	"github.com/go-gl/gl/v4.6-core/gl"
)

type VertexArray struct {
	ID uint32
}

func NewVertexArray() *VertexArray {
	newVertexArray := &VertexArray{}
	gl.GenVertexArrays(1, &newVertexArray.ID)
	return newVertexArray
}

func (vertexArray *VertexArray) LinkAttrib(vertexBuffer *VertexBuffer, layout uint32, numComponents int32, cType uint32, stride int32, offset int) {
	vertexBuffer.Bind()
	// gl.VertexAttribPointer(layout, 3, gl.FLOAT, false, 3*int32(unsafe.Sizeof(float32(0))), offset)
	gl.VertexAttribPointerWithOffset(layout, numComponents, cType, false, stride, uintptr(offset))
	gl.EnableVertexAttribArray(layout)
	vertexBuffer.Unbind()
}
func (vertexArray *VertexArray) Bind() {
	gl.BindVertexArray(vertexArray.ID)
}

func (vertexArray *VertexArray) Unbind() {
	gl.BindVertexArray(0)
}

func (vertexArray *VertexArray) Delete() {
	gl.DeleteVertexArrays(1, &vertexArray.ID)
}
