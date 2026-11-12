package evaluator

import (
	"os"
	"path/filepath"
	"testing"
)

// sdkRootDir 定位共享标准库 fan-sdk 的根目录（其下含 库/ 与 测试/）。
// 先认 FAN_SDK 环境变量，再按工作区布局从测试工作目录逐级向上找。
func sdkRootDir(t *testing.T) string {
	t.Helper()
	if env := os.Getenv("FAN_SDK"); env != "" {
		if info, err := os.Stat(filepath.Join(env, "库")); err == nil && info.IsDir() {
			abs, _ := filepath.Abs(env)
			return abs
		}
	}
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("获取工作目录失败：%v", err)
	}
	for i := 0; i < 6; i++ {
		candidate := filepath.Join(dir, "fan-sdk")
		if info, err := os.Stat(filepath.Join(candidate, "库")); err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("未找到 fan-sdk：请用 FAN_SDK 环境变量指向它，或放在仓库同级目录")
	return ""
}

// setupSDKModuleEnv 把指定标准库模块（仅文件名，如 "系统.凡"、"数学.凡"）
// 按 库/ 子目录结构复制到临时目录，并以该临时目录为脚本基目录构造环境。
// 模块内部以根相对路径（库/系统）互导，Loader 的搜索目录兜底解析。
func setupSDKModuleEnv(t *testing.T, moduleFiles ...string) *Environment {
	t.Helper()
	sdkRoot := sdkRootDir(t)
	dir := t.TempDir()
	libDir := filepath.Join(dir, "库")
	if err := os.MkdirAll(libDir, 0o700); err != nil {
		t.Fatalf("创建临时 库 目录失败：%v", err)
	}
	for _, name := range moduleFiles {
		data, err := os.ReadFile(filepath.Join(sdkRoot, "库", name))
		if err != nil {
			t.Fatalf("读取 库/%s 失败：%v", name, err)
		}
		if err := os.WriteFile(filepath.Join(libDir, name), data, 0o600); err != nil {
			t.Fatalf("写入 库/%s 失败：%v", name, err)
		}
	}
	env := NewEnvironment()
	env.BaseDir = dir
	env.Loader = NewLoader(dir)
	return env
}
