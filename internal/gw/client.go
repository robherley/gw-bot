package gw

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

const (
	DefaultBaseURL = "https://buyerapi.shopgoodwill.com"
)

type Client struct {
	*http.Client
	baseURL string
}

func New() *Client {
	return &Client{
		Client:  http.DefaultClient,
		baseURL: DefaultBaseURL,
	}
}

func (c *Client) Search(ctx context.Context, term string, opts ...SearchOption) ([]Item, error) {
	query, err := NewSearchQuery(term, opts...)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/Search/ItemListing", bytes.NewBuffer(query))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logRequestError(ctx, req, resp)
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	type response struct {
		SearchResults struct {
			Items []Item `json:"items"`
		} `json:"searchResults"`
	}

	var res response
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	return res.SearchResults.Items, nil
}

func (c *Client) FindItem(ctx context.Context, id int64) (*Item, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/api/ItemDetail/GetItemDetailModelByItemId/%d", c.baseURL, id), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logRequestError(ctx, req, resp)
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var item Item
	if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
		return nil, err
	}

	return &item, nil
}

func logRequestError(ctx context.Context, req *http.Request, resp *http.Response) {
	body, _ := io.ReadAll(resp.Body)
	slog.ErrorContext(ctx, "unexpected status code",
		"body", string(body),
		"code", resp.StatusCode,
		"url", req.URL.String(),
		"method", req.Method,
	)
}
