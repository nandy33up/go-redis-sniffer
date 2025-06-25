package config

import (
	"fmt"
	"log"
	"os"

	"github.com/google/gopacket/layers"
	"github.com/panjf2000/ants/v2"
)

var (
	Logger = log.New(os.Stdout, "[REDIS] ", log.LstdFlags)
	GoPool *ants.Pool
)

type Config struct {
	RedisPort      layers.TCPPort //
	StrictMode     bool           // "strict" or "loose"
	Device         string         //
	SnapshotLen    int32          //
	BPFFilter      string         //
	PoolSize       int            //
	PacketChanSize int            //
}

func NewConfig(cfgs ...*Config) *Config {
	cfg := &Config{
		SnapshotLen:    65535,
		BPFFilter:      "tcp port 6379",
		PoolSize:       16,
		PacketChanSize: 1e4,
	}
	if len(cfgs) > 0 {
		cfg.RedisPort = cfgs[0].RedisPort
		cfg.StrictMode = cfgs[0].StrictMode
		cfg.Device = cfgs[0].Device
		cfg.BPFFilter = fmt.Sprintf("tcp port %d", cfg.RedisPort)
	}
	pool, err := ants.NewPool(cfg.PoolSize)
	if err != nil {
		Logger.Fatal(err)
	}
	GoPool = pool
	return cfg
}
