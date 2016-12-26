package main

import (
	"encoding/json"
	"fmt"
)

type WritePlan struct {
	ObjectId string   `json:"object_id"`
	Actions  []Action `json:"actions"`
}

type Action struct {
	ActionType string `json:"action_type"`
	Data       Data   `json:"data"`
}

type Data struct {
	Disks          *string `json:"disks,omited"`
	ExtentId       *string `json:"extent_id,omited"`
	ExtentGroupId  *string `json:"extent_group_id,omited"`
	OffsetAtExtent *int64  `json:"offset_at_extent,omited"`
}

func main() {
	fmt.Println("json go")
	data := `{"object_id":"o1", "actions":[{"action_type":"gopher", "data":{"disks":"d1", "extent_group_id":"eg1"}}]}`

	var action WritePlan
	err := json.Unmarshal([]byte(data), &action)
	if err != nil {
		panic(err)
	}
	fmt.Printf("json Unmarshal is:%+v\n", action)
}
