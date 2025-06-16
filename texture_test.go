package main

import "testing"

func TestGetTextureAtlas(t *testing.T) {
	atlas := GetTextureAtlas()
	if atlas == nil {
		t.Error("GetTextureAtlas should not return nil")
	}
}
