package main

import (
	"fmt"

	"github.com/LyubomirIvanov05/nimbus/internals/broker"
)

func main() {
	fmt.Println("Nimbus server starting...")
	

	b := broker.NewBroker()
	b.Start()

	fmt.Println("Nimbus is running")
}
