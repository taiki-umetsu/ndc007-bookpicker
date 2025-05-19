package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/taiki-umetsu/ndc007-bookpicker/internal/cinii"
	"github.com/taiki-umetsu/ndc007-bookpicker/internal/database"
	"github.com/taiki-umetsu/ndc007-bookpicker/internal/googlebooks"
	"github.com/taiki-umetsu/ndc007-bookpicker/internal/model/book"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	startTime := time.Now()

	if os.Getenv("GO_ENV") != "production" {
		if err := godotenv.Load(".env.development"); err != nil {
			log.Println("Warning: .env.development ファイルの読み込みに失敗しました:", err)
		}
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("環境変数 DATABASE_URL が未設定です")
	}

	appid := os.Getenv("CINII_APPID")
	if appid == "" {
		log.Fatal("環境変数 CINII_APPID が未設定です")
	}

	gbKey := os.Getenv("GOOGLE_BOOKS_KEY")
	if gbKey == "" {
		log.Fatal("環境変数 GOOGLE_BOOKS_KEY が未設定です")
	}

	db, err := database.Setup(dsn)
	if err != nil {
		log.Fatal("DB接続エラー:", err)
	}
	defer db.Close()

	err = database.CreateTable(db)
	if err != nil {
		log.Fatal("テーブル作成エラー:", err)
	}

	ciniiClient := cinii.NewClient(appid)

	const (
		yearFrom = 2020
		//count    = 10
		count = 1
	)

	// 参考: https://www.ndl.go.jp/jp/data/NDC10code201708.pdf
	ndcList := []string{
		"007",    // 情報学．情報科学
		"007.3*", // 情報と社会：情報政策，情報倫理
		// "007.6",   // データ処理．情報処理
		// "007.609", // データ管理：データセキュリティ，データマイニング
		// "007.61",  // システム分析．システム設計．システム開発
		// "007.63",  // コンピュータシステム．ソフトウェア．ミドルウェア．アプリケーション
		// "007.64",  // コンピュータプログラミング
	}

	var isbns []string
	seen := make(map[string]bool)
	for _, ndc := range ndcList {
		fmt.Printf("\nfetch from CiNii 分類コード: %s\n", ndc)

		fetched, fetchErr := ciniiClient.FetchRandomISBNs(ndc, yearFrom, count)
		if fetchErr != nil {
			log.Printf("❌ %s: ISBN取得失敗: %v", ndc, err)
			continue
		}

		if len(fetched) == 0 {
			fmt.Println("ISBN が見つかりませんでした")
			continue
		}

		for _, isbn := range fetched {
			if !seen[isbn] {
				isbns = append(isbns, isbn)
				seen[isbn] = true
			}
		}
	}
	fmt.Println("isbns:", isbns)

	ctx := context.Background()
	gbClient := googlebooks.NewClient(gbKey)

	var books []*book.Book
	for _, isbn := range isbns {
		fmt.Printf("\nfetch from Google isbn: %s\n", isbn)
		gbInfo, gbErr := gbClient.Fetch(isbn)
		fmt.Println(gbInfo)
		if gbErr != nil {
			fmt.Println("本情報取得失敗:", err)
			continue
		}

		books = append(books, book.NewBook(
			isbn,
			gbInfo.Title,
			gbInfo.Subtitle,
			gbInfo.Authors,
			gbInfo.Publisher,
			gbInfo.PublishedDate,
			gbInfo.Description,
			gbInfo.InfoLink,
			gbInfo.ImageLinks.Thumbnail,
		))
	}

	err = book.TruncateAndBulkInsert(ctx, db, books)
	if err != nil {
		log.Fatalf("❌ 書き込み失敗: %v", err)
	}

	elapsedTime := time.Since(startTime)
	log.Printf("[complete] バッチ処理時間: %s", elapsedTime)
}
