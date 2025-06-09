package handler

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"go-redis-sniffer/config"
)

type Consumer struct {
	config   *config.Config
	stopChan chan struct{}
}

func NewConsumer(cfg *config.Config) *Consumer {
	return &Consumer{
		config:   cfg,
		stopChan: make(chan struct{}),
	}
}

func (c *Consumer) Start(packetChan <-chan *Packet) {
	config.Logger.Println("Consumer started")
	for {
		select {
		case <-c.stopChan:
			config.Logger.Println("Consumer stopped")
			return
		case packet := <-packetChan:
			if len(packet.Payload) > 0 {
				c.processPacket(packet)
			}
		}
	}
}

func (c *Consumer) Stop() {
	close(c.stopChan)
}

func (c *Consumer) processPacket(packet *Packet) {
	redisCmd := RedisCommand{
		Timestamp:  packet.Timestamp,
		Client:     fmt.Sprintf("%s:%d", packet.SrcIP, packet.SrcPort),
		Server:     fmt.Sprintf("%s:%d", packet.DstIP, packet.DstPort),
		Direction:  packet.Direction,
		RawCommand: bytes.TrimSpace(packet.Payload),
	}
	if redisCmd.RawCommand[0] == '*' {
		// RESP协议格式
		if err := redisCmd.parseRESPProtocol(); err != nil && c.config.StrictMode {
			config.Logger.Fatal(err)
		}
	} else {
		// 行内命令格式
		if err := redisCmd.parseInlineProtocol(); err != nil && c.config.StrictMode {
			config.Logger.Fatal(err)
		}
	}
	config.Logger.Printf("%s %s %s | %s\n", redisCmd.Client, redisCmd.Direction, redisCmd.Server, redisCmd.String())
}

// 解析后的Redis命令
type RedisCommand struct {
	Timestamp  time.Time
	Client     string
	Server     string
	Direction  string
	RawCommand []byte
	Args       []string
	IsInline   bool
}

func (cmd *RedisCommand) parseRESPProtocol() error {
	defer func() {
		if r := recover(); r != nil {
			println("Error parsing RESP command")
			println(string(cmd.RawCommand))
			os.Exit(1)
		}
	}()
	count := []byte{}
	parts := bytes.Split(cmd.RawCommand, []byte("\r\n"))
	// 参数检查
	count, parts = parts[0], parts[1:]
	n, err := strconv.Atoi(string(count[1:]))
	if err != nil {
		return fmt.Errorf("invalid RESP format, count: %s", string(count))
	}
	if n == 0 {
		return nil
	}
	if len(parts) < 2 {
		return fmt.Errorf("invalid RESP format, parts: %s", string(cmd.RawCommand))
	}
	// 参数整理
	args := make([]string, 0, n/2)
	for _, part := range parts {
		if len(part) > 0 && part[0] == '$' {
			continue
		}
		args = append(args, string(part))
	}
	cmd.Args = args
	return nil
}

func (cmd *RedisCommand) parseInlineProtocol() error {
	cmd.IsInline = true
	cmd.Args = strings.Fields(string(cmd.RawCommand))
	return nil
}

// 完整命令
func (cmd *RedisCommand) String() string {
	return strings.Join(cmd.Args, " ")
}
