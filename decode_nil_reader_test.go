package toml

import "testing"

func TestDecodeNilReader(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panicked: %v", r)
		}
	}()
	var v map[string]any
	err := NewDecoder(nil).Decode(&v)
	if err == nil {
		t.Fatal("want error")
	}
}
