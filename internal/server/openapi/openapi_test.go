package openapi_test

import (
	"embed"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers/gorillamux"

	"github.com/taiki-umetsu/ndc007-bookpicker/internal/database"
	"github.com/taiki-umetsu/ndc007-bookpicker/internal/env"
	"github.com/taiki-umetsu/ndc007-bookpicker/internal/server"
)

//go:embed openapi.yaml
var openapiSpec []byte

//go:embed schemas/*
var schemaFS embed.FS

func TestBooksRandomEndpoint(t *testing.T) {
	env.Load()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("環境変数 DATABASE_URL が未設定です")
	}

	db, err := database.Setup(dsn)
	if err != nil {
		t.Fatalf("DB接続エラー: %v", err)
	}
	t.Cleanup(func() {
		db.Exec(`TRUNCATE books`)
		db.Close()
	})

	if _, err = db.Exec(`TRUNCATE books`); err != nil {
		t.Fatalf("TRUNCATE失敗: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO books (
			isbn, title, subtitle, authors, publisher,
			published_date, description, book_url, image_url
		) VALUES 
		('9784003101018', '吾輩は猫である', '夏目漱石作品集', '夏目漱石', '岩波書店',
		 '1905-01-01', '名前のない猫の視点で描かれる風刺的な小説',
		 'https://example.com/neko', 'https://example.com/neko.jpg'),
		('9784003101025', 'こころ', '', '夏目漱石', '岩波書店',
		 '1914-01-01', '先生と私の関係を描く心理小説',
		 'https://example.com/kokoro', 'https://example.com/kokoro.jpg')
	`)
	if err != nil {
		t.Fatalf("データ挿入失敗: %v", err)
	}

	handler := server.NewHandler(db)
	router := server.NewRouter(handler)
	ts := httptest.NewServer(router)
	defer ts.Close()

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	loader.ReadFromURIFunc = func(loader *openapi3.Loader, uri *url.URL) ([]byte, error) {
		return schemaFS.ReadFile(uri.Path)
	}

	doc, err := loader.LoadFromData(openapiSpec)
	if err != nil {
		t.Fatalf("openapi.yaml パース失敗: %v", err)
	}

	if err = doc.Validate(loader.Context); err != nil {
		t.Fatalf("OpenAPI仕様バリデーション失敗: %v", err)
	}

	doc.Servers = []*openapi3.Server{{URL: ts.URL}}

	routerSpec, err := gorillamux.NewRouter(doc)
	if err != nil {
		t.Fatalf("OpenAPIルーター作成失敗: %v", err)
	}

	testCases := []struct {
		name        string
		url         string
		expectCode  int
		description string
	}{
		{
			name:        "デフォルトパラメータ",
			url:         "/api/v1/books/random",
			expectCode:  http.StatusOK,
			description: "countパラメータなし（デフォルト値3）",
		},
		{
			name:        "count=1",
			url:         "/api/v1/books/random?count=1",
			expectCode:  http.StatusOK,
			description: "count=1を指定",
		},
		{
			name:        "count=2",
			url:         "/api/v1/books/random?count=2",
			expectCode:  http.StatusOK,
			description: "count=2を指定",
		},
		{
			name:        "count=0（無効な値）",
			url:         "/api/v1/books/random?count=0",
			expectCode:  http.StatusBadRequest,
			description: "最小値以下のcount",
		},
		{
			name:        "count=11（無効な値）",
			url:         "/api/v1/books/random?count=11",
			expectCode:  http.StatusBadRequest,
			description: "最大値以上のcount",
		},
		{
			name:        "count=invalid（文字列）",
			url:         "/api/v1/books/random?count=invalid",
			expectCode:  http.StatusBadRequest,
			description: "数値以外のcount",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", ts.URL+tc.url, nil)
			if err != nil {
				t.Fatalf("リクエスト作成失敗: %v", err)
			}
			req.Header.Set("Accept", "application/json")

			res, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("リクエスト失敗: %v", err)
			}
			defer res.Body.Close()

			bodyBytes, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatalf("レスポンス読み取り失敗: %v", err)
			}

			if res.StatusCode != tc.expectCode {
				t.Errorf("ステータスコード不一致: got %d, want %d", res.StatusCode, tc.expectCode)
				t.Logf("レスポンスボディ: %s", string(bodyBytes))
				return
			}

			route, pathParams, err := routerSpec.FindRoute(req)
			if err != nil {
				t.Fatalf("OpenAPIルート解決失敗: %v", err)
			}

			input := &openapi3filter.ResponseValidationInput{
				RequestValidationInput: &openapi3filter.RequestValidationInput{
					Request:    req,
					PathParams: pathParams,
					Route:      route,
				},
				Status: res.StatusCode,
				Header: res.Header,
				Body:   io.NopCloser(strings.NewReader(string(bodyBytes))),
			}

			if err := openapi3filter.ValidateResponse(loader.Context, input); err != nil {
				t.Errorf("OpenAPIレスポンスバリデーション失敗: %v", err)
				t.Logf("レスポンスボディ: %s", string(bodyBytes))
			}

			t.Logf("✓ %s: OpenAPI準拠を確認", tc.description)
		})
	}
}
