package main

import (
	"testing"
	"time"
)

func TestButtonEmphasisRetargetsFromCurrentValue(t *testing.T) {
	var b studioButton
	start := time.Unix(0, 0)
	g := ctx{Now: start}
	if got := b.emphasis(g, 1); got != 0 {
		t.Fatalf("first hover frame = %v, want 0", got)
	}

	g.Now = start.Add(50 * time.Millisecond)
	mid := b.emphasis(g, 1)
	if mid <= 0 || mid >= 1 {
		t.Fatalf("hover at 50ms = %v, want value in (0, 1)", mid)
	}
	if got := b.emphasis(g, 0); got != mid {
		t.Fatalf("hover reversal = %v, want current value %v", got, mid)
	}

	g.Now = start.Add(170 * time.Millisecond)
	if got := b.emphasis(g, 0); got != 0 {
		t.Fatalf("hover after settling = %v, want 0", got)
	}
}

func TestMascotMotionSleepsBetweenIdlePulses(t *testing.T) {
	start := time.Unix(0, 0)
	if frame := mascotMotion(start, false); frame.next < 7*time.Second || frame.lift != 0 || frame.eyes != 1 {
		t.Fatalf("quiet mascot frame = %+v", frame)
	}

	if frame := mascotMotion(start.Add(8100*time.Millisecond), false); frame.next > 35*time.Millisecond || frame.eyes >= 0.5 || frame.lift >= 0 {
		t.Fatalf("idle pulse frame = %+v", frame)
	}

	if frame := mascotMotion(start.Add(100*time.Millisecond), true); frame.next > 35*time.Millisecond || frame.lift <= 0 {
		t.Fatalf("generating mascot frame = %+v", frame)
	}
}
