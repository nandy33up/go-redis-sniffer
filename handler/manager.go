package handler

import (
	"go-redis-sniffer/config"
)

type Manager struct {
	config   *config.Config
	producer *Producer
	consumer *Consumer
}

func NewManager(cfg *config.Config) *Manager {
	producer := NewProducer(cfg)
	consumer := NewConsumer(cfg)
	return &Manager{
		producer: producer,
		consumer: consumer,
		config:   cfg,
	}
}

func (m *Manager) Start() {
	go m.producer.Start()
	go m.consumer.Start(m.producer.packets)
}

func (m *Manager) Stop() {
	m.producer.Stop()
	m.consumer.Stop()
	config.GoPool.Release()
}
