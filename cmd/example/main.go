package main

import (
	"fmt"

	mc "github.com/wang-laoban/mcprotocol"
)

func main() {
	client, err := mc.NewMitsubishiClient(mc.A_1E, "127.0.0.1", 6000, 0)
	// client, err := mc.NewMitsubishiClient(mc.Qna_3E, "127.0.0.1", 6000, 0)
	if err != nil {
		panic(err)
	}

	fmt.Println("start Connecting")
	err = client.Connect()
	if err != nil {
		panic(err)
	}

	// read data from the PLC
	v, err := client.ReadBool("M100")
	if err != nil {
		panic(err)
	}
	fmt.Println("read M100 :", v)

	v1, err := client.ReadInt64("M101")
	if err != nil {
		panic(err)
	}
	fmt.Println("read M101 :", v1)
}
