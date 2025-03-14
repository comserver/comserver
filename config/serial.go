package config

import (
	"fmt"

	"go.bug.st/serial"
)

const (
	// 数据包类型
	PacketTypeData byte = 0x01 // 数据包
	PacketTypeFlow byte = 0x02 // 流控包

	// 流控状态位
	FlowControlCTS byte = 0x01 // Clear To Send
	FlowControlDSR byte = 0x02 // Data Set Ready
	FlowControlDCD byte = 0x04 // Data Carrier Detect
	FlowControlRI  byte = 0x08 // Ring Indicator
)

// SerialConfig represents the configuration for a serial port connection
type SerialConfig struct {
	Address  string `yaml:"address"`  // Serial port address (e.g., COM1, /dev/ttyUSB0)
	BaudRate int    `yaml:"baudrate"` // Communication speed in bits per second
	StopBits int    `yaml:"stopbits"` // Number of stop bits (1 or 2)
	DataBits int    `yaml:"databits"` // Number of data bits
	Parity   string `yaml:"parity"`   // Parity mode (N: None, E: Even, O: Odd)
}

// ToSerialMode converts SerialConfig to serial.Mode
func (c *SerialConfig) ToSerialMode() (*serial.Mode, error) {
	// Convert stop bits
	var stopBits serial.StopBits
	switch c.StopBits {
	case 1:
		stopBits = serial.OneStopBit
	case 2:
		stopBits = serial.TwoStopBits
	default:
		return nil, fmt.Errorf("unsupported stop bits: %d", c.StopBits)
	}

	// Convert parity
	var parity serial.Parity
	switch c.Parity {
	case "N":
		parity = serial.NoParity
	case "E":
		parity = serial.EvenParity
	case "O":
		parity = serial.OddParity
	default:
		return nil, fmt.Errorf("unsupported parity: %s", c.Parity)
	}

	return &serial.Mode{
		BaudRate: c.BaudRate,
		DataBits: c.DataBits,
		Parity:   parity,
		StopBits: stopBits,
	}, nil
}
