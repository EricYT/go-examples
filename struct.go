package main

import (
	"fmt"
	"math"
)

type Circle struct {
	x float64
	y float64
	r float64
}

func (c *Circle) area() float64 {
	return math.Pi * c.r * c.r
}

// type Circle struct { x, y, r float64}

// android and person struct
type Person struct {
	Name string
}

type Android struct {
	Person Person
	Model  string
}

func (pPtr *Person) name() string {
	fmt.Println("person name is ", pPtr.Name)
	return pPtr.Name
}

func main() {
	defer func() {
		str := recover()
		fmt.Println("error ", str)
	}()

	cir := Circle{0, 0, 5}

	fmt.Println(cir.x, cir.y, cir.r)
	fmt.Println(math.Pi)
	fmt.Println(cir.area())

	android := new(Android)
	android.Person.Name = "eric"
	android.Perso.name()
	//	android.Name = "yutao"
	android.Person.Name = "yutao"
	//	android.name()

}
