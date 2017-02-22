package main

import "fmt"

func main() {
	fmt.Println("print go")

	const max = 0xff
	var value uint8
	var vs string
	for index := 0; index <= max; index++ {
		value = uint8(index)
		vs = fmt.Sprintf("%x%x", value>>4&0xf, value&0xf)
		fmt.Printf("index %d 0x: %s\n", value, vs)
	}
}
