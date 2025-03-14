package serial

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"comserver/config"

	"go.bug.st/serial"
)

const (
	maxPacketSize       = 254
	flowControlInterval = 5 * time.Millisecond
)

// Packet represents a network packet
type Packet struct {
	Type byte
	Data []byte
}

// readPacket reads a packet from network connection
func readPacket(conn net.Conn) (*Packet, error) {
	// Read packet length
	lengthBuf := make([]byte, 1)
	if n, err := io.ReadFull(conn, lengthBuf); err != nil {
		return nil, fmt.Errorf("failed to read packet length: %v", err)
	} else {
		log.Printf("Reading packet, length: %d", n)
	}
	length := int(lengthBuf[0])

	if length < 1 {
		return nil, fmt.Errorf("invalid packet length: %d", length)
	}

	// Read packet data
	data := make([]byte, length)
	if n, err := io.ReadFull(conn, data); err != nil {
		return nil, fmt.Errorf("failed to read packet data: %v", err)
	} else {
		log.Printf("Reading packet data, length: %d", n)
	}

	packet := &Packet{
		Type: data[0],
		Data: data[1:],
	}
	log.Printf("Read packet: type=0x%02x, data length=%d", packet.Type, len(packet.Data))
	return packet, nil
}

// writePacket writes a packet to network connection
func writePacket(conn net.Conn, p *Packet) error {
	// Prepare packet data
	data := append([]byte{p.Type}, p.Data...)

	log.Printf("Writing packet: type=0x%02x, total length=%d", p.Type, len(data))

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

// convertModemStatusToByte converts ModemStatusBits to byte
func convertModemStatusToByte(status *serial.ModemStatusBits) byte {
	var result byte
	if status.CTS {
		result |= config.FlowControlCTS
	}
	if status.DSR {
		result |= config.FlowControlDSR
	}
	if !status.DCD {
		result |= config.FlowControlDCD
	}
	if status.RI {
		result |= config.FlowControlRI
	}
	return result
}

// setModemControlLines sets modem control lines based on flow control status
func setModemControlLines(port serial.Port, status byte) error {
	// Set DTR based on DCD status
	if err := port.SetDTR(status&config.FlowControlDCD != 0); err != nil {
		return fmt.Errorf("failed to set DTR: %v", err)
	}
	// Set RTS based on CTS status
	if err := port.SetRTS(status&config.FlowControlCTS != 0); err != nil {
		return fmt.Errorf("failed to set RTS: %v", err)
	}
	return nil
}

// Handler handles data forwarding between serial port and network connection
func Handler(ctx context.Context, conn net.Conn, serialConfig *config.SerialConfig) error {
	// Open serial port
	serialMode, err := serialConfig.ToSerialMode()
	if err != nil {
		return fmt.Errorf("failed to convert serial mode: %v", err)
	}

	serialHandler, err := serial.Open(serialConfig.Address, serialMode)
	if err != nil {
		return fmt.Errorf("failed to open serial port: %v", err)
	}
	defer serialHandler.Close()
	log.Printf("Serial port opened")

	// Data forwarding
	ctxConn, cancelCtxConn := context.WithCancel(ctx)
	defer cancelCtxConn()

	// Network -> Serial
	go func() {
		defer cancelCtxConn()
		for {
			packet, err := readPacket(conn)
			if err != nil {
				log.Printf("Error reading packet: %v", err)
				return
			}

			switch packet.Type {
			case config.PacketTypeData:
				log.Printf("Writing %d bytes to serial port", len(packet.Data))
				if _, err := serialHandler.Write(packet.Data); err != nil {
					log.Printf("Error writing to serial port: %v", err)
					return
				}
				log.Printf("Successfully wrote %d bytes to serial port", len(packet.Data))
			case config.PacketTypeFlow:
				if len(packet.Data) < 1 {
					log.Printf("Invalid flow control packet")
					continue
				}
				// 设置串口流控状态
				log.Printf("Received flow control status: 0x%02x", packet.Data[0])
				if err := setModemControlLines(serialHandler, packet.Data[0]); err != nil {
					log.Printf("Error setting modem control lines: %v", err)
					continue
				}
				log.Printf("Set modem control lines: RTS=%v,DTR=%v",
					packet.Data[0]&config.FlowControlCTS != 0,
					packet.Data[0]&config.FlowControlDCD != 0)
			default:
				log.Printf("Unknown packet type: 0x%02x", packet.Type)
			}
		}
	}()

	// Serial -> Network
	go func() {
		defer cancelCtxConn()
		buf := make([]byte, maxPacketSize)

		for {
			// 读取串口数据
			n, err := serialHandler.Read(buf)
			if err != nil {
				log.Printf("Error reading from serial port: %v", err)
				return
			}
			log.Printf("Read %d bytes from serial port", n)

			// 发送数据包
			packet := &Packet{
				Type: config.PacketTypeData,
				Data: buf[:n],
			}
			if err := writePacket(conn, packet); err != nil {
				log.Printf("Error writing packet: %v", err)
				return
			}
		}
	}()

	// 专门用于读取modem状态的goroutine
	go func() {
		defer cancelCtxConn()
		lastModemStatus := byte(0)
		ticker := time.NewTicker(flowControlInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctxConn.Done():
				return
			case <-ticker.C:
				modemStatus, err := serialHandler.GetModemStatusBits()
				if err != nil {
					log.Printf("Error getting modem status: %v", err)
					continue
				}

				currentStatus := convertModemStatusToByte(modemStatus)
				if currentStatus != lastModemStatus {
					log.Printf("Modem status changed: CTS=%v, DCD=%v",
						modemStatus.CTS,
						modemStatus.DCD)
					lastModemStatus = currentStatus
					flowPacket := &Packet{
						Type: config.PacketTypeFlow,
						Data: []byte{currentStatus},
					}
					if err := writePacket(conn, flowPacket); err != nil {
						log.Printf("Error writing flow control packet: %v", err)
						return
					}
				}
			}
		}
	}()

	<-ctxConn.Done()
	log.Println("Connection closed")
	return nil
}
