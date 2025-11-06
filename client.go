package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"
)

// Mikrotik API Client
// Implements the Mikrotik RouterOS API protocol
// Reference: https://help.mikrotik.com/docs/display/ROS/API
//
// Protocol uses length-encoded words with MD5 challenge-response authentication
// Supports both old API (with challenge) and new API (direct password)

// MikrotikClient represents a connection to a Mikrotik router
type MikrotikClient struct {
	conn net.Conn // TCP connection to Mikrotik API
}

// NewMikrotikClient creates a new Mikrotik API client and performs login
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

// Close closes the connection to the Mikrotik router
func (c *MikrotikClient) Close() error {
	return c.conn.Close()
}

// writeWord writes a word to the Mikrotik API using their length encoding
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

// readWord reads a word from the Mikrotik API using their length encoding
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

// sendCommand sends a command to the Mikrotik API
func (c *MikrotikClient) sendCommand(words ...string) error {
	for _, word := range words {
		if err := c.writeWord(word); err != nil {
			return err
		}
	}
	return c.writeWord("")
}

// readResponse reads a response from the Mikrotik API
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

// login performs authentication with the Mikrotik router
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
