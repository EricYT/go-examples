package main

import "fmt"
import "reflect"

type Foo struct {
	x string `json:"x"`
	y int
}

type Bar interface {
	Foo()
}

func main() {

	var i int = 123
	var s string = "abc"

	bar := &Foo{
		x: "string",
		y: 123,
	}

	fmt.Println(reflect.TypeOf(i))
	fmt.Println(reflect.TypeOf(s))
	fmt.Println(reflect.TypeOf(bar))
	fmt.Println(reflect.TypeOf(bar).Elem())
	fmt.Println(reflect.TypeOf(bar).String())
	fmt.Println(reflect.TypeOf(bar).Name())

	fmt.Println(reflect.TypeOf((*Bar)(nil)).Elem())
	fmt.Println(reflect.TypeOf((*Bar)(nil)).String())
	fmt.Println(reflect.TypeOf((*Bar)(nil)).Name())

	//type filed
	var foo Foo
	typ := reflect.TypeOf(foo)
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fmt.Printf(" %s type is : %s\n", field.Name, field.Type)
	}

	field2, _ := typ.FieldByName("x")
	fmt.Println(field2.Name)

	// type interface
	typ1 := reflect.TypeOf((*Bar)(nil))
	for i := 0; i < typ1.NumField(); i++ {
		field := typ1.Field(i)
		fmt.Printf(" %s type is : %s\n", field.Name, field.Type)
	}

	field3, _ := typ1.FieldByName("x")
	fmt.Println(field3.Name)

}
