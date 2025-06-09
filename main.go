package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-redis-sniffer/config"
	"go-redis-sniffer/handler"
)

var cfg = new(config.Config)

func init() {
	flag.IntVar(&cfg.RedisPort, "port", 0, "redis port")
	flag.BoolVar(&cfg.StrictMode, "strict-mode", false, "error exit")
	flag.StringVar(&cfg.Device, "net device", "any", "eth0 lo ...")
	flag.Parse()
	if cfg.RedisPort == 0 {
		fmt.Println("sidecar for redis network monitoring, usage:")
		fmt.Println("./go-redis-sniffer -port=6379")
		flag.PrintDefaults()
		os.Exit(0)
	}
	cfg = config.NewConfig(cfg)
}

func main() {
	// 初始加载配置
	manager := handler.NewManager(cfg)
	manager.Start()
	config.Logger.Println(" Manager started")
	// 优雅关闭处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	// 阻塞接收信号
	sig := <-sigChan
	println()
	config.Logger.Printf("Received signal: %v", sig)
	// 关闭超时保护
	done := make(chan struct{})
	go func() {
		manager.Stop()
		close(done)
	}()
	select {
	case <-done:
		time.Sleep(time.Second)
	case <-time.After(10 * time.Second):
		config.Logger.Print("Shutdown timeout, forcing exit")
	}
	config.Logger.Println(" Manager stopped gracefully")
}
