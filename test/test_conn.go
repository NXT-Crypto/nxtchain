package main

import (
	"bufio"
	"fmt"
	"net"
	"nxtchain/gonetic"
	"os"
	"strings"
	"time"
)

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

func main() {
	peer, err := gonetic.NewPeer(func(msg string) {
		fmt.Printf("Received: %s", msg)
	}, 10)

	if err != nil {
		fmt.Printf("Error creating peer: %v\n", err)
		return
	}

	go func() {
		if err := peer.Start(); err != nil {
			fmt.Printf("Error starting peer: %v\n", err)
		}
	}()

	time.Sleep(time.Second)

	localIP := getLocalIP()
	fmt.Printf("Your IP address: %s\n", localIP)
	fmt.Printf("Your connection string: %s\n", peer.GetConnString())
	fmt.Printf("Your public ip (MAKE SURE IF THIS WORKS): %s \n", peer.GetConnString())
	fmt.Printf("Port: %s\n", peer.Port)

	fmt.Println("\nTo connect to another peer, enter their connection string (or press Enter to skip):")
	var input string
	fmt.Scanln(&input)

	if strings.TrimSpace(input) != "" {
		if err := peer.Connect(input); err != nil {
			fmt.Printf("Error connecting to peer: %v\n", err)
		} else {
			fmt.Println("Connected successfully!")
		}
	}

	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("Start typing messages (or type 'exit' to quit):")

	for {
		if scanner.Scan() {
			message := scanner.Text()
			if message == "exit" {
				break
			}
			peer.Broadcast(message)
		}
	}

	peer.Stop()
	fmt.Println("\nPeer stopped")
}
