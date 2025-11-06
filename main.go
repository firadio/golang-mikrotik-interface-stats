package main

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Host     string
	Port     string
	Username string
	Password string
}

type InterfaceStats struct {
	Name   string
	RxByte uint64
	TxByte uint64
}

type InterfaceRate struct {
	Name       string
	LastRxByte uint64
	LastTxByte uint64
	LastTime   time.Time
}

func loadConfig() (*Config, error) {
	// Try to load from .env file
	if file, err := os.Open(".env"); err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		envMap := make(map[string]string)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				envMap[parts[0]] = parts[1]
			}
		}

		// Override with .env values if present
		for k, v := range envMap {
			os.Setenv(k, v)
		}
	}

	host := os.Getenv("MIKROTIK_HOST")
	port := os.Getenv("MIKROTIK_PORT")
	username := os.Getenv("MIKROTIK_USERNAME")
	password := os.Getenv("MIKROTIK_PASSWORD")

	if host == "" || port == "" || username == "" || password == "" {
		return nil, fmt.Errorf("missing required environment variables")
	}

	return &Config{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
	}, nil
}

type MikrotikClient struct {
	conn net.Conn
}

func NewMikrotikClient(config *Config) (*MikrotikClient, error) {
	address := net.JoinHostPort(config.Host, config.Port)
	conn, err := net.DialTimeout("tcp", address, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	client := &MikrotikClient{conn: conn}

	// Login
	if err := client.login(config.Username, config.Password); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to login: %w", err)
	}

	return client, nil
}

func (c *MikrotikClient) Close() error {
	return c.conn.Close()
}

func (c *MikrotikClient) writeWord(w string) error {
	length := len(w)
	var lengthBytes []byte

	if length < 0x80 {
		lengthBytes = []byte{byte(length)}
	} else if length < 0x4000 {
		lengthBytes = []byte{byte(length>>8) | 0x80, byte(length)}
	} else if length < 0x200000 {
		lengthBytes = []byte{byte(length>>16) | 0xC0, byte(length >> 8), byte(length)}
	} else if length < 0x10000000 {
		lengthBytes = []byte{byte(length>>24) | 0xE0, byte(length >> 16), byte(length >> 8), byte(length)}
	} else {
		lengthBytes = []byte{0xF0, byte(length >> 24), byte(length >> 16), byte(length >> 8), byte(length)}
	}

	if _, err := c.conn.Write(lengthBytes); err != nil {
		return err
	}
	if _, err := c.conn.Write([]byte(w)); err != nil {
		return err
	}
	return nil
}

func (c *MikrotikClient) readWord() (string, error) {
	c.conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	firstByte := make([]byte, 1)
	if _, err := io.ReadFull(c.conn, firstByte); err != nil {
		return "", err
	}

	var length int
	b := firstByte[0]

	if (b & 0x80) == 0 {
		length = int(b)
	} else if (b & 0xC0) == 0x80 {
		secondByte := make([]byte, 1)
		if _, err := io.ReadFull(c.conn, secondByte); err != nil {
			return "", err
		}
		length = ((int(b) & ^0x80) << 8) + int(secondByte[0])
	} else if (b & 0xE0) == 0xC0 {
		bytes := make([]byte, 2)
		if _, err := io.ReadFull(c.conn, bytes); err != nil {
			return "", err
		}
		length = ((int(b) & ^0xC0) << 16) + (int(bytes[0]) << 8) + int(bytes[1])
	} else if (b & 0xF0) == 0xE0 {
		bytes := make([]byte, 3)
		if _, err := io.ReadFull(c.conn, bytes); err != nil {
			return "", err
		}
		length = ((int(b) & ^0xE0) << 24) + (int(bytes[0]) << 16) + (int(bytes[1]) << 8) + int(bytes[2])
	} else if (b & 0xF8) == 0xF0 {
		bytes := make([]byte, 4)
		if _, err := io.ReadFull(c.conn, bytes); err != nil {
			return "", err
		}
		length = (int(bytes[0]) << 24) + (int(bytes[1]) << 16) + (int(bytes[2]) << 8) + int(bytes[3])
	}

	if length == 0 {
		return "", nil
	}

	data := make([]byte, length)
	if _, err := io.ReadFull(c.conn, data); err != nil {
		return "", err
	}

	return string(data), nil
}

func (c *MikrotikClient) sendCommand(words ...string) error {
	for _, word := range words {
		if err := c.writeWord(word); err != nil {
			return err
		}
	}
	return c.writeWord("")
}

func (c *MikrotikClient) readResponse() ([]map[string]string, error) {
	var result []map[string]string
	currentItem := make(map[string]string)
	debug := false // Set to true for debugging

	for {
		word, err := c.readWord()
		if err != nil {
			if debug {
				log.Printf("DEBUG readResponse: error reading word: %v", err)
			}
			return nil, err
		}

		if debug {
			log.Printf("DEBUG readResponse: word='%s'", word)
		}

		// Empty word is just a sentence delimiter in Mikrotik API, not end of response
		if word == "" {
			continue
		}

		if strings.HasPrefix(word, "!done") {
			if len(currentItem) > 0 {
				result = append(result, currentItem)
			}
			break
		} else if strings.HasPrefix(word, "!trap") || strings.HasPrefix(word, "!fatal") {
			return nil, fmt.Errorf("error response: %s", word)
		} else if strings.HasPrefix(word, "!re") {
			if len(currentItem) > 0 {
				result = append(result, currentItem)
				currentItem = make(map[string]string)
			}
		} else if strings.HasPrefix(word, "=") {
			parts := strings.SplitN(word[1:], "=", 2)
			if len(parts) == 2 {
				currentItem[parts[0]] = parts[1]
			}
		}
	}

	return result, nil
}

