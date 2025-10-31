package geometry

import "math"

func CalcArea2D(X, Z []float64, numPoints int, divisor float64) float64 {
	if len(X) < numPoints || len(Z) < numPoints {
		return 0
	}

	// Triangle fast path
	// if numPoints == 3 {
	// 	return CalcTriangleArea2D(X, Z) / divisor
	// }

	// 	// Rectangle fast path
	// 	if numPoints == 4 {
	// 		minX, maxX := minMax(X)
	// 		minZ, maxZ := minMax(Z)
	// 		return (maxX - minX) * (maxZ - minZ) / divisor
	// 	}

	return CalcPolygonArea2D(X, Z, numPoints)
}

func CalcTriangleArea2D(X, Z []float64) float64 {
	vertA := X[0] * (Z[1] - Z[2])
	vertB := X[1] * (Z[2] - Z[0])
	vertC := X[2] * (Z[0] - Z[1])

	return math.Abs(vertA+vertB+vertC) / 2
}

// Calculates the area of a 2D polygon (with or without irregular vertices) using the Shoelace formula.
func CalcPolygonArea2D(X, Z []float64, numPoints int) float64 {
	area := 0.0
	j := numPoints - 1
	for i := range numPoints {
		area += (X[j] + X[i]) * (Z[j] - Z[i])
		j = i
	}

	return math.Abs(area / 2)
}
