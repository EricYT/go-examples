package main

import (
	"fmt"

	"github.com/codegangsta/inject"
)

type SpecialString interface{}

type MyStruct struct {
	Key   string        `inject:"key"`
	Value SpecialString `inject:"value"`
	Index int64
}

func main() {
	injector := inject.New()
	key := "class"
	injector.Map(key)
	value := "MyStruct"
	injector.MapTo(value, (*SpecialString)(nil))

	fmt.Printf("injector is %+v\n", injector)

	var ms = MyStruct{}
	err := injector.Apply(&ms)
	if err != nil {
		fmt.Println("injector apply error:", err)
		return
	}

	fmt.Printf("MyStruct is %+v\n", ms)

	return
}
