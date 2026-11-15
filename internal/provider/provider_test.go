package provider

import (
	"sync"
	"testing"

	"github.com/dstryaaa/nhtk1/internal/factory"
)

func TestGetInstance_SamePointer(t *testing.T) {
	a := GetInstance()
	b := GetInstance()
	if a != b {
		t.Error("GetInstance должен возвращать один и тот же указатель")
	}
}

func TestSetGetConfig(t *testing.T) {
	p := GetInstance()
	cfg := factory.Config{
		Discounts: []factory.DiscountConfig{
			{Type: "percent", Name: "тест", Percent: 10},
		},
	}
	p.SetConfig(cfg)
	got := p.GetConfig()
	if len(got.Discounts) != 1 || got.Discounts[0].Name != "тест" {
		t.Errorf("GetConfig вернул не тот конфиг: %+v", got)
	}
}

func TestConcurrentAccess(t *testing.T) {
	p := GetInstance()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			p.SetConfig(factory.Config{})
		}()
		go func() {
			defer wg.Done()
			_ = p.GetConfig()
		}()
	}
	wg.Wait()
	// Если есть гонка — -race поймает. Здесь просто проверяем, что не паникует.
}
