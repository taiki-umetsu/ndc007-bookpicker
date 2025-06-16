package env

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"slices"

	"github.com/joho/godotenv"
)

var allowedEnvs = []string{"development", "test", "production"}

func Load() {
	env := os.Getenv("GO_ENV")
	if env == "" {
		log.Fatalf("GO_ENV にデータが設定されていません. 設定可能な値: %v", allowedEnvs)
	}

	if !slices.Contains(allowedEnvs, env) {
		log.Fatalf("GO_ENV に無効な値が設定されています: %s\n有効な値は: %v", env, allowedEnvs)
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
