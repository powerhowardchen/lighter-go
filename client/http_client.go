package client

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"golang.org/x/net/http2"
)

type HTTPClient struct {
	mu                     sync.Mutex
	proxy                  func(*http.Request) (*url.URL, error)
	proxyIP                string
	client                 *http.Client
	lastConnectAt          time.Time
	endpoint               string
	channelName            string
	fatFingerProtection    bool
	autoKeepAliveCanceller context.CancelFunc
}

func NewHTTPClient(proxy func(*http.Request) (*url.URL, error), proxyIP, baseUrl string) (p *HTTPClient) {
	if baseUrl == `` {
		return nil
	}

	p = &HTTPClient{
		proxy:               proxy,
		proxyIP:             proxyIP,
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

func (p *HTTPClient) ProxyIP() string {
	return p.proxyIP
}
