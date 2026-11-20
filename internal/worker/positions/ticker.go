package positions

import "time"

type Ticker interface {
	C() <-chan time.Time
	Stop()
}

type SystemTicker struct {
	ticker *time.Ticker
}

func NewSystemTicker(interval time.Duration) *SystemTicker {
	return &SystemTicker{
		ticker: time.NewTicker(interval),
	}
}

func (t *SystemTicker) C() <-chan time.Time {
	return t.ticker.C
}

func (t *SystemTicker) Stop() {
	t.ticker.Stop()
}
