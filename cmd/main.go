package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"comserver/config"
	"comserver/network"

	"gopkg.in/yaml.v2"
)

// serialYmlContent reads serial configuration from file or stdin
func serialYmlContent(serialYml string) (serialYmlContent []byte, err error) {
	if serialYml == "-" {
		fmt.Printf("[Config] Loading from stdin\n")
		buffer := &bytes.Buffer{}
		if _, err = io.Copy(buffer, os.Stdin); err == nil {
			serialYmlContent = buffer.Bytes()
		}
	} else {
		serialYmlContent, err = os.ReadFile(serialYml)
	}
	return
}

func main() {
	// Define command line flags
	serialYml := "serial.yml"
	serialYmlFilePath := flag.String("f", "", fmt.Sprintf("Load serial config from file. Default '%v', '-' for stdin", serialYml))
	listenAddr := flag.String("l", "", "Listen address (e.g., 0.0.0.0:8080)")
	listenAddrShort := flag.String("listen", "", "Listen address (e.g., 0.0.0.0:8080)")
	connectAddr := flag.String("c", "", "Connect address (e.g., 192.168.1.100:8080)")
	connectAddrShort := flag.String("connect", "", "Connect address (e.g., 192.168.1.100:8080)")

	flag.Parse()

	// Check address parameters
	var addr string
	if *listenAddr != "" {
		addr = *listenAddr
	} else if *listenAddrShort != "" {
		addr = *listenAddrShort
	} else if *connectAddr != "" {
		addr = *connectAddr
	} else if *connectAddrShort != "" {
		addr = *connectAddrShort
	} else {
		log.Fatal("Must specify either --listen/-l or --connect/-c parameter")
	}

	// Handle config file path
	if serialYmlFilePath != nil && *serialYmlFilePath != "" {
		serialYml = *serialYmlFilePath
	}

	// Read config file
	serialYmlContent, err := serialYmlContent(serialYml)
	if err != nil {
		log.Fatalf("Failed to read serial config: %v", err.Error())
		return
	}

	serialConfig := &config.SerialConfig{}
	if err := yaml.Unmarshal(serialYmlContent, serialConfig); err != nil {
		log.Fatalf("Failed to parse serial config: %v", err.Error())
		return
	}
	log.Printf("Serial config: %#v", serialConfig)

	ctx := context.Background()

	// Choose mode based on parameters
	if *listenAddr != "" || *listenAddrShort != "" {
		// Listen mode
		listener, err := network.ListenAndServe(ctx, addr)
		if err != nil {
			log.Fatalf("Failed to listen on address: %v", err.Error())
		}
		defer listener.Close()
		log.Printf("Listening on %v", listener.Addr())
		for {
			if !network.HandleListener(ctx, listener, serialConfig) {
				break
			}
		}
	} else {
		// Connect mode
		log.Printf("Connecting to %v", addr)
		for {
			if err := network.HandleConnect(ctx, addr, serialConfig); err != nil {
				log.Printf("Connection failed: %v", err.Error())
				// Wait before retry
				select {
				case <-ctx.Done():
					return
				case <-time.After(time.Second * 5):
					continue
				}
			}
		}
	}
}
