package cinii

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/avast/retry-go"
)

const (
	SortByScore       = 1 // 検索スコア順
	SortByYearAsc     = 2 // 出版年昇順
	SortByYearDesc    = 3 // 出版年降順
	SortByLibraryAsc  = 4 // 所蔵館数昇順
	SortByLibraryDesc = 5 // 所蔵館数降順
)

var sortOptions = []int{
	SortByScore,
	SortByYearAsc,
	SortByYearDesc,
	SortByLibraryAsc,
	SortByLibraryDesc,
}

type Client struct {
	HTTPClient *http.Client
	AppID      string
	FetchDelay time.Duration
	Rand       *rand.Rand
}

func NewClient(appID string) *Client {
	return &Client{
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
		AppID:      appID,
		FetchDelay: 1 * time.Second,
		Rand:       rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

type ciniiResponse struct {
	Graph []struct {
		TotalResults string `json:"opensearch:totalResults"`
		Items        []struct {
			HasPart []struct {
				ID string `json:"@id"`
			} `json:"dcterms:hasPart"`
		} `json:"items"`
	} `json:"@graph"`
}

func (c *Client) fetch(ndc string, count, page, yearFrom, sort int) ([]byte, error) {
	url := fmt.Sprintf(
		"https://ci.nii.ac.jp/books/opensearch/search?format=json&lang=jpn&appid=%s&clas=%s&count=%d&p=%d&year_from=%d&sortorder=%d",
		c.AppID, ndc, count, page, yearFrom, sort,
	)

	time.Sleep(c.FetchDelay) // 負荷分散のため

	var body []byte
	err := retry.Do(
		func() error {
			resp, err := c.HTTPClient.Get(url)
			if err != nil {
				return fmt.Errorf("HTTPリクエスト失敗: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("HTTPステータス: %d", resp.StatusCode)
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
		return nil, fmt.Errorf("CiNii Books API リクエストエラー: %w", err)
	}
	return body, nil
}

func (c *Client) fetchTotalResults(ndc string, yearFrom int) (int, error) {
	raw, err := c.fetch(ndc, 1, 1, yearFrom, 1)
	if err != nil {
		return 0, err
	}

	var cr ciniiResponse
	err = json.Unmarshal(raw, &cr)
	if err != nil {
		return 0, fmt.Errorf("JSONパース失敗: %w", err)
	}

	if len(cr.Graph) == 0 {
		return 0, fmt.Errorf("レスポンスに@graphがありません")
	}

	total, err := strconv.Atoi(cr.Graph[0].TotalResults)
	if err != nil {
		return 0, fmt.Errorf("totalResults のパース失敗: %w", err)
	}

	return total, nil
}

func (c *Client) FetchRandomISBNs(ndc string, yearFrom, count int) ([]string, error) {
	if count <= 0 {
		return nil, fmt.Errorf("count が正の整数ではありません: %d", count)
	}

	total, err := c.fetchTotalResults(ndc, yearFrom)
	if err != nil {
		return nil, fmt.Errorf("検索結果の取得に失敗: %w", err)
	}
	if total <= 0 {
		return nil, fmt.Errorf("totalResults が正の整数ではありません: %d", total)
	}

	maxPage := (total + count - 1) / count
	if maxPage <= 0 {
		return nil, fmt.Errorf("有効なページがありません")
	}

	page := c.Rand.Intn(maxPage) + 1
	sort := sortOptions[c.Rand.Intn(len(sortOptions))]

	raw, err := c.fetch(ndc, count, page, yearFrom, sort)
	if err != nil {
		return nil, err
	}

	var cr ciniiResponse
	if err := json.Unmarshal(raw, &cr); err != nil {
		return nil, fmt.Errorf("JSONパース失敗: %w", err)
	}

	if len(cr.Graph) == 0 || len(cr.Graph[0].Items) == 0 {
		return []string{}, nil
	}

	isbns := make([]string, 0, count)
	for _, itm := range cr.Graph[0].Items {
		for _, p := range itm.HasPart {
			if strings.HasPrefix(p.ID, "urn:isbn:") {
				isbns = append(isbns, strings.TrimPrefix(p.ID, "urn:isbn:"))
			}
		}
	}
	return isbns, nil
}
