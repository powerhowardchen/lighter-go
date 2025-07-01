package client

import (
	"net/http"
)

type HTTPClient struct {
	client              *http.Client
	endpoint            string
	channelName         string
	fatFingerProtection bool
}

func NewHTTPClient(client *http.Client, baseUrl string) *HTTPClient {
	if baseUrl == "" {
		return nil
	}

	return &HTTPClient{
		client:              client,
		endpoint:            baseUrl,
		channelName:         "",
		fatFingerProtection: true,
	}
}

func (c *HTTPClient) SetFatFingerProtection(enabled bool) {
	c.fatFingerProtection = enabled
}
