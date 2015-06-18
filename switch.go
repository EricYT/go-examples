package main

import (
	"fmt"
)

func main() {
	varString := "string"

	switch t := type(varString) {
	default:
		fmt.Printf("unexpected type %T", t)
	case bool:
		fmt.Printf("boolean  %t\n", t)
	case int:
		fmt.Printf("int  %t\n", t)
	}
}
