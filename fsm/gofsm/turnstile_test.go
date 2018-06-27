package turnstile

import (
	"log"
	"testing"
)

func TestFSM(t *testing.T) {
	ts := &Turnstile{
		ID:     1,
		State:  "Locked",
		States: []string{"Locked"},
	}
	fsm := initFSM()

	//推门
	//没刷卡/投币不可进入
	log.Println("step: 1 event: Push")
	err := fsm.Trigger(ts.State, "Push", ts)
	if err != nil {
		t.Errorf("trigger err: %v", err)
	}

	//推门
	//没刷卡/投币不可进入
	log.Println("step: 2 event: Push")
	err = fsm.Trigger(ts.State, "Push", ts)
	if err != nil {
		t.Errorf("trigger err: %v", err)
	}

	//刷卡或者投币
	//不容易啊，终于解锁了
	log.Println("step: 3 event: Coin")
	err = fsm.Trigger(ts.State, "Coin", ts)
	if err != nil {
		t.Errorf("trigger err: %v", err)
	}

	//刷卡或者投币
	//这时才解锁
	log.Printf("step: 4 event: Coin current state: %s", ts.State)
	err = fsm.Trigger(ts.State, "Coin", ts)
	if err != nil {
		t.Errorf("trigger err: %v", err)
	}

	//推门
	//这时才能进入，进入后闸门被锁
	log.Printf("step: 5 event: Push current state: %s", ts.State)
	err = fsm.Trigger(ts.State, "Push", ts)
	if err != nil {
		t.Errorf("trigger err: %v", err)
	}

	//推门
	//无法进入，闸门已锁
	log.Println("step: 6 event: Push")
	err = fsm.Trigger(ts.State, "Push", ts)
	if err != nil {
		t.Errorf("trigger err: %v", err)
	}

	//lastState := Turnstile{
	//	ID:         1,
	//	EventCount: 6,
	//	CoinCount:  2,
	//	PassCount:  1,
	//	State:      "Locked",
	//	States:     []string{"Locked", "Unlocked", "Locked"},
	//}

	//if !gofsm.compareTurnstile(&lastState, ts) {
	//	t.Errorf("Expected last state: %+v, but got %+v", lastState, ts)
	//} else {
	//	t.Logf("最终的状态: %+v", ts)
	//}

}
