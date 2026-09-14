package message

import (
	"context"
	"time"
)

type BroadcastWorker struct {
	service  *Service
	interval time.Duration
}

func NewBroadcastWorker(service *Service, interval time.Duration) *BroadcastWorker {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	return &BroadcastWorker{
		service:  service,
		interval: interval,
	}
}

func (w *BroadcastWorker) Start(ctx context.Context) {
	if w == nil || w.service == nil {
		return
	}
	go w.run(ctx)
}

func (w *BroadcastWorker) run(ctx context.Context) {
	w.service.ProcessNextBroadcast(ctx)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.service.ProcessNextBroadcast(ctx)
		}
	}
}
