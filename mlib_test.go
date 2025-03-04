package mlib

import (
    "testing"
    "time"
)

func TestDuration2Human(t *testing.T) {
    duration := time.Duration(time.Second * 3600 * 24)
    s := Duration2Human(duration, false, false)
    if s[:16] != "0 days, 24 hours" {
        t.Errorf("expected '0 days, 24 hours' at the beginning, got '%s'", s[:16])
    }

    s = Duration2Human(duration, true, false)
    if s[:8] != "24 hours" {
        t.Errorf("expected '24 hours' at the beginning, got '%s'", s[:8])
    }

    s = Duration2Human(duration, false, true)
    if len(s) != 8 {
        t.Errorf("expected 8 chars: '%s'", s)
    }
}
