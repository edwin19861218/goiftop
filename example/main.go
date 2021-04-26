package main

import (
	"fmt"
	"github.com/google/gopacket/pcap"
	"time"
)

func main() {
	fmt.Println(time.Now())
	if ifaces, err := pcap.FindAllDevs(); err == nil {
		for _, iface := range ifaces {
			fmt.Println(iface)
		}
	}
}
