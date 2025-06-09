package handler

import (
	"net"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"

	"go-redis-sniffer/config"
	"go-redis-sniffer/utils"
)

type Packet struct {
	Timestamp time.Time
	SrcIP     net.IP
	DstIP     net.IP
	SrcPort   layers.TCPPort
	DstPort   layers.TCPPort
	Payload   []byte
	Direction string
}

type Producer struct {
	config   *config.Config
	packets  chan *Packet
	stopChan chan struct{}
	ticker   *time.Ticker
}

func NewProducer(cfg *config.Config) *Producer {
	return &Producer{
		config:   cfg,
		packets:  make(chan *Packet, cfg.PacketChanSize),
		stopChan: make(chan struct{}),
		ticker:   time.NewTicker(10 * time.Second), // 定时器用于定期检查
	}
}

func (p *Producer) Start() {
	handle, err := pcap.OpenLive(p.config.Device, p.config.SnapshotLen, true, pcap.BlockForever)
	if err != nil {
		config.Logger.Fatal(err)
	}
	err = handle.SetBPFFilter(p.config.BPFFilter)
	if err != nil {
		config.Logger.Fatal(err)
	}
	source := gopacket.NewPacketSource(handle, handle.LinkType())
	source.DecodeOptions.Lazy = true
	source.DecodeOptions.NoCopy = true
	config.Logger.Println("Producer started")
	for {
		select {
		case packet := <-source.Packets():
			if packet != nil {
				p.processPacket(packet)
			}
		case <-p.ticker.C:
			config.Logger.Println("Packet queue suze:", len(p.packets))
		case <-p.stopChan:
			config.Logger.Println("Producer stopped")
			return
		}
	}
}

func (p *Producer) Stop() {
	close(p.stopChan)
}

func (p *Producer) processPacket(packet gopacket.Packet) {
	netLayer := packet.NetworkLayer()
	if netLayer == nil {
		return
	}

	transportLayer := packet.TransportLayer()
	if transportLayer == nil {
		return
	}

	tcp, ok := transportLayer.(*layers.TCP)
	if !ok || len(tcp.Payload) == 0 {
		return
	}

	var srcIP, dstIP net.IP
	switch v := netLayer.(type) {
	case *layers.IPv4:
		srcIP, dstIP = v.SrcIP, v.DstIP
	case *layers.IPv6:
		srcIP, dstIP = v.SrcIP, v.DstIP
	default:
		return
	}

	p.packets <- &Packet{
		Timestamp: packet.Metadata().Timestamp,
		SrcIP:     srcIP,
		DstIP:     dstIP,
		SrcPort:   tcp.SrcPort,
		DstPort:   tcp.DstPort,
		Payload:   tcp.Payload,
		Direction: utils.IF(tcp.DstPort == 6379, "->", "<-"),
	}
}

func (p *Producer) PacketChan() <-chan *Packet {
	return p.packets
}
