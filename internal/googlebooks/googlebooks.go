package googlebooks

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/avast/retry-go"
)

type Client struct {
	HTTPClient *http.Client
	APIKey     string
	FetchDelay time.Duration
}

func NewClient(apiKey string) *Client {
	return &Client{
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
		APIKey:     apiKey,
		FetchDelay: 1 * time.Second,
	}
}

type VolumeInfo struct {
	Title         string   `json:"title"`
	Subtitle      string   `json:"subtitle"`
	Authors       []string `json:"authors"`
	Publisher     string   `json:"publisher"`
	PublishedDate string   `json:"publishedDate"`
	Description   string   `json:"description"`
	InfoLink      string   `json:"infoLink"`
	ImageLinks    struct {
		Thumbnail string `json:"thumbnail"`
	} `json:"imageLinks"`
}

type GoogleBooksResponse struct {
	Items []struct {
		VolumeInfo VolumeInfo `json:"volumeInfo"`
	} `json:"items"`
}

func (c *Client) Fetch(isbn string) (*VolumeInfo, error) {
	time.Sleep(c.FetchDelay) // レート制限準拠のため

	url := fmt.Sprintf("https://www.googleapis.com/books/v1/volumes?q=isbn:%s&key=%s", isbn, c.APIKey)

	var body []byte
	err := retry.Do(
		func() error {
			resp, err := c.HTTPClient.Get(url)
			if err != nil {
				return fmt.Errorf("HTTPリクエスト失敗: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("不正なステータスコード: %d", resp.StatusCode)
			}

			body, err = io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("レスポンス読み込み失敗: %w", err)
			}
			return nil
		},
		retry.Attempts(3),
		retry.DelayType(retry.BackOffDelay),
		retry.Delay(2*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("Google Books API リクエストエラー: %w", err)
	}

	var response GoogleBooksResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("JSONパース失敗: %w", err)
	}

	if len(response.Items) == 0 {
		return nil, fmt.Errorf("ISBN %s の書籍が見つかりません", isbn)
	}

	return &response.Items[0].VolumeInfo, nil
}
