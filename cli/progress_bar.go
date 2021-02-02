package cli

import (
	"sync"
	"time"

	"github.com/gosuri/uiprogress"
)

type Progress struct {
	p        *uiprogress.Progress
	interval time.Duration
	Bars     []ProgressBar
}

type ProgressBar interface {
	Finish()
	Cancel()
}

func NewProgress() *Progress {
	p := uiprogress.New()
	p.Start()
	return &Progress{p: p, interval: time.Millisecond * 20}
}

func WithProgress(name string, duration time.Duration, fn func()) {
	progress := NewProgress()
	bar := progress.NewBar(name, duration)
	defer func() {
		bar.Finish()
		progress.Stop()
	}()
	fn()
}

func (p *Progress) NewBar(name string, duration time.Duration) ProgressBar {
	bar := p.p.AddBar(int(duration / p.interval))
	bar.AppendCompleted()
	bar.PrependElapsed()
	if name != "" {
		bar.PrependFunc(func(b *uiprogress.Bar) string {
			return name
		})
	}
	stopc := make(chan struct{}, 1)
	go func() {
		for bar.Incr() {
			select {
			case <-time.After(p.interval):
			case <-stopc:
				return
			}
		}
	}()
	bs := createBarStub(func(s bool) {
		close(stopc)
		if s {
			bar.Set(bar.Total)
		}
	})
	p.Bars = append(p.Bars, bs)
	return bs
}

func (b *Progress) Stop() {
	b.p.Stop()
}

type pBar struct {
	once   *sync.Once
	stopFn func(success bool)
}

func createBarStub(fn func(bool)) *pBar {
	return &pBar{stopFn: fn, once: new(sync.Once)}
}

func (p *pBar) Finish() {
	p.once.Do(func() {
		p.stopFn(true)
	})
}

func (p *pBar) Cancel() {
	p.once.Do(func() {
		p.stopFn(false)
	})
}
