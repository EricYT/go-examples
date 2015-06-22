package main

import "fmt"
import "time"
import "reflect"

type message struct {
  message string
  id uint64
}

func (msg *message) ShowMessage() {
  fmt.Println(msg.message)
}

func (msg *message) ShowId() {
  fmt.Println(msg.id)
}

func main() {
  msg := make(chan message, 1)

  msgTemplate := &message{
    message : "hello jone",
    id : 123,
  }

  msg<-*msgTemplate
  close(msg)

  tmp := <-msg
  tmp.ShowMessage()
  fmt.Println("msg type ", reflect.TypeOf(tmp))

  time.Sleep(time.Second*4)
}

