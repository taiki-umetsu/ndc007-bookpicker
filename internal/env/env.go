package env

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/joho/godotenv"
)

func Load() {
	env := os.Getenv("GO_ENV")
	if env == "" {
		log.Fatal("GO_ENV にデータが設定されていません")
	}
	if env == "production" {
		return
	}

	// env.go があるディレクトリを基準にする
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("env.go のパス取得に失敗しました")
	}

	envDir := filepath.Dir(thisFile)
	envFile := filepath.Join(envDir, fmt.Sprintf(".env.%s", env))

	if err := godotenv.Load(envFile); err != nil {
		log.Fatalf("エラー: %s の読み込みに失敗しました: %v", envFile, err)
	}
}
