package main

import (
	"fmt"
	"log"
	"os"

	"github.com/taiki-umetsu/ndc007-bookpicker/internal/cinii"
	"github.com/taiki-umetsu/ndc007-bookpicker/internal/database"
	"github.com/taiki-umetsu/ndc007-bookpicker/internal/googlebooks"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
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

	ndcList := []string{
		"007", // General（完全一致）
		//"007.1*",  // 情報学基礎理論
		//"007.3*",  // 情報機器・装置
		//"007.5*",  // 情報処理・情報システム
		//"007.6*",  // 情報ネットワーク・通信
		//"007.63*", // インターネット
		//"007.64*", // ウェブ
	}
	var isbns []string
	for _, ndc := range ndcList {
		fmt.Printf("\nfetch from CiNii 分類コード: %s\n", ndc)

		isbns, err = ciniiClient.FetchRandomISBNs(ndc, yearFrom, count)
		if err != nil {
			log.Printf("❌ %s: ISBN取得失敗: %v", ndc, err)
			continue
		}

		if len(isbns) == 0 {
			fmt.Println("ISBN が見つかりませんでした")
			continue
		}
	}
	fmt.Println(isbns)

	gbClient := googlebooks.NewClient(gbKey)
	for _, isbn := range isbns {
		fmt.Printf("\nfetch from Google isbn: %s\n", isbn)

		book, err := gbClient.Fetch(isbn)
		if err != nil {
			log.Fatal("本情報取得失敗:", err)
		}
		fmt.Println(book)
	}
}
