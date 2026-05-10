package thirdparty

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed aria2c/*
var Aria2cFile embed.FS

func ReadAndWriteForAria2c() (string, error) {
	data, err := Aria2cFile.ReadFile("aria2c/aria2c.exe")
	if err != nil {
		return "", fmt.Errorf("读取嵌入的 aria2server 文件失败: %w", err)
	}
	aria2Path := filepath.Join(os.TempDir(), "aria2c.exe")
	if err := os.WriteFile(aria2Path, data, 0755); err != nil {
		return "", fmt.Errorf("写入 aria2server 到临时目录失败: %w", err)
	}
	return aria2Path, nil
}
