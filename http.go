package main

import "fmt"
import "os"
import "io"
import "net/http"

func main() {
  if len(os.Args) != 2 {
    fmt.Println("Usage : ", os.Args[0], " service")
    os.Exit(1)
  }

  service := os.Args[1]

  resp, err := http.Get(service)
  if err != nil {
    fmt.Println(err)
    os.Exit(2)
  }
  defer resp.Body.Close()
  io.Copy(os.Stdout, resp.Body)
}


