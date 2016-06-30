package main

import (
	"flag"
	"fmt"
	"math/rand"

	"github.com/EricYT/go-examples/flags/bar"
)

var foo string

func init() {
	flag.StringVar(&foo, "foo", "default foo", "foo option")
}

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
	fmt.Println(foo)
	fmt.Println("bar is ", bar.Display())
}
