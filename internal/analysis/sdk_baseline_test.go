package analysis

import (
	"os"
	"path/filepath"
	"testing"
)

// TestSDKBaselineNoUndefined 保证对整个 SDK 静态扫描时，
// undefined-name 与 import-not-found 不产生误报。
func TestSDKBaselineNoUndefined(t *testing.T) {
	sdkDir := filepath.Join("..", "..", "..", "fan-sdk")
	// 标准库已按 库/ 子目录组织，模块间用根相对路径（库/系统）互导。
	files, err := filepath.Glob(filepath.Join(sdkDir, "库", "*.凡"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Skip("未找到 SDK 文件")
	}
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		// 以 SDK 根为基目录构建索引，使根相对导入（库/系统）能经搜索目录解析。
		uri := "file://" + f
		idx := buildIndexWithBase(sdkDir, uri, string(data))
		for _, d := range AnalyzeIndex(idx) {
			if d.Code == "undefined-name" || d.Code == "import-not-found" {
				t.Errorf("%s:%d %s: %s", filepath.Base(f), d.Line, d.Code, d.Message)
			}
		}
	}
}
