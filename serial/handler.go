package serial

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"comserver/config"
	"comserver/packet"

	"go.bug.st/serial"
)

const (
	flowControlInterval = 5 * time.Millisecond
)

// convertModemStatusToByte converts ModemStatusBits to byte
func convertModemStatusToByte(status *serial.ModemStatusBits, serialConfig *config.SerialConfig) byte {
	var result byte
	if status.CTS != serialConfig.InvertCTS {
		result |= config.FlowControlCTS
	}
	if status.DSR {
		result |= config.FlowControlDSR
	}
	if status.DCD != serialConfig.InvertDCD {
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
			p, err := packet.Read(conn)
			if err != nil {
				log.Printf("Error reading packet: %v", err)
				return
			}

			switch p.Type {
			case config.PacketTypeData:
				log.Printf("Writing %d bytes to serial port", len(p.Data))
				if _, err := serialHandler.Write(p.Data); err != nil {
					log.Printf("Error writing to serial port: %v", err)
					return
				}
				log.Printf("Successfully wrote %d bytes to serial port", len(p.Data))
			case config.PacketTypeFlow:
				if len(p.Data) < 1 {
					log.Printf("Invalid flow control packet")
					continue
				}
				// 设置串口流控状态
				log.Printf("Received flow control status: 0x%02x", p.Data[0])
				if err := setModemControlLines(serialHandler, p.Data[0]); err != nil {
					log.Printf("Error setting modem control lines: %v", err)
					continue
				}
				log.Printf("Set modem control lines: RTS=%v,DTR=%v",
					p.Data[0]&config.FlowControlCTS != 0,
					p.Data[0]&config.FlowControlDCD != 0)
			default:
				log.Printf("Unknown packet type: 0x%02x", p.Type)
			}
		}
	}()

	// Serial -> Network
	go func() {
		defer cancelCtxConn()
		buf := make([]byte, 0xFF-1)

		for {
			// 读取串口数据
			n, err := serialHandler.Read(buf)
			if err != nil {
				log.Printf("Error reading from serial port: %v", err)
				return
			}
			log.Printf("Read %d bytes from serial port", n)

			// 发送数据包
			p := &packet.Packet{
				Type: config.PacketTypeData,
				Data: buf[:n],
			}
			if err := packet.Write(conn, p); err != nil {
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

				currentStatus := convertModemStatusToByte(modemStatus, serialConfig)
				if currentStatus != lastModemStatus {
					log.Printf("Modem status changed: CTS=%v, DCD=%v",
						modemStatus.CTS,
						modemStatus.DCD)
					lastModemStatus = currentStatus
					p := &packet.Packet{
						Type: config.PacketTypeFlow,
						Data: []byte{currentStatus},
					}
					if err := packet.Write(conn, p); err != nil {
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
