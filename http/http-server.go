package main

import (
<<<<<<< HEAD
	"fmt"
	"io/ioutil"
	"net/http"
)

type handler struct{}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resBody, _ := ioutil.ReadAll(r.Body)
	fmt.Println("server receive body:", string(resBody))

	// print http header
	for key, value := range r.Header {
		fmt.Println("server Key:", key, " value:", value)
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write([]byte("\r\n\r\n\r\n"))
}

func main() {
	fmt.Println("Start http server")
	port := 9090
	http.ListenAndServe(fmt.Sprintf(":%d", port), new(handler))
=======
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

type Response struct {
	Key   string `json:"key"`
	Value string `json:"value`
}

func handler(w http.ResponseWriter, r *http.Request) {

	reqBody, _ := ioutil.ReadAll(r.Body)
	fmt.Println("handle message from client:", string(reqBody))

	w.Header().Set("Content-type", "application/json; charset=utf-8")
	res := Response{Key: "foo", Value: "world"}
	resJson, err := json.Marshal(res)
	if err != nil {
		log.Println("json ecnoding error ", err)
		return
	}
	w.Write(resJson)
	return
}

func main() {
	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", nil)
>>>>>>> dc650f08853d307c5e3169b322d2bd09b7d0420c
}
