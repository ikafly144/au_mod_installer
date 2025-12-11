package rest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ikafly144/au_mod_installer/common/rest"
)

type clientImpl struct {
	BaseURL    string
	UserAgent  string
	HTTPClient *http.Client
}

var _ Client = (*clientImpl)(nil)

type config func(*clientImpl)

func WithHTTPClient(httpClient *http.Client) config {
	return func(c *clientImpl) {
		c.HTTPClient = httpClient
	}
}

func NewClient(baseURL string, configs ...config) *clientImpl {
	client := &clientImpl{
		BaseURL:    baseURL,
		HTTPClient: &http.Client{},
	}
	for _, config := range configs {
		config(client)
	}
	return client
}

func (c *clientImpl) do(endpoint *rest.CompiledEndpoint, rqBody any, rsBody any, tries int) error {
	var (
		rawRequestBody []byte
		err            error
		contentType    string
	)
	if rqBody != nil {
		switch v := rqBody.(type) {
		case []byte:
			contentType = "application/octet-stream"
			rawRequestBody = v
		default:
			contentType = "application/json"
			if rawRequestBody, err = json.Marshal(rqBody); err != nil {
				return err
			}
		}
	}

	rq, err := http.NewRequest(endpoint.Endpoint.Method, c.BaseURL+endpoint.URL, bytes.NewReader(rawRequestBody))
	if err != nil {
		return err
	}

	rq.Header.Set("User-Agent", c.UserAgent)
	if contentType != "" {
		rq.Header.Set("Content-Type", contentType)
	}

	resp, err := c.HTTPClient.Do(rq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("request failed with status code %d", resp.StatusCode)
	}
	if rsBody != nil {
		if err := json.NewDecoder(resp.Body).Decode(rsBody); err != nil {
			return err
		}
	}
	return nil
}
