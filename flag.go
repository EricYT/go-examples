package main

import (
	"flag"
	"fmt"
	"math/rand"
)

func main() {
	// Define Flags
	maxp := flag.Int("max", 6, "the max value")

	var strPtr *string
	strPtr = flag.String("str", "default", "the string")
	// Parse
	flag.Parse()
	// Generate a random value between 0 and max
	fmt.Println(rand.Intn(*maxp))
	fmt.Println(*strPtr)

	for _, value := range flag.Args() {
		fmt.Println(value)
	}
}
