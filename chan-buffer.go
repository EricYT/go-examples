package main

import "fmt"

func main() {
	// make a channel that capital is 2
	message := make(chan string, 2)

	message <- "foo"
	message <- "bar"

	/* error */
	//	message <- "third"

	fmt.Println(<-message)
	fmt.Println(<-message)
}
