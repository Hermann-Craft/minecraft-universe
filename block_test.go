package main

import "testing"

func TestNewBlock(t *testing.T) {
	b := NewBlock(BlockTypeDirt, [3]float32{0, 0, 0}, [3]float32{1, 1, 1})
	if b == nil {
		t.Error("NewBlock should not return nil")
	}
}
