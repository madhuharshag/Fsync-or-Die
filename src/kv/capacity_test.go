package kv

import (
	"errors"
	"math"
	"testing"
)

func TestCheckCapacity(t *testing.T) {
	// Unlimited
	if err := CheckCapacity(1000, 500, 0); err != nil {
		t.Fatalf("expected nil for unlimited cap, got %v", err)
	}

	// Fits exactly
	if err := CheckCapacity(500, 500, 1000); err != nil {
		t.Fatalf("expected nil when exactly fitting, got %v", err)
	}

	// Fits comfortably
	if err := CheckCapacity(100, 200, 1000); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}

	// Exceeds cap
	err := CheckCapacity(900, 101, 1000)
	if !errors.Is(err, ErrFull) {
		t.Fatalf("expected ErrFull, got %v", err)
	}

	// Negative offset
	err = CheckCapacity(-1, 100, 1000)
	if !errors.Is(err, ErrNegativeOffset) {
		t.Fatalf("expected ErrNegativeOffset, got %v", err)
	}

	// Overflow
	err = CheckCapacity(int64(math.MaxInt64), math.MaxUint64/2, 1000)
	if !errors.Is(err, ErrFull) {
		t.Fatalf("expected ErrFull on overflow, got %v", err)
	}
}

func TestCalculateCommittedCapacity(t *testing.T) {
	used, rem := CalculateCommittedCapacity(200, 1000)
	if used != 200 || rem != 800 {
		t.Fatalf("unexpected used=%d rem=%d", used, rem)
	}

	used, rem = CalculateCommittedCapacity(1200, 1000)
	if used != 1200 || rem != 0 {
		t.Fatalf("unexpected full capacity: used=%d rem=%d", used, rem)
	}

	used, rem = CalculateCommittedCapacity(500, 0)
	if used != 500 || rem != math.MaxUint64 {
		t.Fatalf("unexpected unlimited remaining capacity")
	}
}
