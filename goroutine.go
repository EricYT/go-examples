package main

import (
	"fmt"
	"math/rand"
	"time"
)

func do_loop(n int) {
	for i := 0; i < 10; i++ {
		fmt.Println(i, " : ", n)
		amt := time.Duration(rand.Intn(250))
		time.Sleep(time.Millisecond * amt)
	}
}

func do_loop_delay(n int) {
	for i := 0; i < 10; i++ {
		fmt.Println(n, " : ", i)
		amt := time.Duration(rand.Intn(250))
		time.Sleep(time.Millisecond * amt)
	}
}

func main() {
	go do_loop(10)

	for i := 0; i < 10; i++ {
		go do_loop_delay(i)
	}

	var input string
	fmt.Scanln(&input)
	fmt.Println("input is ", input)
}
