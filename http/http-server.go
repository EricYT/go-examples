package main

import (
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
}
