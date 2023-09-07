package utils

import "math"

func NormalizeVec(d int, v []float32) {
	var norm float64
	for i := 0; i < d; i++ {
		norm += float64(v[i]) * float64(v[i])
	}
	norm = math.Sqrt(norm)
	for i := 0; i < d; i++ {
		v[i] = float32(float64(v[i]) / norm)
	}
}

func FlattenFloat32Slice(input [][]float32) []float32 {

	var result []float32

	for _, arr := range input {
		result = append(result, arr...)

		//for _, item := range arr {
		//	result = append(result, item)
		//}
	}
	return result
}
