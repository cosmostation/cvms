// this client can be used to mock the client.Client interface
// It will implement all the methods and just returns what you want it to return
package client

import (
	"context"

	"github.com/sirupsen/logrus"
)

type MockClient struct {
	endpoint      string
	returnMessage map[string][]byte
}

func (m *MockClient) SetLogger(logger *logrus.Logger) Client {
	return m
}

func (m *MockClient) SetEndpoint(endpoint string) Client {
	m.endpoint = endpoint
	return m
}

func (m *MockClient) SetHeaders(headers map[string]string) {
	// do nothing
}

func (m *MockClient) GetEndpoint() (string, error) {
	return m.endpoint, nil
}

func (m *MockClient) Get(ctx context.Context, uri string) ([]byte, error) {
	return m.returnMessage[uri], nil
}

func (m *MockClient) Post(ctx context.Context, uri string, body ...[]byte) ([]byte, error) {
	return m.returnMessage[uri], nil
}

func NewMockClient(endpoint string, returnMessage map[string][]byte) *MockClient {
	return &MockClient{
		endpoint:      endpoint,
		returnMessage: returnMessage,
	}
}
