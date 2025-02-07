package client

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"errors"

	"github.com/go-resty/resty/v2"
	"github.com/sirupsen/logrus"
)

const (
	retryCount               = 10
	retryWaitTimeDuration    = 10 * time.Millisecond
	retryMaxWaitTimeDuration = 3 * time.Second
)

type RestyClient struct {
	client   *resty.Client
	endpoint string
	logger   *logrus.Logger
}

func (rc *RestyClient) SetLogger(logger *logrus.Logger) Client {
	rc.logger = logger
	rc.client.SetLogger(logger)
	return rc
}

func (rc *RestyClient) SetEndpoint(endpoint string) Client {
	rc.endpoint = endpoint
	rc.client.SetBaseURL(endpoint)

	return rc
}

func (rc *RestyClient) GetEndpoint() (string, error) {
	if rc.endpoint == "" {
		return "", errors.New("endpoint is not set")
	}
	return rc.endpoint, nil
}

func (rc *RestyClient) SetHeaders(header map[string]string) {
	rc.client.SetHeaders(header)
}

func NewRestyClient() *RestyClient {
	rc := &RestyClient{}
	rc.client = resty.New().
		SetRetryCount(retryCount).
		SetRetryWaitTime(retryMaxWaitTimeDuration).
		SetRetryMaxWaitTime(retryMaxWaitTimeDuration).
		SetHeader("Content-Type", "application/json")

	return rc
}

func (rc *RestyClient) Get(context context.Context, uri string) ([]byte, error) {
	resp, err := rc.client.R().
		SetContext(context).
		Get(rc.endpoint + uri)
	if err != nil {
		rc.logger.Errorf("GET request failed: %s", err.Error())
		return nil, err
	}
	rc.logger.Debugf("Received %d status code from %s", resp.StatusCode(), resp.Request.URL)
	if resp.StatusCode() != http.StatusOK {
		rc.logger.Errorf("GET request failed with status code: %d", resp.StatusCode())
		return nil, fmt.Errorf("api error: got %d code from %s", resp.StatusCode(), resp.Request.URL)
	}
	return resp.Body(), nil
}

func (rc *RestyClient) Post(context context.Context, uri string, body ...[]byte) ([]byte, error) {
	resp, err := rc.client.R().
		SetHeader("Content-Type", "application/json").
		SetContext(context).
		SetBody(body).
		Post(rc.endpoint + uri)
	if err != nil {
		rc.logger.Errorf("POST request failed: %s", err.Error())
		return nil, err
	}
	rc.logger.Debugf("Received %d status code from %s", resp.StatusCode(), resp.Request.URL)
	// This ensures that the response status code is within the 200-202 range,
	// which indicates a successful POST request.
	if resp.StatusCode() <= http.StatusOK && resp.StatusCode() <= http.StatusAccepted {
		rc.logger.Errorf("POST request failed with status code: %d", resp.StatusCode())
		return nil, fmt.Errorf("api error: got %d code from %s", resp.StatusCode(), resp.Request.URL)
	}
	return resp.Body(), nil
}
