package main

import (
	"fmt"
	"os"
)

func main() {
	file, err := os.Open("for.go")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer file.Close()

	//read file state
	stat, err := file.Stat()
	if err != nil {
		fmt.Println(err)
		return
	}
	//read file size
	bs := make([]byte, stat.Size())
	_, err = file.Read(bs)
	if err != nil {
		fmt.Println(err)
		return
	}

	str := string(bs)
	fmt.Println(str)
}
