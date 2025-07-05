package client

import (
	"context"
	"golang.org/x/net/http2"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type HTTPClient struct {
	mu                     sync.Mutex
	proxy                  func(*http.Request) (*url.URL, error)
	client                 *http.Client
	lastConnectAt          time.Time
	endpoint               string
	channelName            string
	fatFingerProtection    bool
	autoKeepAliveCanceller context.CancelFunc
}

func NewHTTPClient(proxy func(*http.Request) (*url.URL, error), baseUrl string) (p *HTTPClient) {
	if baseUrl == `` {
		return nil
	}

	p = &HTTPClient{
		proxy:               proxy,
		endpoint:            baseUrl,
		channelName:         "",
		fatFingerProtection: true,
	}

	p.prepare()

	return
}

func (p *HTTPClient) SetFatFingerProtection(enabled bool) {
	p.fatFingerProtection = enabled
}

func (p *HTTPClient) prepare() {
	dialer := &net.Dialer{
		KeepAlive: 120 * time.Second,
	}

	transport := &http.Transport{
		Proxy:               p.proxy,
		DialContext:         (dialer).DialContext,
		MaxIdleConns:        50,
		MaxIdleConnsPerHost: 50,
		MaxConnsPerHost:     50,
		IdleConnTimeout:     120 * time.Second,
		TLSHandshakeTimeout: 5 * time.Second,
		DisableKeepAlives:   false,
	}

	_ = http2.ConfigureTransport(transport)

	p.client = &http.Client{
		Transport: transport,
		Timeout:   20 * time.Second,
	}
}

func (p *HTTPClient) KeepAliveCancel() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.autoKeepAliveCanceller != nil {
		p.autoKeepAliveCanceller()
		p.autoKeepAliveCanceller = nil
	}
}

func (p *HTTPClient) KeepAliveRun() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.autoKeepAliveCanceller != nil {
		return
	}

	var ctx context.Context

	ctx, p.autoKeepAliveCanceller = context.WithCancel(context.Background())

	touch := func(now time.Time) {
		p.mu.Lock()
		defer p.mu.Unlock()
		if now.Sub(p.lastConnectAt) >= time.Minute {
			if req, err := http.NewRequest(`HEAD`,
				p.endpoint+`/api/v1/sendTx`, nil); err == nil {
				_, _ = p.client.Do(req)
				p.lastConnectAt = now
			}
		}
	}

	go func() {
		touch(time.Now())
		t := time.NewTicker(10 * time.Second)
		for {
			select {
			case n := <-t.C:
				go touch(n)
			case <-ctx.Done():
				return
			}
		}
	}()
}
