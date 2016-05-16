package proto

//go:generine msgp

// test map type
type Foo struct {
	Actions map[string]Runer `msg:"action"`
}

type Runer interface {
	Run()
}

type Oper1 struct {
	Key   string `msg:"key"`
	Value string `msg:"value"`
}

func (o Oper1) Run() {}

type Oper2 struct {
	Key   string `msg:"key"`
	Value string `msg:"value"`
}

func (o Oper2) Run() {}
