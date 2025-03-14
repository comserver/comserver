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
	log.Printf("[Serial] Port %s opened with baudrate %d", serialConfig.Address, serialConfig.BaudRate)

	// Data forwarding
	ctxConn, cancelCtxConn := context.WithCancel(ctx)
	defer cancelCtxConn()

	// Network -> Serial
	go func() {
		defer cancelCtxConn()
		for {
			p, err := packet.Read(conn)
			if err != nil {
				log.Printf("[Network] Error reading packet: %v", err)
				return
			}

			switch p.Type {
			case config.PacketTypeData:
				if _, err := serialHandler.Write(p.Data); err != nil {
					log.Printf("[Serial] Error writing data: %v", err)
					return
				}
				log.Printf("[Network->Serial] Forwarded %d bytes", len(p.Data))
			case config.PacketTypeFlow:
				if len(p.Data) < 1 {
					log.Printf("[Network] Invalid flow control packet")
					continue
				}
				// 设置串口流控状态
				if err := setModemControlLines(serialHandler, p.Data[0]); err != nil {
					log.Printf("[Serial] Error setting flow control: %v", err)
					continue
				}
				log.Printf("[Network->Serial] Flow control: RTS=%v, DTR=%v",
					p.Data[0]&config.FlowControlCTS != 0,
					p.Data[0]&config.FlowControlDCD != 0)
			default:
				log.Printf("[Network] Unknown packet type: 0x%02x", p.Type)
			}
		}
	}()

	// Serial -> Network
	go func() {
		defer cancelCtxConn()
		buf := make([]byte, 0xFF-1)

		for {
			n, err := serialHandler.Read(buf)
			if err != nil {
				log.Printf("[Serial] Error reading data: %v", err)
				return
			}

			p := &packet.Packet{
				Type: config.PacketTypeData,
				Data: buf[:n],
			}
			if err := packet.Write(conn, p); err != nil {
				log.Printf("[Network] Error writing packet: %v", err)
				return
			}
			log.Printf("[Serial->Network] Forwarded %d bytes", n)
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
					log.Printf("[Serial] Error reading modem status: %v", err)
					continue
				}

				currentStatus := convertModemStatusToByte(modemStatus, serialConfig)
				if currentStatus != lastModemStatus {
					log.Printf("[Serial] Modem status changed: CTS=%v, DSR=%v, DCD=%v, RI=%v",
						modemStatus.CTS,
						modemStatus.DSR,
						modemStatus.DCD,
						modemStatus.RI)
					lastModemStatus = currentStatus
					p := &packet.Packet{
						Type: config.PacketTypeFlow,
						Data: []byte{currentStatus},
					}
					if err := packet.Write(conn, p); err != nil {
						log.Printf("[Network] Error writing flow control packet: %v", err)
						return
					}
				}
			}
		}
	}()

	<-ctxConn.Done()
	log.Println("[System] Connection closed")
	return nil
}
