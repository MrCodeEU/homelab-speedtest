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

	// Tiny delay to ensure buffers are flushed over SSH
	time.Sleep(100 * time.Millisecond)
}

func runServer(port int) {
	// Simple TCP echo/sink server
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Worker error listening: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = ln.Close() }()
	fmt.Fprintf(os.Stderr, "Worker server listening on :%d\n", port)

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Worker accept error: %v\n", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer func() { _ = conn.Close() }()
	fmt.Fprintf(os.Stderr, "Worker accepted connection from %s\n", conn.RemoteAddr())
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
	fmt.Fprintf(os.Stderr, "Worker client connecting to %s\n", target)
	// TCP throughput test
	start := time.Now()
	conn, err := net.DialTimeout("tcp", target, 5*time.Second)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Worker client dial error: %v\n", err)
		resp.Success = false
		resp.Error = fmt.Sprintf("dial error: %v", err)
		printJson(resp)
		return
	}
	defer func() { _ = conn.Close() }()
	fmt.Fprintf(os.Stderr, "Worker client connected, starting data transfer...\n")

	// Send data for 10 seconds
	duration := 10 * time.Second
	deadline := start.Add(duration)
	_ = conn.SetDeadline(deadline)

	buf := make([]byte, 32*1024) // 32KB chunks
	var totalBytes int64

	for time.Now().Before(deadline) {
		n, err := conn.Write(buf)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Worker client write error: %v\n", err)
			break
		}
		totalBytes += int64(n)
	}

	elapsed := time.Since(start).Seconds()
	mbps := (float64(totalBytes) * 8 / 1000000) / elapsed

	fmt.Fprintf(os.Stderr, "Worker client finished. Bytes sent: %d, Speed: %.2f Mbps\n", totalBytes, mbps)

	resp.BandwidthMbps = mbps
	resp.Success = true // Ensure success is true if we sent data
	printJson(resp)
}

func runPing(target string, resp *orchestrator.WorkerResponse) {
	fmt.Fprintf(os.Stderr, "Worker pinging %s\n", target)
	start := time.Now()
	conn, err := net.DialTimeout("tcp", target, 2*time.Second)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Worker ping error: %v\n", err)
		resp.Success = false
		resp.Error = err.Error()
		printJson(resp)
		return
	}
	defer func() { _ = conn.Close() }()

	latency := time.Since(start).Seconds() * 1000 // ms
	fmt.Fprintf(os.Stderr, "Worker ping success: %.2f ms\n", latency)
	resp.LatencyMs = latency
	resp.Success = true
	printJson(resp)
}

func printJson(v interface{}) {
	data, _ := json.Marshal(v)
	fmt.Println(string(data))
	os.Stdout.Sync()
}
