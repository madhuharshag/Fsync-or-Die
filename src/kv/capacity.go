package kv

import (
	"fmt"
	"math"
)

// CheckCapacity verifies whether a proposed write of size additionalBytes can fit within
// maxBytes starting at currentOffset, guarding against integer overflow.
func CheckCapacity(currentOffset int64, additionalBytes uint64, maxBytes uint64) error {
	if currentOffset < 0 {
		return ErrNegativeOffset
	}
	if maxBytes == 0 {
		// 0 indicates unlimited maximum capacity
		return nil
	}

	cur := uint64(currentOffset)
	if math.MaxUint64-cur < additionalBytes {
		return fmt.Errorf("%w: file offset calculation overflow", ErrFull)
	}

	targetSize := cur + additionalBytes
	if targetSize > maxBytes {
		return fmt.Errorf("%w: target size %d exceeds maximum allowed capacity %d", ErrFull, targetSize, maxBytes)
	}
	return nil
}

// CalculateCommittedCapacity returns the current byte usage and remaining available capacity.
// A remaining value of math.MaxUint64 indicates unlimited capacity.
func CalculateCommittedCapacity(currentOffset int64, maxBytes uint64) (used uint64, remaining uint64) {
	if currentOffset <= 0 {
		used = 0
	} else {
		used = uint64(currentOffset)
	}
	if maxBytes == 0 {
		return used, math.MaxUint64
	}
	if used >= maxBytes {
		return used, 0
	}
	return used, maxBytes - used
}
