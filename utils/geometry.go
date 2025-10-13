package utils

import "math"

var ACTION_SPEEDS = struct {
	SNEAK  float32
	WALK   float32
	SPRINT float32
	BOAT   float32
}{
	SNEAK:  1.295,
	WALK:   4.317,
	SPRINT: 5.612,
	BOAT:   8.0,
}

// Calculates the "taxi-cab" distance, which is the distance between two points using the shortest path in a grid like manner.
// As opposed to Euclidean distance, Manhattan avoids diagonal intersections and is generally used for fast measurements where precision isn't required.
//
// For Minecraft, this is the function we will usually be using to get distance between blocks or chunks.
// This func is NOT suitable for measuring distance where very high accuracy is required such as within a single chunk
// or when exact player distance (not in blocks) is required. In such cases, Euclidean should always be preferred.
func ManhattanDistance2D(x1, x2, z1, z2 float64) float64 {
	return math.Abs(x2-x1) + math.Abs(z2-z1)
}

// Calculates the definitively fastest distance between two points, allowing intersections at any point, even through potential obstacles.
// This func is especially useful when precise measurements are required such as the distance between two players themselves,
// rather than the distance between the blocks they are located at.
func EuclideanDistance2D(x1, z1, x2, z2 float64) float64 {
	return math.Sqrt(math.Pow(x2-x1, 2) + math.Pow(z2-z1, 2))
}

// Checks whether two points along a single axis (X, Y, or Z) are within a specified distance of each other.
// It is neither Manhattan or Euclidean since they become identical in a single dimension.
// To check a point is within a two-dimensional radius, see WithinManhattanRadius2D and WithinEuclideanRadius2D.
func WithinRadius1D(a, b, radius float64) bool {
	return math.Abs(a-b) <= radius
}

// Uses Manhattan geometry to check whether a point is range of another point using a box.
// When both radius inputs are the same, this func essentially checks that the point (X, Z) is within a perfectly square box.
// Otherwise, when the inputs are different, it will check in a rectangular manner, spanning further in one direction than the other.
func WithinManhattanRadius2D(x, z, originX, originZ, radiusX, radiusZ float64) bool {
	return math.Abs(x-originX) <= radiusX && math.Abs(z-originZ) <= radiusZ
}

// Uses Euclidean geometry to check a point is within range of another point using a perfect circle.
func WithinEuclideanRadius2D(x, z, originX, originZ, radius float64) bool {
	return math.Hypot(x-originX, z-originZ) <= radius
}
