package main

import "fmt"
import "encoding/json"

type Json1 struct {
  Id int64
  Name string
  Index []int
  Addres []string
}

type Json2 struct {
  Id int64       `json:"id"`
  Name string    `json:"name"`
  Index []int    `json:"index"`
  Addres []string`json:"addres"`
}


func main() {
  str1 := "string"
  strJ, _ := json.Marshal(str1)
  fmt.Println(string(strJ))

  struct1 := &Json1{
    Id : 1234,
    Name : "zhou",
    Index : []int{1, 2, 3},
    Addres : []string{"a", "b", "c"},
  }
  structJ, _ := json.Marshal(struct1)
  fmt.Println(string(structJ))

  struct2 := &Json2{
    Id : 1234,
    Name : "eric",
    Index : []int{1, 2, 3},
    Addres : []string{"a", "b", "c"},
  }
  structJ1, _ := json.Marshal(struct2)
  fmt.Println(string(structJ1))

}
