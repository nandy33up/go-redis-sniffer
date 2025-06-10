package handler

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"time"

	"go-redis-sniffer/config"

	"github.com/allegro/bigcache/v3"
	"github.com/bytedance/sonic"
)

var json = sonic.ConfigFastest

type Consumer struct {
	config   *config.Config
	stopChan chan struct{}
	cacher   *bigcache.BigCache
	ticker   *time.Ticker
}

func NewConsumer(cfg *config.Config) *Consumer {
	d := 2 * time.Second
	cache, err := bigcache.New(context.Background(), bigcache.DefaultConfig(d))
	if err != nil {
		config.Logger.Fatalf("Failed to create cache: %v", err)
	}
	return &Consumer{
		config:   cfg,
		stopChan: make(chan struct{}),
		cacher:   cache,
		ticker:   time.NewTicker(d),
	}
}

func (c *Consumer) Start(packets <-chan *Packet) {
	config.Logger.Println("Consumer started")
	for {
		select {
		case packet := <-packets:
			if len(packet.Payload) > 0 {
				if err := c.processPacket(packet); err != nil && c.config.StrictMode {
					config.Logger.Fatalf("Error processing packet: %v", err)
				}
			}
		case <-c.ticker.C:
			config.Logger.Println("Consumer cacher size:", c.cacher.Len())
		case <-c.stopChan:
			config.Logger.Println("Consumer stopped")
			return
		}
	}
}

func (c *Consumer) Stop() {
	close(c.stopChan)
}

func (c *Consumer) processPacket(packet *Packet) error {
	cmd := &Command{
		Timestamp:  packet.Timestamp,
		Src:        fmt.Appendf(nil, "%s:%d", packet.SrcIP, packet.SrcPort),
		Dst:        fmt.Appendf(nil, "%s:%d", packet.DstIP, packet.DstPort),
		RawCommand: bytes.TrimSpace(packet.Payload),
	}
	if err := cmd.parse(); err != nil {
		return err
	}
	if packet.IsReq {
		key := fmt.Sprintf("%s%s%d", cmd.Src, cmd.Dst, packet.Seq)
		val, _ := json.Marshal(cmd)
		if err := c.cacher.Set(key, val); err != nil {
			return err
		}
	} else {
		key := fmt.Sprintf("%s%s%d", cmd.Dst, cmd.Src, packet.Seq)
		if val, err := c.cacher.Get(key); err == nil {
			var req *Command
			json.Unmarshal(val, &req)
			config.Logger.Printf("Client req %s %s %s | %d | %s\n", req.Src, "->", req.Dst, packet.Seq, req.command())
			c.cacher.Delete(key)
		}
		config.Logger.Printf("Server rsp %s %s %s | %d | %s\n", cmd.Dst, "<-", cmd.Src, packet.Seq, cmd.command())
	}
	return nil
}

type Command struct {
	Timestamp  time.Time
	Src        []byte
	Dst        []byte
	RawCommand []byte
	Args       [][]byte
}

func (cmd *Command) parse() error {
	defer func() error {
		if r := recover(); r != nil {
			return fmt.Errorf("invalid RESP format")
		}
		return nil
	}()
	// RESP协议格式
	if cmd.RawCommand[0] == '*' {
		return cmd.parseRESPProtocol()
	}
	// 行内命令格式
	return cmd.parseLineProtocol()
}

func (cmd *Command) parseRESPProtocol() error {
	lens := []byte{}
	args := bytes.Split(cmd.RawCommand, []byte("\r\n"))
	// 参数检查
	lens, args = args[0], args[1:]
	n, err := strconv.Atoi(string(lens[1:]))
	if err != nil {
		return fmt.Errorf("invalid RESP format, lens: %s", string(lens))
	}
	if n == 0 {
		return nil
	}
	// 参数整理
	cmd.Args = make([][]byte, 0, n/2)
	for _, arg := range args {
		if !bytes.HasPrefix(arg, []byte("$")) {
			cmd.Args = append(cmd.Args, arg)
		}
	}
	return nil
}

func (cmd *Command) parseLineProtocol() error {
	cmd.Args = bytes.Split(cmd.RawCommand, []byte("\r\n"))
	return nil
}

// 完整命令
func (cmd *Command) command() []byte {
	return bytes.Join(cmd.Args, []byte(" "))
}
