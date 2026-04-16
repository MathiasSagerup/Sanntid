package main

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

func findAddress() *net.UDPAddr {
	udpNullAdresse := net.UDPAddr{IP: net.IPv4zero, Port: 30000}
	buffer := make([]byte, 1024)

	fmt.Println("Starting to listen for server broadcast on port 30000...")

	udpConnection, err := net.ListenUDP("udp4", &udpNullAdresse)
	defer udpConnection.Close()

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Listening... waiting for broadcast message...")
	length, serverAddress, err := udpConnection.ReadFromUDP(buffer)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Server IP found:", serverAddress.String())
	fmt.Println("Message:", string(buffer[:length]))

	return serverAddress
}

func sendMessage(serverAddr *net.UDPAddr) {
	// Use workspace number n for port 20000 + n
	// Default to 11 for testing, but you can change this
	n := 11

	serverSendConnection, err := net.Dial("udp", fmt.Sprintf("%s:%d", serverAddr.IP.String(), 20000+n))
	if err != nil {
		log.Fatalln(err)
	}
	defer serverSendConnection.Close()

	for i := 0; i < 5; i++ {
		_, err = serverSendConnection.Write([]byte("Hello World!"))
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Println("Message sent")
		time.Sleep(100 * time.Millisecond) // Be nice to the network
	}
}

func printResponse(n int) {
	// Listen on port 20000 + n for responses
	udpListenAdresse := net.UDPAddr{IP: net.IPv4zero, Port: 20000 + n}
	listenConnection, err := net.ListenUDP("udp4", &udpListenAdresse)
	defer listenConnection.Close()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Listening for responses on port %d\n", 20000+n)

	buffer := make([]byte, 1024)
	listenConnection.SetDeadline(time.Now().Add(10 * time.Second))
	for {
		length, err := listenConnection.Read(buffer)

		if err != nil {
			log.Fatalln(err)
		}
		fmt.Println("Response:", string(buffer[:length]))
	}
}

func main() {
	n := 11 // Change this to your workspace number

	// First, find the server address by listening to port 30000
	serverAddr := findAddress()

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		printResponse(n)
	}()

	// Give the listener a moment to start before sending
	time.Sleep(100 * time.Millisecond)

	go func() {
		defer wg.Done()
		sendMessage(serverAddr)
	}()

	wg.Wait()
}
