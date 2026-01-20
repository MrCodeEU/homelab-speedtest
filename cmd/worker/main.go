package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/user/homelab-speedtest/internal/orchestrator"
)

func main() {
	mode := flag.String("mode", "", "Operation mode: server, client, ping")
	target := flag.String("target", "", "Target address (ip:port for client, ip for ping)")
	port := flag.Int("port", 8080, "Port to listen on (server mode)")
	// duration := flag.Int("duration", 10, "Test duration in seconds")

	flag.Parse()

	resp := orchestrator.WorkerResponse{Success: true}

	switch *mode {
	case orchestrator.ModeServer:
		runServer(*port)
	case orchestrator.ModeClient:
		runClient(*target, &resp)
	case orchestrator.ModePing:
		runPing(*target, &resp)
	default:
		fmt.Println("Usage: worker --mode [server|client|ping] ...")
		os.Exit(1)
	}
}

func runServer(port int) {
	// Simple TCP echo/sink server
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listening: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = ln.Close() }()
	fmt.Printf("Listening on :%d\n", port)

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer func() { _ = conn.Close() }()
	// Discard data
	buf := make([]byte, 32*1024)
	for {
		_, err := conn.Read(buf)
		if err != nil {
			return
		}
	}
}

func runClient(target string, resp *orchestrator.WorkerResponse) {
	// TCP throughput test
	start := time.Now()
	conn, err := net.Dial("tcp", target)
	if err != nil {
		resp.Success = false
		resp.Error = err.Error()
		printJson(resp)
		return
	}
	defer func() { _ = conn.Close() }()

	// Send data for 10 seconds (hardcoded for now, should be configurable)
	duration := 10 * time.Second
	deadline := start.Add(duration)
	_ = conn.SetDeadline(deadline)

	buf := make([]byte, 32*1024) // 32KB chunks
	var totalBytes int64

	for time.Now().Before(deadline) {
		n, err := conn.Write(buf)
		if err != nil {
			break
		}
		totalBytes += int64(n)
	}

	elapsed := time.Since(start).Seconds()
	mbps := (float64(totalBytes) * 8 / 1000000) / elapsed

	resp.BandwidthMbps = mbps
	printJson(resp)
}

func runPing(target string, resp *orchestrator.WorkerResponse) {
	// TCP Connect Ping
	// We expect target to be "host:port". If just host, we default to 80 (or the server port if internal)
	// But orchestrator should send host:port.
	
	start := time.Now()
	conn, err := net.DialTimeout("tcp", target, 2*time.Second)
	if err != nil {
		resp.Success = false
		resp.Error = err.Error()
		printJson(resp)
		return
	}
	defer func() { _ = conn.Close() }()
	
	latency := time.Since(start).Seconds() * 1000 // ms
	resp.LatencyMs = latency
	resp.Success = true
	printJson(resp)
}

func printJson(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	_ = enc.Encode(v)
}
