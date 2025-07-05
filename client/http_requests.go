package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/elliottech/lighter-go/types/txtypes"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func (p *HTTPClient) parseResultStatus(respBody []byte) error {
	resultStatus := &ResultCode{}
	if err := json.Unmarshal(respBody, resultStatus); err != nil {
		return err
	}
	if resultStatus.Code != CodeOK {
		return errors.New(resultStatus.Message)
	}
	return nil
}

func (p *HTTPClient) getAndParseL2HTTPResponse(path string, params map[string]any, result interface{}) error {
	u, err := url.Parse(p.endpoint)
	if err != nil {
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	u.Path = path

	q := u.Query()
	for k, v := range params {
		q.Set(k, fmt.Sprintf("%v", v))
	}
	u.RawQuery = q.Encode()

	p.lastConnectAt = time.Now()

	resp, err := p.client.Get(u.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New(string(body))
	}
	if err = p.parseResultStatus(body); err != nil {
		return err
	}
	if err := json.Unmarshal(body, result); err != nil {
		return err
	}
	return nil
}

func (p *HTTPClient) GetNextNonce(accountIndex int64, apiKeyIndex uint8) (int64, error) {
	result := &NextNonce{}
	err := p.getAndParseL2HTTPResponse("api/v1/nextNonce", map[string]any{"account_index": accountIndex, "api_key_index": apiKeyIndex}, result)
	if err != nil {
		return -1, err
	}
	return result.Nonce, nil
}

func (p *HTTPClient) GetApiKey(accountIndex int64, apiKeyIndex uint8) (*AccountApiKeys, error) {
	result := &AccountApiKeys{}
	err := p.getAndParseL2HTTPResponse("api/v1/apikeys", map[string]any{"account_index": accountIndex, "api_key_index": apiKeyIndex}, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (p *HTTPClient) SendRawTx(tx txtypes.TxInfo) (string, error) {
	txType := tx.GetTxType()
	txInfo, err := tx.GetTxInfo()
	if err != nil {
		return "", err
	}

	data := url.Values{"tx_type": {strconv.Itoa(int(txType))}, "tx_info": {txInfo}}

	if p.fatFingerProtection == false {
		data.Add("price_protection", "false")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	req, _ := http.NewRequest("POST", p.endpoint+"/api/v1/sendTx", strings.NewReader(data.Encode()))
	req.Header.Set("Channel-Name", p.channelName)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	p.lastConnectAt = time.Now()

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", errors.New(string(body))
	}
	if err = p.parseResultStatus(body); err != nil {
		return "", err
	}
	res := &TxHash{}
	if err := json.Unmarshal(body, res); err != nil {
		return "", err
	}

	return res.TxHash, nil
}
