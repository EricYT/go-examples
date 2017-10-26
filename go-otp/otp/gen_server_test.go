package otp

import (
	"context"
	"log"
	"testing"
	"time"
)

type fake struct {
}

func (f *fake) Init(args ...interface{}) (context.Context, error) {
	log.Println("fake initialize")
	return context.TODO(), nil
}

func (f *fake) HandleCall(ctx context.Context, req interface{}) (context.Context, interface{}, error) {
	log.Printf("fake handle call %#v\n", req)
	ctx = context.WithValue(ctx, "call", 1)
	return ctx, "pong", nil
}

func (f *fake) HandleInfo(ctx context.Context, req interface{}) (context.Context, error) {
	log.Printf("fake handle info %#v\n", req)
	ctx = context.WithValue(ctx, "info", 2)
	return ctx, nil
}

func (f *fake) HandleCast(ctx context.Context, req interface{}) (context.Context, error) {
	log.Printf("fake handle cast %#v\n", req)
	ctx = context.WithValue(ctx, "cast", 3)
	return ctx, nil
}

func (f *fake) Terminate(ctx context.Context, reason interface{}) error {
	log.Printf("fake terminate reason: %#v\n", reason)
	return nil
}

func TestGenServerCall(t *testing.T) {
	fakeName := Name("fake")
	f := &fake{}
	NewGenServer(fakeName, f, 1, 2, 3)
	time.Sleep(time.Second * 3)

	res, err := Call(fakeName, "ping")
	if err != nil {
		t.Errorf("test gen server call error: %s\n", err)
	}
	log.Printf("test gen server call return: %#v\n", res)
}
