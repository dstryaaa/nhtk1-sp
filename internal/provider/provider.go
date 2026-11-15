package provider

import (
	"sync"

	"github.com/dstryaaa/nhtk1/internal/factory"
)

var (
	once     sync.Once
	instance *ConfigProvider
)

// ConfigProvider хранит только конфиг. Состояние заказа сюда не кладём —
// иначе заказы начнут мешать друг другу.
type ConfigProvider struct {
	mu     sync.RWMutex
	config factory.Config
}

func GetInstance() *ConfigProvider {
	once.Do(func() {
		instance = &ConfigProvider{}
	})
	return instance
}

func (p *ConfigProvider) SetConfig(cfg factory.Config) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.config = cfg
}

func (p *ConfigProvider) GetConfig() factory.Config {
	// RLock, а не Lock — читателей много, они не должны блокировать друг друга.
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.config
}
