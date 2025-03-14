package network

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"

	"comserver/config"
	"comserver/serial"
)

// ListenAndServe creates a network listener for the given URL
func ListenAndServe(ctx context.Context, url string) (net.Listener, error) {
	urlInfos := strings.Split(url, "://")
	network, addr := "tcp", url
	if len(urlInfos) > 1 {
		network, addr = urlInfos[0], urlInfos[1]
	}
	listenConfig := &net.ListenConfig{}
	return listenConfig.Listen(ctx, network, addr)
}

// HandleListener handles incoming connections in listen mode
func HandleListener(ctx context.Context, listener net.Listener, serialConfig *config.SerialConfig) bool {
	conn, err := listener.Accept()
	if err != nil {
		log.Fatalf("[Network] Failed to accept connection: %v", err.Error())
		return false
	}
	defer conn.Close()
	log.Printf("[Network] New connection from %v", conn.RemoteAddr())

	if err := serial.Handler(ctx, conn, serialConfig); err != nil {
		log.Printf("[Network] Connection handler error: %v", err)
	}
	return true
}

// HandleConnect handles outgoing connections in connect mode
func HandleConnect(ctx context.Context, addr string, serialConfig *config.SerialConfig) error {
	// Parse address
	urlInfos := strings.Split(addr, "://")
	network, address := "tcp", addr
	if len(urlInfos) > 1 {
		network, address = urlInfos[0], urlInfos[1]
	}

	// Establish connection
	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, network, address)
	if err != nil {
		return fmt.Errorf("failed to connect: %v", err)
	}
	defer conn.Close()
	log.Printf("[Network] Connected to %v", conn.RemoteAddr())

	return serial.Handler(ctx, conn, serialConfig)
}
