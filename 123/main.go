package main

import "fmt"

func find(arr []int) int {
	l := len(arr)
	sum := 0
	for _, val := range arr {
		sum += val
	}
	avg := float32(sum) / float32(l)
	res := arr[0]
	distance := abs(float32(arr[0]) - avg)
	for i := 1; i < len(arr); i++ {
		if abs(float32(arr[i])-avg) < distance {
			distance = abs(float32(arr[i]) - avg)
			res = arr[i]
		}
	}
	return res
}

func abs(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}

func main() {
	fmt.Println(find([]int{0, 1, 2, 3, 4}))

}
