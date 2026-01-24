package main

import (
	"net"
	"log"
	"fmt"
	"time"
)

func SendReceiceMessage(){

	

}

func main(){
	serverSendConnection, err := net.Dial("tcp", "10.100.23.11:34933")
	if err!=nil{
		log.Fatalln(err)
	}
	buffer := make([]byte, 1024)
	length, err := serverSendConnection.Read(buffer)

	if err!=nil{
		log.Fatal(err)
	}

	fmt.Println("Message:", string(buffer[:length]))
	
	//writing to server and reading its answer

	msg := make([]byte, 1024) //lager minne som er 2014 byte
	copy(msg, []byte("Hei erver")) //kopierer melidngen min inn i minneomrdået med fast størrelse 
	serverSendConnection.Write(msg) //svarer server 
	fmt.Println("Message sent to server")
	serverSendConnection.SetDeadline(time.Now().Add(5*time.Second))
	for {
		n, err := serverSendConnection.Read(buffer)
		if err!=nil{
			log.Fatal(err)
		}
		fmt.Println("mottatt:", string(buffer[:n]))
	}

	//accepting connection 

	




}