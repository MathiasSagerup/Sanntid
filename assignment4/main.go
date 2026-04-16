package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

const (
	PRIMARY_PORT       = "12345"
	BACKUP_PORT        = "12346"
	LOCALHOST          = "127.0.0.1"
	HEARTBEAT_INTERVAL = 100 * time.Millisecond
	TIMEOUT_DURATION   = 500 * time.Millisecond
	STATE_FILE         = "./process_pairs_state"
)

func readStateFromFile() int {
	data, err := os.ReadFile(STATE_FILE)
	if err != nil {
		return 0
	}
	counter, err := strconv.Atoi(string(data))
	if err != nil {
		return 0
	}
	return counter
}

func writeStateToFile(counter int) {
	os.WriteFile(STATE_FILE, []byte(strconv.Itoa(counter)), 0644)
}

// Primary process: counts and sends heartbeats
func runAsPrimary() {
	fmt.Println("[PRIMARY] Starting as primary")

	counter := readStateFromFile()

	// Create UDP connection for sending heartbeats to backup
	backupAddr, _ := net.ResolveUDPAddr("udp", LOCALHOST+":"+BACKUP_PORT)
	conn, err := net.DialUDP("udp", nil, backupAddr)
	if err != nil {
		fmt.Println("[PRIMARY] Warning: Could not create connection to backup:", err)
	}
	defer func() {
		if conn != nil {
			conn.Close()
		}
	}()

	// Spawn backup process
	spawnBackup()

	// Give backup time to start
	time.Sleep(500 * time.Millisecond)

	// Setup signal handler to cleanly exit
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(HEARTBEAT_INTERVAL)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			counter++
			fmt.Println(counter)
			writeStateToFile(counter)

			// Send heartbeat to backup (non-blocking)
			if conn != nil {
				conn.Write([]byte(fmt.Sprintf("%d", counter)))
			}

		case <-sigChan:
			fmt.Println("[PRIMARY] Shutting down")
			os.Exit(0)
		}
	}
}

// Backup process: waits for heartbeats and takes over if primary dies
func runAsBackup() {
	fmt.Println("[BACKUP] Starting as backup")

	addr, err := net.ResolveUDPAddr("udp", LOCALHOST+":"+BACKUP_PORT)
	if err != nil {
		fmt.Println("[BACKUP] Error resolving address:", err)
		os.Exit(1)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Println("[BACKUP] Error listening on "+BACKUP_PORT+":", err)
		// Port might already be in use, wait and retry
		time.Sleep(1 * time.Second)
		conn, err = net.ListenUDP("udp", addr)
		if err != nil {
			fmt.Println("[BACKUP] Still cannot listen:", err)
			os.Exit(1)
		}
	}
	defer conn.Close()

	buffer := make([]byte, 1024)
	consecutiveTimeouts := 0
	const TIMEOUT_THRESHOLD = 5

	for {
		conn.SetReadDeadline(time.Now().Add(TIMEOUT_DURATION))

		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			consecutiveTimeouts++
			if consecutiveTimeouts >= TIMEOUT_THRESHOLD {
				fmt.Println("[BACKUP] Primary detected as dead, taking over")
				conn.Close()
				runAsPrimary()
				return
			}
		} else {
			consecutiveTimeouts = 0
			// Update our knowledge of the current counter
			counter, err := strconv.Atoi(string(buffer[:n]))
			if err == nil {
				writeStateToFile(counter)
			}
		}
	}
}

func spawnBackup() {
	var cmd *exec.Cmd

	// Check if we're running from source (main.go exists) or compiled binary
	if _, err := os.Stat("main.go"); err == nil {
		// Running from source: use go run
		cmd = exec.Command("gnome-terminal", "--", "go", "run", "main.go", "backup")
	} else {
		// Running from compiled binary: use the executable
		exePath, err := os.Executable()
		if err != nil {
			fmt.Println("[PRIMARY] Error getting executable path:", err)
			return
		}
		cmd = exec.Command("gnome-terminal", "--", exePath, "backup")
	}

	err := cmd.Start()
	if err != nil {
		fmt.Println("[PRIMARY] Error spawning backup with gnome-terminal:", err)
		// Try alternative terminal
		if _, err := os.Stat("main.go"); err == nil {
			cmd = exec.Command("xterm", "-e", "go", "run", "main.go", "backup")
		} else {
			exePath, _ := os.Executable()
			cmd = exec.Command("xterm", "-e", exePath, "backup")
		}

		err = cmd.Start()
		if err != nil {
			fmt.Println("[PRIMARY] Error spawning backup with xterm:", err)
		}
	}
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "backup" {
		runAsBackup()
	} else {
		runAsPrimary()
	}
}
