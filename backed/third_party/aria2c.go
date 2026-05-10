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

	// 使用 PID 生成唯一文件名，避免多实例/重启时文件被锁导致写入失败
	aria2Path := filepath.Join(os.TempDir(), fmt.Sprintf("aria2c_%d.exe", os.Getpid()))
	if err := os.WriteFile(aria2Path, data, 0755); err != nil {
		return "", fmt.Errorf("写入 aria2server 到临时目录失败: %w", err)
	}
	return aria2Path, nil
}
