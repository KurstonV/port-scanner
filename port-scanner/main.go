// Purpose: This program demonstrates how to create a TCP network connection using Go

package main

import (
	"flag"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"
)


func worker(wg *sync.WaitGroup, tasks chan string, dialer net.Dialer) {
	defer wg.Done()
	maxRetries := 3
	openPorts := 0
    for addr := range tasks {
		var success bool
		for i := range maxRetries {      
		conn, err := dialer.Dial("tcp", addr)
		
		if err == nil {
			conn.Close()
			fmt.Printf("Connection to %s was successful\n", addr)
			success = true

			// Send 1 to openPorts to count this
			openPorts++ 
			break
		}
		backoff := time.Duration(1<<i) * time.Second
		fmt.Printf("Attempt %d to %s failed. Waiting %v...\n", i+1,  addr, backoff)
		time.Sleep(backoff)
	    }
		if !success {
			fmt.Printf("Failed to connect to %s after %d attempts\n", addr, maxRetries)
		}
	}
}

func main() {
	// Define command-line flags
	target := "scanme.nmap.org"
	targetPtr := flag.String("target", target, "Target IP address or hostname")
	startPort := flag.Int("start-port", 1, "Start of port range to scan")
	endPort := flag.Int("end-port", 1024, "End of port range to scan")
	workers := flag.Int("workers", 100, "Number of concurrent scanning workers")
	flag.Parse()

	// Validate required flags
	if *targetPtr == "" {
		fmt.Println("Error: -target is required")
		flag.Usage()
		return
	}

	// Validate port range
	if *startPort < 1 || *endPort > 65535 || *startPort > *endPort {
		fmt.Println("Error: Invalid port range")
		return
	}

	if *workers < 1 {
		fmt.Println("Error: -workers must be at least 1")
		return
	}

	startTime := time.Now() // Start timing

	var wg sync.WaitGroup
	tasks := make(chan string, 100)
	openPorts := make(chan int, *endPort - *startPort + 1) // to track open ports

   

	dialer := net.Dialer {
		Timeout: 5 * time.Second,
	}
  
	// Start workers
    for i := 1; i <= *workers; i++ {
		wg.Add(1)
		go worker(&wg, tasks, dialer)
	}

	//ports := 512

	// Send tasks
	totalPorts := 0
	for p := *startPort; p <= *endPort; p++ {
		port := strconv.Itoa(p)
        address := net.JoinHostPort(*targetPtr, port)
		tasks <- address
		totalPorts++
	}
	close(tasks)
	wg.Wait()
	close(openPorts)

	// Count open ports
	openCount := 0
	for range openPorts {
		openCount++
	}

	duration := time.Since(startTime)

	// Summary
	fmt.Println("\n====== Scan Summary ======")
	fmt.Printf("Target:           %s\n", *targetPtr)
	fmt.Printf("Ports scanned:    %d\n", totalPorts)
	fmt.Printf("Open ports:       %d\n", openCount)
	fmt.Printf("Time taken:       %s\n", duration)
	fmt.Println("===========================")
}