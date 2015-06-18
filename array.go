package main

import "fmt"

func modify(arr [4]int) {
  arr[2] = 31
  fmt.Println("In modify:", arr)
}

func main() {
  array := [4]int{1, 2, 3, 4}

  modify(array)

  fmt.Println("Modify array : ", array)
}
