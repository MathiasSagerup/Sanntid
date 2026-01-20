package main

import (
	"fmt"
	"log"
	"net"
	"time"
	"sync"
)

func findAddress() *net.UDPAddr {
	udpNullAdresse := net.UDPAddr{IP: net.IPv4zero, Port: 30000}
	buffer := make([]byte, 1024)

	udpConnection , err := net.ListenUDP("udp4", &udpNullAdresse)
	defer udpConnection.Close()
	
	length, serverAddress, err := udpConnection.ReadFromUDP(buffer)

	fmt.Println("Message:", string(buffer[:length]))

	if err!=nil{
		log.Fatal(err)
	}

	return serverAddress
}

func sendMessage(){
	serverSendConnection, err := net.Dial("udp", "10.100.23.11:20011")
	if err!=nil{
		log.Fatalln(err)
	}
	defer serverSendConnection.Close()
	
	_, err = serverSendConnection.Write([]byte("Hello World!"))
	if err!=nil{
		log.Fatalln(err)
	}
	fmt.Println(("Message sent"))
}

func printResponse(){
	udpNullAdresse := net.UDPAddr{IP: net.IPv4zero, Port: 20011}
	listenConnection , err := net.ListenUDP("udp4", &udpNullAdresse)
	defer listenConnection.Close()
	if err!=nil{
		log.Fatal(err)
	}

	buffer := make([]byte, 1024)
	listenConnection.SetDeadline(time.Now().Add(5*time.Second))
	for{
		length , err := listenConnection.Read(buffer)

		if err!=nil{
			log.Fatalln(err)
		}
		fmt.Println("Message:", string(buffer[:length]))
	}
}

func main(){
	//findAddress()
	//ServerAddr := net.UDPAddr{IP: serverBroadcastAddress.IP, Port: 20011}
	//SendAddr := net.UDPAddr{IP: net.ParseIP("10.100.23.21"), Port: 2321}
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func(){
		defer wg.Done()
		printResponse()
	}()
	go func(){
		defer wg.Done()
		sendMessage()
	}()
	wg.Wait()
}