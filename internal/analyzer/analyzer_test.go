package analyzer

import (
"testing"
"time"
)

func TestNew(t *testing.T) {
	a := New("python3", "script.py", 30*time.Second)
	if a == nil {
		t.Error("New() returned nil")
	}
}
