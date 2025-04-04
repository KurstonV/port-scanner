// Purpose: This program demonstrates how to create a TCP network connection using Go

package main

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"time"
)

func main() {
    target := "scanme.nmap.org"
	port := 80
	portStr := strconv.Itoa(port)
	address := net.JoinHostPort(target, portStr)

	dialer := net.Dialer {
		Timeout: 5 * time.Second,
	}
  
  conn, err := dialer.Dial("tcp", address)
  if err != nil {
	  log.Fatalf("Unable to connect to %s: %v", address, err)
  }
  defer conn.Close() 
  
  fmt.Printf("Connection to %s was successful\n", address)
}