package main

import (
	"fmt"
	"time"

	mc "github.com/wang-laoban/mcprotocol"
)

func main() {
	client, err := mc.NewMitsubishiClient(mc.A_1E, "127.0.0.1", 6000, 0)
	// client, err := mc.NewMitsubishiClient(mc.Qna_3E, "127.0.0.1", 6001, 0)
	if err != nil {
		panic(err)
	}
	fmt.Println("start Connecting")
	err = client.Connect()
	if err != nil {
		panic(err)
	}
	for {
		v, err := client.ReadBool("M100")
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println("read done ", v)
		time.Sleep(time.Second)
	}
}
