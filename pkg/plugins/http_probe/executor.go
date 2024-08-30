package httpprobe

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/magicsong/okg-sidecar/pkg/extractor"
	"github.com/magicsong/okg-sidecar/pkg/store"
)

// Executor holds the HTTP client and provides methods for probing
type Executor struct {
	client *http.Client
	store.StorageFactory
}

// NewExecutor creates a new Prober with the provided timeout
func NewExecutor(timeout int, factory store.StorageFactory) *Executor {
	return &Executor{
		client: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
		StorageFactory: factory,
	}
}

// Probe performs the HTTP request based on the provided configuration
func (p *Executor) Probe(config EndpointConfig) error {
	req, err := http.NewRequest(config.Method, config.URL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers
	for key, value := range config.Headers {
		req.Header.Set(key, value)
	}

	// Perform the request
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}
	// Check expected status code
	if resp.StatusCode != config.ExpectedStatusCode {
		return fmt.Errorf("unexpected status code: got %v, expected %v, body: %s", resp.StatusCode, config.ExpectedStatusCode, string(body))
	}

	// Extract data
	data, err := p.extractData(body, config.JSONPathConfig)
	if err != nil {
		return fmt.Errorf("failed to extract data: %v", err)
	}
	// Store data
	if err := p.storeData(data, &config.StorageConfig); err != nil {
		return fmt.Errorf("failed to store data: %v", err)
	}
	return nil
}

func (p *Executor) extractData(data []byte, extractorConfig *store.JSONPathConfig) (interface{}, error) {
	if extractorConfig != nil {
		return extractor.GetDataFromJsonText(string(data), extractorConfig.JSONPath)
	}
	return string(data), nil
}

func (p *Executor) storeData(data interface{}, storeConfig *store.StorageConfig) error {
	return storeConfig.StoreData(p.StorageFactory, data)
}
