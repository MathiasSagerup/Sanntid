package main

import (
	"fmt"
	"log"
	"net"
)

func findAddress() *net.UDPAddr {
	udpNullAdresse := net.UDPAddr{IP: net.IPv4zero, Port: 30000}
	buffer := make([]byte, 1024)

	udpConnection , err := net.ListenUDP("udp4", &udpNullAdresse)
	length, serverAddress, err := udpConnection.ReadFromUDP(buffer)

	fmt.Println("Message:", string(buffer[:length]))

	if err!=nil{
		log.Fatal(err)
	}

	return serverAddress
}

func sendMessage(serverSendConnection *net.UDPConn, message []byte) {
	serverSendConnection.WriteMsgUDP(message, nil, nil)
}

func main(){
	serverBroadcastAddress := findAddress()

	ServerAddr := net.UDPAddr{IP: serverBroadcastAddress.IP, Port: 20011}
	SendAddr := net.UDPAddr{IP: net.ParseIP("10.100.23.21"), Port: 2321}
	serverSendConnection, err := net.DialUDP("udp4", &SendAddr, &ServerAddr)
	
	if err!=nil{
		log.Fatal(err)
	}

	sendMessage(serverSendConnection, []byte("Hello World"))
	
	buffer := make([]byte, 1024)
	_, _, err = serverSendConnection.ReadFromUDP(buffer)

	if err!=nil{
		log.Fatal(err)
	}

	fmt.Println("Message:", string(buffer[:]))
}