package main

import (
	"fmt"

	"github.com/hashicorp/memberlist"
)

func main() {
	/* Create the initial memberlist from a safe configuration.
	   Please reference the godoc for other default config types.
	   http://godoc.org/github.com/hashicorp/memberlist#Config
	*/
	list, err := memberlist.Create(memberlist.DefaultLocalConfig())
	if err != nil {
		panic("Failed to create memberlist: " + err.Error())
	}

	// Join an existing cluster by specifying at least one known member.
	n, err := list.Join([]string{"1.2.3.4"})
	if err != nil {
		panic("Failed to join cluster: " + err.Error())
	}

	fmt.Println("join result:", n)

	// Ask for members of the cluster
	for _, member := range list.Members() {
		fmt.Printf("Member: %s %s\n", member.Name, member.Addr)
	}

	// Continue doing whatever you need, memberlist will maintain membership
	// information in the background. Delegates can be used for receiving
	// events when members join or leave.
}
