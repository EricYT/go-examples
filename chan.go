package main

import "fmt"
import "time"

func main() {
	message := make(chan string)

	go func() {
		time.Sleep(time.Millisecond * 3000)
		message <- "ping"
	}()

	mes := <-message

	fmt.Println("Block until receive a message:", mes)
}
