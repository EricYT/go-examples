package turnstile

import (
	"log"

	gofsm "github.com/smallnest/gofsm"
)

type Turnstile struct {
	ID         uint64
	EventCount uint64   //事件统计
	CoinCount  uint64   //投币事件统计
	PassCount  uint64   //顾客通过事件统计
	State      string   //当前状态
	States     []string //历史经过的状态
}

func initFSM() *gofsm.StateMachine {
	delegate := &gofsm.DefaultDelegate{P: &TurnstileEventProcessor{}}

	transitions := []gofsm.Transition{
		gofsm.Transition{From: "Locked", Event: "Coin", To: "Unlocked", Action: "check"},
		gofsm.Transition{From: "Locked", Event: "Push", To: "Locked", Action: "invalid-push"},
		gofsm.Transition{From: "Unlocked", Event: "Push", To: "Locked", Action: "pass"},
		gofsm.Transition{From: "Unlocked", Event: "Coin", To: "Unlocked", Action: "repeat-check"},
	}

	return gofsm.NewStateMachine(delegate, transitions...)
}

type TurnstileEventProcessor struct{}

func (p *TurnstileEventProcessor) OnExit(fromState string, args []interface{}) {
	log.Printf("OnExit: From state: %s", fromState)
}

func (p *TurnstileEventProcessor) Action(action string, fromState string, toState string, args []interface{}) {
	log.Printf("Action: From state: %s => state: %s action: %s", fromState, toState, action)
	ts := args[0].(*Turnstile)
	ts.State = toState
}

func (p *TurnstileEventProcessor) OnEnter(toState string, args []interface{}) {
	log.Printf("OnEnter: To state: %s", toState)
}
