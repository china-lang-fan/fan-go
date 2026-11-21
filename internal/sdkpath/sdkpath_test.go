package sdkpath

import (
	"os"
	"path/filepath"
	"testing"
)

// 在临时目录构造一个含 库/ 子目录的假 fan-sdk，返回其根路径。
func writeFakeSDK(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "库"), 0o700); err != nil {
		t.Fatalf("构造假 sdk 失败：%v", err)
	}
	return root
}

func TestRootFromEnv(t *testing.T) {
	fake := writeFakeSDK(t)
	t.Setenv("FAN_SDK", fake)

	got, err := Root()
	if err != nil {
		t.Fatalf("FAN_SDK 指向有效目录时不应报错：%v", err)
	}
	want, _ := filepath.Abs(fake)
	if got != want {
		t.Fatalf("FAN_SDK 解析错误：得到 %s，期望 %s", got, want)
	}
}

func TestRootEnvWithoutLibDirIgnored(t *testing.T) {
	// FAN_SDK 指向不含 库/ 的目录时应被忽略，转而走上溯逻辑；
	// 这里同时给一个有效的上溯 fan-sdk，验证忽略后能回退成功。
	invalid := t.TempDir()
	t.Setenv("FAN_SDK", invalid)

	// 切换工作目录到临时区，避免命中开发机真实的上级 fan-sdk。
	work := t.TempDir()
	origWd, _ := os.Getwd()
	if err := os.Chdir(work); err != nil {
		t.Fatalf("切换工作目录失败：%v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWd) })

	sdk := filepath.Join(work, "fan-sdk")
	if err := os.MkdirAll(filepath.Join(sdk, "库"), 0o700); err != nil {
		t.Fatalf("构造上溯 fan-sdk 失败：%v", err)
	}

	got, err := Root()
	if err != nil {
		t.Fatalf("无效 FAN_SDK 应回退上溯并成功：%v", err)
	}
	if got != sdk {
		t.Fatalf("上溯解析错误：得到 %s，期望 %s", got, sdk)
	}
}

func TestRootNotFound(t *testing.T) {
	t.Setenv("FAN_SDK", "")
	work := t.TempDir()
	origWd, _ := os.Getwd()
	if err := os.Chdir(work); err != nil {
		t.Fatalf("切换工作目录失败：%v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWd) })

	if _, err := Root(); err == nil {
		t.Fatal("无 FAN_SDK 且上级无 fan-sdk 时应返回 ErrNotFound")
	}
}

func TestLibFile(t *testing.T) {
	fake := writeFakeSDK(t)
	t.Setenv("FAN_SDK", fake)

	got, err := LibFile("系统.凡")
	if err != nil {
		t.Fatalf("LibFile 不应报错：%v", err)
	}
	want := filepath.Join(fake, "库", "系统.凡")
	if got != want {
		t.Fatalf("LibFile 路径错误：得到 %s，期望 %s", got, want)
	}
}
