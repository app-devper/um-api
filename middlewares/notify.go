package middlewares

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"net/http"
)

type Response struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

type client struct {
	httpClient *http.Client
}

type configuration struct {
	token       string
	method      string
	endpoint    string
	contentType string
}

func NotifyMassage(endpoint string) (*Response, error) {
	c := newClient()
	res, err := c.notify(context.Background(), endpoint)
	if err != nil {
		logrus.Error(err)
		return res, err
	}
	return res, nil
}

func newClient() *client {
	return &client{httpClient: http.DefaultClient}
}

func (c *client) notify(ctx context.Context, endpoint string) (*Response, error) {
	configuration := c.createConfiguration(endpoint)
	req, err := http.NewRequestWithContext(ctx, configuration.method, configuration.endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to new request: %w", err)
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to notify: %w", err)
	}
	nResp := &Response{}
	err = json.NewDecoder(res.Body).Decode(nResp)
	if err != nil {
		return nil, fmt.Errorf("failed to decode notification response: %w", err)
	}

	if res.StatusCode == http.StatusUnauthorized {
		return nResp, errors.New("invalid access token")
	}

	if res.StatusCode != http.StatusOK {
		return nResp, errors.New(nResp.Message)
	}
	return nResp, nil
}

func (c *client) createConfiguration(endpoint string) configuration {
	return configuration{
		endpoint:    endpoint,
		method:      "GET",
		contentType: "application/json",
	}
}
