package main

import "fmt"

func main() {
	for i := 0; i < 10; i++ {
    // The L just a lable,break to layer L.Not a target
	L:
		for {
			//L:
			for {
				break L
				//        break
			}
			fmt.Println("GO out 1 ", i)
		}
		fmt.Println("GO out ", i)
	}
}
