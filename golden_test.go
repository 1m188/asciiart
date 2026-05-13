// golden_test.go — 黄金文件测试（Golden File Tests）
//
// 本文件实现黄金文件测试：将 test.png 的预期字符画输出保存为
// testdata/golden/default.txt 等黄金文件，测试时对比实际输出与
// 黄金文件是否一致。若算法有预期内的变化，使用 -update 标志更新黄金文件。
//
// 黄金文件列表：
//   - testdata/golden/default.txt    : 40 列默认梯度灰度输出
//   - testdata/golden/colored.txt    : 40 列彩色 ANSI 输出
//   - testdata/golden/detailed.txt   : 40 列详细梯度灰度输出
//
// 运行方式：
//
//	go test -run TestGolden        # 验证黄金文件
//	go test -run TestGolden -update # 更新黄金文件
package main

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"
)

// updateGolden 控制是否更新黄金文件
var updateGolden = flag.Bool("update", false, "更新黄金文件")

// TestGolden_Default 默认梯度灰度输出黄金文件测试
func TestGolden_Default(t *testing.T) {
	opts := DefaultConvertOptions()
	opts.Width = 40
	opts.AspectRatio = 0.5

	result, err := Convert(testPNGPath, opts)
	if err != nil {
		t.Fatalf("Convert() 返回错误: %v", err)
	}

	var buf bytes.Buffer
	if err := RenderToWriter(&buf, result); err != nil {
		t.Fatalf("RenderToWriter() 返回错误: %v", err)
	}

	goldenPath := filepath.Join("testdata", "golden", "default.txt")
	goldenTest(t, goldenPath, buf.Bytes())
}

// TestGolden_Colored 彩色 ANSI 输出黄金文件测试
func TestGolden_Colored(t *testing.T) {
	opts := DefaultConvertOptions()
	opts.Width = 40
	opts.AspectRatio = 0.5

	result, err := ConvertColor(testPNGPath, opts)
	if err != nil {
		t.Fatalf("ConvertColor() 返回错误: %v", err)
	}

	var buf bytes.Buffer
	if err := RenderColorToWriter(&buf, result); err != nil {
		t.Fatalf("RenderColorToWriter() 返回错误: %v", err)
	}

	goldenPath := filepath.Join("testdata", "golden", "colored.txt")
	goldenTest(t, goldenPath, buf.Bytes())
}

// TestGolden_Detailed 详细梯度灰度输出黄金文件测试
func TestGolden_Detailed(t *testing.T) {
	opts := ConvertOptions{
		Width:       40,
		Ramp:        RampDetailed,
		Invert:      false,
		AspectRatio: 0.5,
	}

	result, err := Convert(testPNGPath, opts)
	if err != nil {
		t.Fatalf("Convert(detailed) 返回错误: %v", err)
	}

	var buf bytes.Buffer
	if err := RenderToWriter(&buf, result); err != nil {
		t.Fatalf("RenderToWriter() 返回错误: %v", err)
	}

	goldenPath := filepath.Join("testdata", "golden", "detailed.txt")
	goldenTest(t, goldenPath, buf.Bytes())
}

// goldenTest 黄金文件对比核心逻辑。
// 若 -update 标志为 true，将 actual 写入 goldenPath 更新黄金文件。
// 否则对比 actual 与黄金文件内容是否一致。
func goldenTest(t *testing.T, goldenPath string, actual []byte) {
	t.Helper()

	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0755); err != nil {
			t.Fatalf("创建黄金文件目录失败: %v", err)
		}
		if err := os.WriteFile(goldenPath, actual, 0644); err != nil {
			t.Fatalf("写入黄金文件 %q 失败: %v", goldenPath, err)
		}
		t.Logf("已更新黄金文件: %s (%d 字节)", goldenPath, len(actual))
		return
	}

	expected, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("读取黄金文件 %q 失败: %v\n提示: 使用 -update 标志生成黄金文件", goldenPath, err)
	}

	if !bytes.Equal(actual, expected) {
		// 输出差异摘要：前几行不同的位置
		t.Errorf("黄金文件 %q 不匹配 (%d vs %d 字节)", goldenPath, len(actual), len(expected))
		actualLines := bytes.Split(actual, []byte("\n"))
		expectedLines := bytes.Split(expected, []byte("\n"))
		diffCount := 0
		for i := 0; i < len(actualLines) && i < len(expectedLines) && diffCount < 3; i++ {
			if !bytes.Equal(actualLines[i], expectedLines[i]) {
				t.Errorf("  第 %d 行:\n    got:  %q\n    want: %q", i+1,
					string(actualLines[i]), string(expectedLines[i]))
				diffCount++
			}
		}
		if len(actualLines) != len(expectedLines) {
			t.Errorf("  行数: got %d, want %d", len(actualLines), len(expectedLines))
		}
		t.Log("提示: 若算法变更是预期的，使用 go test -run TestGolden -update 更新黄金文件")
	}
}
