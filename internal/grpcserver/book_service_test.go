package grpcserver_test

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	pb "github.com/taiki-umetsu/ndc007-bookpicker/api/v1"
	"github.com/taiki-umetsu/ndc007-bookpicker/internal/database"
	"github.com/taiki-umetsu/ndc007-bookpicker/internal/env"
	"github.com/taiki-umetsu/ndc007-bookpicker/internal/grpcserver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestBookService_GetRandomBooks(t *testing.T) {
	env.Load()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("環境変数 DATABASE_URL が未設定のためテストをスキップします")
	}

	db, err := database.Setup(dsn)
	if err != nil {
		t.Fatalf("DB接続エラー: %v", err)
	}
	defer db.Close()

	setupTestData(t, db)

	addr, stop := setupGRPCServer(t, db)
	defer stop()

	client := setupGRPCClient(t, addr)

	tests := []struct {
		name          string
		requestCount  int32
		expectedCount int
		expectError   bool
		errorContains string
	}{
		{
			name:          "デフォルトカウント（0指定）",
			requestCount:  0,
			expectedCount: 3,
			expectError:   false,
		},
		{
			name:          "通常のリクエスト（2件）",
			requestCount:  2,
			expectedCount: 2,
			expectError:   false,
		},
		{
			name:          "通常のリクエスト（5件）",
			requestCount:  5,
			expectedCount: 5,
			expectError:   false,
		},
		{
			name:          "最大カウント（10件）",
			requestCount:  10,
			expectedCount: 10,
			expectError:   false,
		},
		{
			name:          "カウント上限超過",
			requestCount:  11,
			expectedCount: 0,
			expectError:   true,
			errorContains: "count must be between 1 and 10",
		},
		{
			name:          "負の数指定",
			requestCount:  -1,
			expectedCount: 3, // デフォルト値が使われる
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			resp, err := client.GetRandomBooks(ctx, &pb.RandomBooksRequest{Count: tt.requestCount})

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Len(t, resp.Books, tt.expectedCount)

				for _, book := range resp.Books {
					assert.NotEmpty(t, book.Id)
					assert.NotEmpty(t, book.Title)
					assert.NotEmpty(t, book.Authors)
					assert.NotEmpty(t, book.Publisher)
					assert.NotEmpty(t, book.Isbn)
				}
			}
		})
	}
}

func setupTestData(t *testing.T, db *sql.DB) {
	t.Helper()

	if _, err := db.Exec(`TRUNCATE books`); err != nil {
		t.Fatalf("TRUNCATE失敗: %v", err)
	}

	insertQuery := `
		INSERT INTO books (
			isbn, title, subtitle, authors, publisher,
			published_date, description, book_url, image_url
		) VALUES 
		('9784003101018', '吾輩は猫である', '', '夏目漱石', '岩波書店',
		 '1905-01-01', '猫の視点から人間社会を描いた名作',
		 'https://example.com/neko', 'https://example.com/neko.jpg'),
		('9784003101025', '坊っちゃん', '', '夏目漱石', '岩波書店',
		 '1906-01-01', '正義感の強い青年教師の物語',
		 'https://example.com/botchan', 'https://example.com/botchan.jpg'),
		('9784003101032', 'こころ', '', '夏目漱石', '岩波書店',
		 '1914-01-01', '友情と愛情の葛藤を描いた心理小説',
		 'https://example.com/kokoro', 'https://example.com/kokoro.jpg'),
		('9784003101049', '羅生門', '', '芥川龍之介', '岩波書店',
		 '1915-01-01', '人間のエゴイズムを描いた短編小説',
		 'https://example.com/rashomon', 'https://example.com/rashomon.jpg'),
		('9784003101056', '蜘蛛の糸', '', '芥川龍之介', '岩波書店',
		 '1918-01-01', '仏教的な慈悲を題材にした寓話',
		 'https://example.com/kumo', 'https://example.com/kumo.jpg'),
		('9784003101063', '人間失格', '', '太宰治', '岩波書店',
		 '1948-01-01', '自己嫌悪と絶望を描いた自伝的小説',
		 'https://example.com/ningen', 'https://example.com/ningen.jpg'),
		('9784003101070', '走れメロス', '', '太宰治', '岩波書店',
		 '1940-01-01', '友情と信頼をテーマにした短編小説',
		 'https://example.com/melos', 'https://example.com/melos.jpg'),
		('9784003101087', '銀河鉄道の夜', '', '宮沢賢治', '岩波書店',
		 '1934-01-01', '幻想的な世界観で描かれた友情の物語',
		 'https://example.com/ginga', 'https://example.com/ginga.jpg'),
		('9784003101094', '風の又三郎', '', '宮沢賢治', '岩波書店',
		 '1931-01-01', '転校生を巡る子供たちの物語',
		 'https://example.com/kaze', 'https://example.com/kaze.jpg'),
		('9784003101100', '檸檬', '', '梶井基次郎', '岩波書店',
		 '1925-01-01', '青年の心境を詩的に描いた短編',
		 'https://example.com/lemon', 'https://example.com/lemon.jpg')
	`

	if _, err := db.Exec(insertQuery); err != nil {
		t.Fatalf("テストデータ挿入失敗: %v", err)
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM books").Scan(&count); err != nil {
		t.Fatalf("データ確認失敗: %v", err)
	}
	if count != 10 {
		t.Fatalf("期待されるデータ件数: 10, 実際: %d", count)
	}

	t.Cleanup(func() {
		db.Exec(`TRUNCATE books`)
	})
}

func setupGRPCServer(t *testing.T, db *sql.DB) (addr string, stop func()) {
	t.Helper()

	lis, err := net.Listen("tcp", "localhost:0") // :0 で空いてるポート自動選択
	if err != nil {
		t.Fatalf("リスナー作成失敗: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterBookServiceServer(s, grpcserver.NewBookServiceServer(db))

	go func() {
		if err := s.Serve(lis); err != nil {
			log.Printf("gRPCサーバーエラー: %v", err)
		}
	}()

	return lis.Addr().String(), func() {
		s.Stop()
		lis.Close()
	}
}

func setupGRPCClient(t *testing.T, addr string) pb.BookServiceClient {
	t.Helper()

	target := fmt.Sprintf("dns:///%s", addr)
	conn, err := grpc.NewClient(
		target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("gRPCクライアント作成失敗: %v", err)
	}

	t.Cleanup(func() {
		conn.Close()
	})

	return pb.NewBookServiceClient(conn)
}
