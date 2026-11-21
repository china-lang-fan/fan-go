// Package sdkpath 定位共享标准库 fan-sdk（其下含 库/ 与 测试/）。
//
// 标准库由 Go 版与 Rust 版共用、单独成仓，既可能放在实现仓库的同级目录
// （本机开发），也可能被 CI 检出到仓库内部并用 FAN_SDK 指向它。本包统一两种
// 布局的解析：先认 FAN_SDK 环境变量，再从工作目录逐级向上寻找 fan-sdk。
package sdkpath

import (
	"errors"
	"os"
	"path/filepath"
)

// ErrNotFound 表示既没有可用的 FAN_SDK，也没能在上级目录找到 fan-sdk。
var ErrNotFound = errors.New("未找到 fan-sdk：请用 FAN_SDK 环境变量指向它，或放在仓库同级目录")

// Root 返回 fan-sdk 根目录的绝对路径。
//
// 解析顺序：
//  1. FAN_SDK 环境变量（其下需含 库/ 子目录，否则忽略）；
//  2. 从当前工作目录逐级向上，寻找含 库/ 子目录的 fan-sdk 目录（最多上溯 6 级）。
func Root() (string, error) {
	if env := os.Getenv("FAN_SDK"); env != "" {
		if info, err := os.Stat(filepath.Join(env, "库")); err == nil && info.IsDir() {
			return filepath.Abs(env)
		}
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for i := 0; i < 6; i++ {
		candidate := filepath.Join(dir, "fan-sdk")
		if info, err := os.Stat(filepath.Join(candidate, "库")); err == nil && info.IsDir() {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", ErrNotFound
}

// LibFile 返回标准库 库/ 目录下指定文件（仅文件名，如 "系统.凡"）的绝对路径。
func LibFile(name string) (string, error) {
	root, err := Root()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "库", name), nil
}
