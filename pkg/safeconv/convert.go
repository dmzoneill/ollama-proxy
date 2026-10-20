// Package safeconv converts integers without silent overflow.
package safeconv

import "math"

// Int32FromInt converts v to int32, saturating at the int32 limits.
func Int32FromInt(v int) int32 {
	return Int32FromInt64(int64(v))
}

// Int32FromInt64 converts v to int32, saturating at the int32 limits.
func Int32FromInt64(v int64) int32 {
	if v > math.MaxInt32 {
		return math.MaxInt32
	}
	if v < math.MinInt32 {
		return math.MinInt32
	}
	return int32(v)
}

// Uint32FromInt converts a non-negative int to uint32.
// Negative values become 0 and values above MaxUint32 saturate.
func Uint32FromInt(v int) uint32 {
	if v <= 0 {
		return 0
	}
	if uint64(v) > math.MaxUint32 {
		return math.MaxUint32
	}
	return uint32(v)
}

// Uint16FromInt converts a non-negative int to uint16, saturating out of range.
func Uint16FromInt(v int) uint16 {
	if v <= 0 {
		return 0
	}
	if v > math.MaxUint16 {
		return math.MaxUint16
	}
	return uint16(v)
}

// IntFromUint64 converts v to int, saturating at MaxInt.
func IntFromUint64(v uint64) int {
	if v > uint64(math.MaxInt) {
		return math.MaxInt
	}
	return int(v)
}

// Int64FromUint64 converts v to int64, saturating at MaxInt64.
func Int64FromUint64(v uint64) int64 {
	if v > uint64(math.MaxInt64) {
		return math.MaxInt64
	}
	return int64(v)
}

// Low8 returns the least significant 8 bits of v for little-endian encoding.
func Low8(v uint32) byte {
	return byte(v & 0xff)
}
