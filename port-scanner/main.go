// Purpose: This program demonstrates how to create a TCP network connection using Go

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ScanResult struct {
	Target string `json:"target"`
	Port   int    `json:"port"`
	Banner string `json:"banner,omitempty"`
	Open   bool   `json:"open"`
}

func worker(wg *sync.WaitGroup, tasks chan string, dialer net.Dialer, results chan ScanResult, showProgress bool, totalPorts int, progressChan chan int) {
	defer wg.Done()
	maxRetries := 3

	for task := range tasks {
		host, portStr, _ := net.SplitHostPort(task)
		port, _ := strconv.Atoi(portStr)

		if showProgress {
			progressChan <- 1
		}

		var success bool
		var banner string
		for i := 0; i < maxRetries; i++ {
			conn, err := dialer.Dial("tcp", task)
			if err == nil {
				// Banner grabbing
				conn.SetReadDeadline(time.Now().Add(2 * time.Second))
				buf := make([]byte, 1024)
				n, _ := conn.Read(buf)
				banner = strings.TrimSpace(string(buf[:n]))
				conn.Close()

				success = true
				break
			}
			time.Sleep(time.Duration(1<<i) * time.Second)
		}

		results <- ScanResult{
			Target: host,
			Port:   port,
			Banner: banner,
			Open:   success,
		}
	}
}

func parsePorts(portList string, startPort, endPort int) []int {
	if portList != "" {
		parts := strings.Split(portList, ",")
		ports := make([]int, 0, len(parts))
		for _, p := range parts {
			if val, err := strconv.Atoi(strings.TrimSpace(p)); err == nil {
				ports = append(ports, val)
			}
		}
		return ports
	}

	// Default to range
	ports := make([]int, 0)
	for p := startPort; p <= endPort; p++ {
		ports = append(ports, p)
	}
	return ports
}

func main() {
	// Flags

	targetsFlag := flag.String("targets", "scanme.nmap.org", "Comma-separated list of target hosts")
	startPort := flag.Int("start-port", 1, "Start port")
	endPort := flag.Int("end-port", 1024, "End port")
	portsFlag := flag.String("ports", "", "Comma-separated list of specific ports (overrides start/end)")
	workers := flag.Int("workers", 100, "Number of workers")
	timeout := flag.Float64("timeout", 3.0, "Connection timeout (seconds)")
	jsonOutput := flag.Bool("json", false, "Output results in JSON format")

	flag.Parse()

	if *targetsFlag == "" {
		fmt.Println("Error: -targets is required")
		flag.Usage()
		return
	}

	targets := strings.Split(*targetsFlag, ",")
	ports := parsePorts(*portsFlag, *startPort, *endPort)

	dialer := net.Dialer{
		Timeout: time.Duration(*timeout * float64(time.Second)),
	}

	var wg sync.WaitGroup
	tasks := make(chan string, len(targets)*len(ports))
	results := make(chan ScanResult, len(targets)*len(ports))
	progressChan := make(chan int, len(targets)*len(ports))

	start := time.Now()

	// Start workers
	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go worker(&wg, tasks, dialer, results, true, len(ports)*len(targets), progressChan)
	}

	// Enqueue tasks
	for _, target := range targets {
		for _, port := range ports {
			tasks <- net.JoinHostPort(strings.TrimSpace(target), strconv.Itoa(port))
		}
	}
	close(tasks)

	// Progress monitor (in separate goroutine)
	go func() {
		total := len(targets) * len(ports)
		count := 0
		for range progressChan {
			count++
			fmt.Printf("\rScanning port %d/%d", count, total)
		}
	}()

	wg.Wait()
	close(results)
	close(progressChan)

	// Collect results
	var openResults []ScanResult
	for res := range results {
		if res.Open {
			openResults = append(openResults, res)
		}
	}

	duration := time.Since(start)

	// Output results
	if *jsonOutput {
		json.NewEncoder(os.Stdout).Encode(openResults)
	} else {
		fmt.Println("\n\n====== Open Ports ======")
		for _, r := range openResults {
			fmt.Printf("%s:%d - %s\n", r.Target, r.Port, r.Banner)
		}
		fmt.Println("========================")
		fmt.Printf("Targets scanned:  %d\n", len(targets))
		fmt.Printf("Ports per target: %d\n", len(ports))
		fmt.Printf("Open ports found: %d\n", len(openResults))
		fmt.Printf("Time taken:       %s\n", duration)
	}
}