func (c *MikrotikClient) login(username, password string) error {
	// Send login command
	if err := c.sendCommand("/login", "=name="+username, "=password="+password); err != nil {
		return err
	}

	// Read response
	responses, err := c.readResponse()
	if err != nil {
		return err
	}

	// Check for challenge (old API)
	if len(responses) > 0 {
		if challenge, ok := responses[0]["ret"]; ok {
			// Old API with challenge
			hash := md5.Sum([]byte("\x00" + password + challenge))
			hashedPassword := hex.EncodeToString(hash[:])

			if err := c.sendCommand("/login", "=name="+username, "=response=00"+hashedPassword); err != nil {
				return err
			}

			_, err := c.readResponse()
			return err
		}
	}

	return nil
}

func (c *MikrotikClient) GetInterfaceStats() ([]InterfaceStats, error) {
	// Query with server-side filtering using Mikrotik API query syntax
	// =stats              : get real-time statistics (live counters)
	// =.proplist=         : only return specified properties (name, rx-byte, tx-byte)
	// ?name=              : filter where name equals the value
	// ?#|                 : OR operator (matches if any condition is true)
	// This sends only the filtered results from Mikrotik, reducing network traffic
	err := c.sendCommand(
		"/interface/print",
		"=stats",
		"=.proplist=name,rx-byte,tx-byte",
		"?name=vlan2622",
		"?name=vlan2624",
		"?#|",
	)
	if err != nil {
		return nil, err
	}

	responses, err := c.readResponse()
	if err != nil {
		return nil, err
	}

	var stats []InterfaceStats
	for _, resp := range responses {
		name := resp["name"]
		rxByteStr := resp["rx-byte"]
		txByteStr := resp["tx-byte"]

		if name == "" {
			continue
		}

		rxByte, _ := strconv.ParseUint(rxByteStr, 10, 64)
		txByte, _ := strconv.ParseUint(txByteStr, 10, 64)

		stats = append(stats, InterfaceStats{
			Name:   name,
			RxByte: rxByte,
			TxByte: txByte,
		})
	}

	return stats, nil
}

func formatBytes(bytes float64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%.2f B/s", bytes)
	}
	div, exp := float64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB/s", bytes/div, "KMGTPE"[exp])
}

func main() {
	config, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	client, err := NewMikrotikClient(config)
	if err != nil {
		log.Fatalf("Failed to connect to Mikrotik: %v", err)
	}
	defer client.Close()

	log.Printf("Connected to Mikrotik at %s:%s", config.Host, config.Port)

	// Store previous stats for rate calculation
	rateMap := make(map[string]*InterfaceRate)

	// Use ticker to avoid missed seconds
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// Get initial stats
	stats, err := client.GetInterfaceStats()
	if err != nil {
		log.Printf("Warning: Failed to get initial stats: %v", err)
	} else {
		now := time.Now()
		for _, stat := range stats {
			rateMap[stat.Name] = &InterfaceRate{
				Name:       stat.Name,
				LastRxByte: stat.RxByte,
				LastTxByte: stat.TxByte,
				LastTime:   now,
			}
		}
	}

	fmt.Println("\nMonitoring interface traffic (Ctrl+C to stop):")
	fmt.Println(strings.Repeat("=", 80))

	for range ticker.C {
		stats, err := client.GetInterfaceStats()
		if err != nil {
			log.Printf("Error getting stats: %v", err)
			continue
		}

		if len(stats) == 0 {
			// No matching interfaces found, skip silently
			continue
		}

		now := time.Now()
		timestamp := now.Format("2006-01-02 15:04:05")

		for _, stat := range stats {
			if prev, ok := rateMap[stat.Name]; ok {
				// Calculate time difference
				timeDiff := now.Sub(prev.LastTime).Seconds()
				if timeDiff > 0 {
					// Calculate rates (bytes per second)
					rxRate := float64(stat.RxByte-prev.LastRxByte) / timeDiff
					txRate := float64(stat.TxByte-prev.LastTxByte) / timeDiff

					// Update stored values for next calculation
					prev.LastRxByte = stat.RxByte
					prev.LastTxByte = stat.TxByte
					prev.LastTime = now

					// Print human-readable output
					fmt.Printf("[%s] %s: RX: %s  TX: %s\n",
						timestamp,
						stat.Name,
						formatBytes(rxRate),
						formatBytes(txRate),
					)
				}
			} else {
				// Initialize for new interface
				rateMap[stat.Name] = &InterfaceRate{
					Name:       stat.Name,
					LastRxByte: stat.RxByte,
					LastTxByte: stat.TxByte,
					LastTime:   now,
				}
			}
		}
	}
}
