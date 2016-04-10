package main

import (
	"fmt"
	"reflect"
	"sync"
	"time"
)

func main() {
	chans := make([]chan string, 10)

	for index, _ := range chans {
		chans[index] = make(chan string)
	}

	cases := make([]reflect.SelectCase, len(chans))
	for i, ch := range chans {
		cases[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ch)}
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		for {
			chosen, value, ok := reflect.Select(cases)
			if !ok {
				fmt.Println("select receive error")
				return
			}
			msg := value.String()
			chans[chosen] <- "pong"
			fmt.Println("Get msg from select cases:", msg)
		}
	}()
	time.Sleep(time.Second * 1)

	chans[2] <- "ping"

	fmt.Println("receive msg:", <-chans[2])

	time.Sleep(time.Second * 10)

	return
}
