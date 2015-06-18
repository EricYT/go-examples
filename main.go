package main

import "fmt"
import "os"
import "channel"

// This is a comment

func main() {
	fmt.Println("Hello world")
	fmt.Println("1 + 1 =", 1+1)
	fmt.Println(len("hello world"))
	fmt.Println("hello world"[1])
	fmt.Println("hello " + "world")
	os.Exit(1)
}
