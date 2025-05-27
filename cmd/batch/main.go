package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/taiki-umetsu/ndc007-bookpicker/internal/cinii"
	"github.com/taiki-umetsu/ndc007-bookpicker/internal/database"
	"github.com/taiki-umetsu/ndc007-bookpicker/internal/env"
	"github.com/taiki-umetsu/ndc007-bookpicker/internal/googlebooks"
	"github.com/taiki-umetsu/ndc007-bookpicker/internal/model/book"

	_ "github.com/lib/pq"
)

const (
	ciniiFetchCount     = 10
	bulkInsertChunkSize = 10
	yearFrom            = 2020
)

func main() {
	startTime := time.Now()

	env.Load()

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
	gbClient := googlebooks.NewClient(gbKey)
	ctx := context.Background()

	// 参考: https://www.ndl.go.jp/jp/data/NDC10code201708.pdf
	ndcList := []string{
		"007",     // 情報学．情報科学
		"007.3*",  // 情報と社会：情報政策，情報倫理
		"007.6",   // データ処理．情報処理
		"007.609", // データ管理：データセキュリティ，データマイニング
		"007.61",  // システム分析．システム設計．システム開発
		"007.63",  // コンピュータシステム．ソフトウェア．ミドルウェア．アプリケーション
		"007.64",  // コンピュータプログラミング
	}

	baf := len(ndcList) * ciniiFetchCount
	isbnCh := make(chan string, baf)
	bookCh := make(chan *book.Book, baf)
	errChan := make(chan error)

	tx, txErr := db.BeginTx(ctx, nil)
	if txErr != nil {
		log.Fatalf("トランザクション開始エラー: %v", txErr)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, "TRUNCATE TABLE books")
	if err != nil {
		tx.Rollback()
		log.Fatalf("TRUNCATEエラー: %v", err)
	}
	fmt.Println("books テーブルを TRUNCATE しました")

	// エラーチャネルを監視するゴルーチン
	go func() {
		for err := range errChan {
			log.Println("エラー:", err)
		}
	}()

	// 1. CiNii から ISBN を取得するゴルーチン
	go func() {
		defer close(isbnCh)
		seen := make(map[string]struct{})
		for _, ndc := range ndcList {
			fmt.Printf("\nfetch from CiNii 分類コード: %s\n", ndc)
			isbns, fetchErr := ciniiClient.FetchRandomISBNs(ndc, yearFrom, ciniiFetchCount)
			if fetchErr != nil {
				errChan <- fmt.Errorf("CiNii ISBN 取得エラー (%s): %w", ndc, fetchErr)
				continue
			}
			for _, isbn := range isbns {
				if _, ok := seen[isbn]; !ok {
					isbnCh <- isbn
					seen[isbn] = struct{}{}
				}
			}
		}
	}()

	// 2. Google Books API から書籍情報を取得するゴルーチン
	go func() {
		defer close(bookCh)
		for isbn := range isbnCh {
			fmt.Printf("fetch from Google isbn: %s\n", isbn)
			gbInfo, gbErr := gbClient.Fetch(isbn)
			fmt.Println(gbInfo)
			if gbErr != nil {
				errChan <- fmt.Errorf("google books API 取得エラー (isbn: %s): %w", isbn, gbErr)
				continue
			}
			bookCh <- book.NewBook(
				isbn,
				gbInfo.Title,
				gbInfo.Subtitle,
				gbInfo.Authors,
				gbInfo.Publisher,
				gbInfo.PublishedDate,
				gbInfo.Description,
				gbInfo.InfoLink,
				gbInfo.ImageLinks.Thumbnail,
			)
		}
	}()

	// 3. 書籍情報をチャンクごとにバルクインサート
	insertedCnt := 0
	var bookChunk []*book.Book
	for b := range bookCh {
		bookChunk = append(bookChunk, b)
		if len(bookChunk) >= bulkInsertChunkSize {
			cnt, err := book.BulkInsert(ctx, tx, bookChunk)
			if err != nil {
				errChan <- fmt.Errorf("チャンクのバルクインサートエラー: %w", err)
			} else {
				fmt.Printf("チャンクのバルクインサート完了: %d 件\n", cnt)
				insertedCnt += cnt
			}
			bookChunk = nil
		}
	}
	if len(bookChunk) > 0 {
		cnt, err := book.BulkInsert(ctx, tx, bookChunk)
		if err != nil {
			errChan <- fmt.Errorf("残りのチャンクのバルクインサートエラー: %w", err)
		} else {
			fmt.Printf("残りのチャンクのバルクインサート完了: %d 件\n", cnt)
			insertedCnt += cnt
		}
	}

	close(errChan)

	if insertedCnt > 0 {
		if err := tx.Commit(); err != nil {
			log.Fatalf("コミットエラー: %v", err)
		} else {
			fmt.Printf("%d 件保存しました\n", insertedCnt)
			fmt.Println("トランザクションをコミットしました")
		}
	} else {
		log.Println("トランザクションをロールバックしました")
	}

	elapsedTime := time.Since(startTime)
	log.Printf("[complete] バッチ処理時間: %s", elapsedTime)
}
