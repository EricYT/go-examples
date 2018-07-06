package main

import "fmt"

func sliceNotGrow(data []int) {
	data[0] = 100
}

func sliceGrow(data []int) []int {
	dataLen := len(data)
	dataCap := cap(data)

	for i := 0; i < (dataCap-dataLen)+1; i++ {
		data = append(data, i)
	}

	return data
}

func main() {
	data := make([]int, 3, 10)
	data[0] = 1
	data[1] = 2
	data[2] = 3
	fmt.Println(data)
	sliceNotGrow(data)
	fmt.Println(data)
	newone := sliceGrow(data)
	newone[0] = 20000
	fmt.Printf("data: %v newone: %v\n", data, newone)
}
