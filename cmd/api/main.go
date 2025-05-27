package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/taiki-umetsu/ndc007-bookpicker/internal/database"
	"github.com/taiki-umetsu/ndc007-bookpicker/internal/env"
	"github.com/taiki-umetsu/ndc007-bookpicker/internal/server"
)

func main() {
	env.Load()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("環境変数 DATABASE_URL が未設定です")
	}

	db, err := database.Setup(dsn)
	if err != nil {
		log.Fatal("DB接続エラー:", err)
	}
	defer db.Close()

	handler := server.NewHandler(db)
	router := server.NewRouter(handler)

	fmt.Println("Server is running on port 8080")
	http.ListenAndServe(":8080", router)
}
