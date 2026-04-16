package main

import (
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

// Connect to TCP server using fixed-size messages (port 34933)
func testFixedSizeMessages(serverIP string) {
	fmt.Println("\n=== Testing Fixed-Size Messages (Port 34933) ===")

	addr := fmt.Sprintf("%s:34933", serverIP)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Fatalln("Connection failed:", err)
	}
	defer conn.Close()

	// Read welcome message
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		log.Fatal("Failed to read welcome:", err)
	}
	fmt.Println("Welcome message:", strings.TrimSpace(string(buffer[:n])))

	// Send fixed-size message (must be exactly 1024 bytes)
	msg := make([]byte, 1024)
	copy(msg, []byte("Hei server!")) // Rest will be zeros
	_, err = conn.Write(msg)
	if err != nil {
		log.Fatalln("Failed to send:", err)
	}
	fmt.Println("Message sent")

	// Read echo response
	n, err = conn.Read(buffer)
	if err != nil {
		log.Fatal("Failed to read echo:", err)
	}
	fmt.Println("Echo response:", strings.TrimSpace(string(buffer[:n])))
}

// Connect to TCP server using null-terminated messages (port 33546)
func testNullTerminatedMessages(serverIP string) {
	fmt.Println("\n=== Testing Null-Terminated Messages (Port 33546) ===")

	addr := fmt.Sprintf("%s:33546", serverIP)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Fatalln("Connection failed:", err)
	}
	defer conn.Close()

	// Read welcome message (until null terminator)
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		log.Fatal("Failed to read welcome:", err)
	}
	fmt.Println("Welcome message:", strings.TrimRight(string(buffer[:n]), "\x00"))

	// Send null-terminated message
	msg := []byte("Hei server!\x00")
	_, err = conn.Write(msg)
	if err != nil {
		log.Fatalln("Failed to send:", err)
	}
	fmt.Println("Message sent")

	// Read echo response
	n, err = conn.Read(buffer)
	if err != nil {
		log.Fatal("Failed to read echo:", err)
	}
	fmt.Println("Echo response:", strings.TrimRight(string(buffer[:n]), "\x00"))
}

// Accept reverse connection from server
func acceptReverseConnection(serverIP string) {
	fmt.Println("\n=== Accepting Reverse Connection (Port 33546) ===")

	// Listen on a local port
	listener, err := net.Listen("tcp", "0.0.0.0:9999")
	if err != nil {
		log.Fatalln("Failed to listen:", err)
	}
	defer listener.Close()

	fmt.Println("Listening on port 9999 for reverse connection")

	// Get local IP (same first 3 bytes as server IP)
	localIP := net.ParseIP("10.100.23.21") // This will be replaced with your actual IP
	connectMsg := fmt.Sprintf("Connect to: %s:9999\x00", localIP.String())

	fmt.Println("Sending:", strings.TrimRight(connectMsg, "\x00"))

	// Connect to server and send the connect message
	serverConn, err := net.Dial("tcp", fmt.Sprintf("%s:33546", serverIP))
	if err != nil {
		log.Fatalln("Failed to connect to server:", err)
	}

	// Read welcome message
	buffer := make([]byte, 1024)
	n, err := serverConn.Read(buffer)
	if err != nil {
		log.Fatal("Failed to read welcome:", err)
	}
	fmt.Println("Welcome:", strings.TrimRight(string(buffer[:n]), "\x00"))

	// Send the "Connect to" message
	_, err = serverConn.Write([]byte(connectMsg))
	if err != nil {
		log.Fatalln("Failed to send connect message:", err)
	}

	// Now accept the reverse connection
	reverseConn, err := listener.Accept()
	if err != nil {
		log.Fatalln("Failed to accept connection:", err)
	}
	defer reverseConn.Close()

	fmt.Println("Server connected back from:", reverseConn.RemoteAddr())

	// Send a message on the reverse connection
	msg := []byte("Hello from reverse connection!\x00")
	_, err = reverseConn.Write(msg)
	if err != nil {
		log.Fatalln("Failed to send:", err)
	}
	fmt.Println("Message sent on reverse connection")

	// Read echo response
	n, err = reverseConn.Read(buffer)
	if err != nil {
		log.Fatal("Failed to read echo:", err)
	}
	fmt.Println("Echo response:", strings.TrimRight(string(buffer[:n]), "\x00"))

	// Clean up the server connection
	serverConn.Close()
}

func main() {
	serverIP := "10.100.23.21" // Change this to your server IP

	fmt.Println("TCP Exercise - Part 2")

	// Test fixed-size messages
	testFixedSizeMessages(serverIP)

	time.Sleep(500 * time.Millisecond)

	// Test null-terminated messages
	testNullTerminatedMessages(serverIP)

	time.Sleep(500 * time.Millisecond)

	// Test accepting reverse connection
	acceptReverseConnection(serverIP)
}
