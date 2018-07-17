package doer

//go:generate mockgen -destination=../mocks/mock_doer.go -package=mocks github.com/EricYT/go-examples/gomock/doer Doer

type Doer interface {
	DoSomething(int, string) error

	// maybe we didn't care about it now
	DoOther() error
}
