package packet

import (
	"fmt"
	"io"
)

// Packet represents a network packet
type Packet struct {
	Type byte
	Data []byte
}

// Read reads a packet from network connection
func Read(conn io.Reader) (*Packet, error) {
	// Read packet length
	lengthBuf := make([]byte, 1)
	n, err := io.ReadFull(conn, lengthBuf)
	if err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("connection closed by peer")
		}
		return nil, fmt.Errorf("failed to read packet length: %v (read %d bytes)", err, n)
	}
	length := int(lengthBuf[0])

	if length < 1 {
		return nil, fmt.Errorf("invalid packet length: %d", length)
	}

	// Read packet data
	data := make([]byte, length)
	n, err = io.ReadFull(conn, data)
	if err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("connection closed by peer while reading data")
		}
		return nil, fmt.Errorf("failed to read packet data: %v (read %d/%d bytes)", err, n, length)
	}

	packet := &Packet{
		Type: data[0],
		Data: data[1:],
	}
	return packet, nil
}

// Write writes a packet to network connection
func Write(conn io.Writer, p *Packet) error {
	// Prepare packet data
	data := append([]byte{p.Type}, p.Data...)

	// Write packet length
	if _, err := conn.Write([]byte{byte(len(data))}); err != nil {
		return fmt.Errorf("failed to write packet length: %v", err)
	}

	// Write packet data
	remaining := data
	for len(remaining) > 0 {
		n, err := conn.Write(remaining)
		if err != nil {
			return fmt.Errorf("failed to write packet data: %v", err)
		}
		remaining = remaining[n:]
	}

	return nil
}
