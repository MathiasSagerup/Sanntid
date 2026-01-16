package main

import (
	"fmt"
	"runtime"
)

func main() {
	fmt.Println("Go is properly installed!")
	fmt.Printf("Go version: %s\n", runtime.Version())
	fmt.Printf("OS: %s\n", runtime.GOOS)
	fmt.Printf("Architecture: %s\n", runtime.GOARCH)
	fmt.Printf("Github Repository test")
}
