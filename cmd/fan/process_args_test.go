package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunPassesScriptArgs(t *testing.T) {
	dir := t.TempDir()
	libDir := filepath.Join(dir, "库")
	if err := os.MkdirAll(libDir, 0o700); err != nil {
		t.Fatalf("创建临时 库 目录失败：%v", err)
	}
	for _, name := range []string{"系统.凡", "转换.凡", "进程.凡"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "..", "fan-sdk", "库", name))
		if err != nil {
			t.Fatalf("读取 库/%s 失败：%v", name, err)
		}
		if err := os.WriteFile(filepath.Join(libDir, name), data, 0600); err != nil {
			t.Fatalf("写入 库/%s 失败：%v", name, err)
		}
	}
	path := filepath.Join(dir, "参数.凡")
	source := `导入 "库/进程" 作为 进程模块

变量 参数列表 = 进程模块.参数()
变量 首参 = 参数列表[0]
变量 脚本路径 = 进程模块.脚本()
首参
`
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatalf("写入脚本失败：%v", err)
	}
	if err := run([]string{"run", path, "参数值"}); err != nil {
		t.Fatalf("运行脚本失败：%v", err)
	}
}
