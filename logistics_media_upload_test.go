package main

import "testing"

func TestPartSizeIsStable(t *testing.T) {
	if partSize != 8*1024*1024 {
		t.Fatalf("part size changed: %d", partSize)
	}
}
