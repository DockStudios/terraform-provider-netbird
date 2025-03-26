package provider

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	BaseUrl     string
	BearerToken string
	AccessToken string
	httpClient  *http.Client
}

func NewClient(baseURL string, bearerToken string, accessToken string) *Client {
	return &Client{
		BaseUrl:     baseURL,
		BearerToken: bearerToken,
		AccessToken: accessToken,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (s *Client) doRequest(req *http.Request) ([]byte, error) {
	if s.BearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.BearerToken)
	}
	if s.AccessToken != "" {
		req.Header.Set("Authorization", "Token "+s.AccessToken)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%s", body)
	}
	return body, nil
}
