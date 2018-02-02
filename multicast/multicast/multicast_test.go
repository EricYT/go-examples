package multicast

import (
	"encoding/hex"
	"log"
	"net"
	"testing"
	"time"
)

var (
	addSrv string = "224.0.0.1:9999"
)

func msgHandler(src *net.UDPAddr, n int, data []byte) {
	log.Println(n, " bytes read from ", src)
	log.Println(hex.Dump(data[:n]))
}

func msgSend(t *testing.T, c chan int) {
	var count int
	for count = 0; count < 5; count++ {
		n, err := Notify(addSrv, []byte("Hello, world."))
		if n != 13 {
			t.Errorf("msg send %d not %d", n, 13)
		}
		if err != nil {
			t.Errorf("msg send error: %s", err)
		}
		time.Sleep(time.Second * 1)
	}
	c <- count
}

func TestMulticastServer(t *testing.T) {
	c := make(chan int)
	srv, err := NewServer(addSrv)
	if err != nil {
		t.Errorf("new server error: %s", err)
	}
	go msgSend(t, c)
	err = srv.RegisterHandler("msgHandler", msgHandler)
	if err != nil {
		t.Errorf("register handler error: %s", err)
	}
	nn := <-c
	if nn != 5 {
		t.Errorf("message send %d got %d", 5, nn)
	}
}

func TestMulticast(t *testing.T) {
	c := make(chan int)
	err := RegisterMulticast(addSrv, "msgHandler", msgHandler)
	if err != nil {
		t.Errorf("register multicast error: %s", err)
	}
	go msgSend(t, c)
	nn := <-c
	if nn != 5 {
		t.Errorf("message send %d got %d", 5, nn)
	}
}
