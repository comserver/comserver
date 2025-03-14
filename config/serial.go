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

// SerialConfig represents serial port configuration
type SerialConfig struct {
	Address   string `yaml:"address"`    // 串口地址
	BaudRate  int    `yaml:"baudrate"`   // 波特率
	DataBits  int    `yaml:"databits"`   // 数据位
	StopBits  int    `yaml:"stopbits"`   // 停止位
	Parity    string `yaml:"parity"`     // 校验位
	InvertCTS bool   `yaml:"invert_cts"` // 是否反转CTS信号
	InvertDCD bool   `yaml:"invert_dcd"` // 是否反转DCD信号
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
