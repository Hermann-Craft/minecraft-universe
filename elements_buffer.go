package main

import (
	"encoding/binary"

	"github.com/go-gl/gl/v4.6-core/gl"
)

type ElementsBuffer struct {
	ID uint32
}

func NewElementsBuffer(elements []uint32) *ElementsBuffer {
	newElementsBuffer := &ElementsBuffer{}
	gl.GenBuffers(1, &newElementsBuffer.ID)
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, newElementsBuffer.ID)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, binary.Size(elements), gl.Ptr(elements), gl.STATIC_DRAW)
	return newElementsBuffer
}

func (elementsBuffer *ElementsBuffer) Bind() {
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, elementsBuffer.ID)
}

func (elementsBuffer *ElementsBuffer) Unbind() {
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, 0)
}

func (elementsBuffer *ElementsBuffer) Delete() {
	gl.DeleteBuffers(1, &elementsBuffer.ID)
}
