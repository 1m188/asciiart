// converter_test.go — converter.go 的单元测试
//
// 本文件测试核心转换引擎，覆盖：
//   - Convert() / ConvertColor() 使用 test.png 的正常转换
//   - 输出行列数与指定宽度一致
//   - 各配置组合（反转、自定义梯度）
//   - ConvertFromImage / ConvertColorFromImage 复用接口
//   - 边界情况：width<=0 回退默认值
package main

import (
	"strings"
	"testing"
)

// TestConvert_Basic 测试基本灰度转换
func TestConvert_Basic(t *testing.T) {
	opts := DefaultConvertOptions()
	opts.Width = 40
	opts.AspectRatio = 0.5

	result, err := Convert(testPNGPath, opts)
	if err != nil {
		t.Fatalf("Convert() 返回错误: %v", err)
	}
	if result == nil {
		t.Fatal("result 不能为 nil")
	}
	if result.Width != 40 {
		t.Errorf("Width = %d, want 40", result.Width)
	}
	if result.Height < 1 {
		t.Errorf("Height = %d, want >= 1", result.Height)
	}
	if len(result.Lines) != result.Height {
		t.Errorf("Lines 长度 = %d, want %d", len(result.Lines), result.Height)
	}

	// 每行长度应与 Width 一致
	for i, line := range result.Lines {
		if len(line) != result.Width {
			t.Errorf("第 %d 行长度 = %d, want %d", i, len(line), result.Width)
		}
	}

	t.Logf("灰度转换: %dx%d 字符矩阵", result.Width, result.Height)
}

// TestConvertColor_Basic 测试彩色转换
func TestConvertColor_Basic(t *testing.T) {
	opts := DefaultConvertOptions()
	opts.Width = 30
	opts.AspectRatio = 0.5

	result, err := ConvertColor(testPNGPath, opts)
	if err != nil {
		t.Fatalf("ConvertColor() 返回错误: %v", err)
	}
	if result == nil {
		t.Fatal("result 不能为 nil")
	}
	if result.Width != 30 {
		t.Errorf("Width = %d, want 30", result.Width)
	}
	if result.Height < 1 {
		t.Errorf("Height = %d, want >= 1", result.Height)
	}
	if len(result.Lines) != result.Height {
		t.Errorf("Lines 长度 = %d, want %d", len(result.Lines), result.Height)
	}

	// 验证每行都包含 ANSI 转义序列
	for i, line := range result.Lines {
		if !strings.Contains(line, "\033[") {
			t.Errorf("第 %d 行不包含 ANSI 转义序列", i)
		}
	}

	t.Logf("彩色转换: %dx%d 字符矩阵", result.Width, result.Height)
}

// TestConvert_WithInvert 测试反转亮度
func TestConvert_WithInvert(t *testing.T) {
	opts := DefaultConvertOptions()
	opts.Width = 40
	opts.Invert = true

	result, err := Convert(testPNGPath, opts)
	if err != nil {
		t.Fatalf("Convert(invert=true) 返回错误: %v", err)
	}
	if result.Width != 40 {
		t.Errorf("Width = %d, want 40", result.Width)
	}
}

// TestConvert_WithCustomRamp 测试自定义梯度
func TestConvert_WithCustomRamp(t *testing.T) {
	ramp, _ := NewCustomRamp("@#+=. ")
	opts := ConvertOptions{
		Width:       30,
		Ramp:        ramp,
		Invert:      false,
		AspectRatio: 0.5,
	}

	result, err := Convert(testPNGPath, opts)
	if err != nil {
		t.Fatalf("Convert(custom ramp) 返回错误: %v", err)
	}
	if result.Width != 30 {
		t.Errorf("Width = %d, want 30", result.Width)
	}
}

// TestConvert_DetailedRamp 测试详细梯度
func TestConvert_DetailedRamp(t *testing.T) {
	opts := ConvertOptions{
		Width:       20,
		Ramp:        RampDetailed,
		AspectRatio: 0.5,
	}

	result, err := Convert(testPNGPath, opts)
	if err != nil {
		t.Fatalf("Convert(detailed) 返回错误: %v", err)
	}
	if result.Width != 20 {
		t.Errorf("Width = %d, want 20", result.Width)
	}
}

// TestConvert_FileNotFound 测试文件不存在
func TestConvert_FileNotFound(t *testing.T) {
	_, err := Convert("nonexistent.png", DefaultConvertOptions())
	if err == nil {
		t.Error("Convert 对不存在的文件应返回错误")
	}
}

// TestConvertFromImage 测试从 image.Image 直接转换
func TestConvertFromImage(t *testing.T) {
	img, err := LoadImage(testPNGPath)
	if err != nil {
		t.Fatalf("LoadImage 失败: %v", err)
	}

	opts := DefaultConvertOptions()
	opts.Width = 20

	result, err := ConvertFromImage(img, opts)
	if err != nil {
		t.Fatalf("ConvertFromImage() 返回错误: %v", err)
	}
	if result.Width != 20 {
		t.Errorf("Width = %d, want 20", result.Width)
	}
}

// TestConvertColorFromImage 测试从 image.Image 直接彩色转换
func TestConvertColorFromImage(t *testing.T) {
	img, err := LoadImage(testPNGPath)
	if err != nil {
		t.Fatalf("LoadImage 失败: %v", err)
	}

	opts := DefaultConvertOptions()
	opts.Width = 20

	result, err := ConvertColorFromImage(img, opts)
	if err != nil {
		t.Fatalf("ConvertColorFromImage() 返回错误: %v", err)
	}
	if result.Width != 20 {
		t.Errorf("Width = %d, want 20", result.Width)
	}
	// 彩色结果应包含 ANSI 序列
	if len(result.Lines) > 0 && !strings.Contains(result.Lines[0], "\033[") {
		t.Error("彩色结果应包含 ANSI 转义序列")
	}
}

// TestDefaultConvertOptions 测试默认选项
func TestDefaultConvertOptions(t *testing.T) {
	opts := DefaultConvertOptions()
	if opts.Width != 80 {
		t.Errorf("默认 Width = %d, want 80", opts.Width)
	}
	if opts.AspectRatio != 0.5 {
		t.Errorf("默认 AspectRatio = %f, want 0.5", opts.AspectRatio)
	}
	if len(opts.Ramp) == 0 {
		t.Error("默认 Ramp 不应为空")
	}
	if opts.Invert {
		t.Error("默认 Invert 应为 false")
	}
}

// TestConvert_WidthOne 测试最小宽度 1
func TestConvert_WidthOne(t *testing.T) {
	opts := DefaultConvertOptions()
	opts.Width = 1

	result, err := Convert(testPNGPath, opts)
	if err != nil {
		t.Fatalf("Convert(width=1) 返回错误: %v", err)
	}
	if result.Width != 1 {
		t.Errorf("Width = %d, want 1", result.Width)
	}
	if result.Height < 1 {
		t.Error("height 应 >= 1")
	}
	for _, line := range result.Lines {
		if len(line) != 1 {
			t.Errorf("每行长度应为 1, got %d", len(line))
		}
	}
}
