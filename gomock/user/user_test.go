package user_test

import (
	"errors"
	"log"
	"testing"

	"github.com/EricYT/go-examples/gomock/mocks"
	"github.com/EricYT/go-examples/gomock/user"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDoer := mocks.NewMockDoer(ctrl)

	// test user
	testUser := &user.User{Doer: mockDoer}

	// Expect Do to be called once with 123 and "Hello GoMock"
	// as parameters, and return nil from the mocked call.
	mockDoer.EXPECT().DoSomething(123, "Hello GoMock").Return(nil).Times(1).Do(func(x int, y string) {
		log.Printf("mock get x: %d y: %s\n", x, y)
	})
	testUser.Use()

	mockDoer.EXPECT().DoSomething(123, "Hello GoMock").DoAndReturn(func(x int, y string) error {
		log.Printf("mock get x: %d y: %s trigger a error\n", x, y)
		// return nil will trigger a failed when assert.Nil check the .Use result
		return errors.New("balabala")
	})
	assert.NotNil(t, testUser.Use())
}
