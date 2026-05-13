// render_test.go — render.go 的单元测试
//
// 本文件测试输出渲染模块，覆盖：
//   - RenderToWriter() 正常输出及 nil 保护
//   - RenderColorToWriter() 正常输出及 nil 保护
//   - RenderToHTML() 完整 HTML 文档结构验证 + nil 保护
//   - ansiToHTML() ANSI 转义序列 → HTML span 转换正确性
package main

import (
	"bytes"
	"strings"
	"testing"
)

// TestRenderToWriter_Basic 测试纯文本渲染到 bytes.Buffer
func TestRenderToWriter_Basic(t *testing.T) {
	result := &ASCIIResult{
		Lines:  []string{"abc", "123"},
		Width:  3,
		Height: 2,
	}

	var buf bytes.Buffer
	err := RenderToWriter(&buf, result)
	if err != nil {
		t.Fatalf("RenderToWriter() 返回错误: %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	if len(lines) != 2 {
		t.Errorf("输出行数 = %d, want 2", len(lines))
	}
	if lines[0] != "abc" || lines[1] != "123" {
		t.Errorf("输出内容不符: %q", lines)
	}
}

// TestRenderToWriter_Nil 测试 nil 保护
func TestRenderToWriter_Nil(t *testing.T) {
	var buf bytes.Buffer
	err := RenderToWriter(&buf, nil)
	if err == nil {
		t.Error("RenderToWriter(nil) 应返回错误")
	}
	if !strings.Contains(err.Error(), "nil") {
		t.Errorf("错误信息应包含 nil, got %q", err.Error())
	}
}

// TestRenderColorToWriter_Basic 测试彩色渲染输出
func TestRenderColorToWriter_Basic(t *testing.T) {
	result := &ColorResult{
		Lines:  []string{"\033[38;2;255;0;0mX\033[0m", "\033[38;2;0;255;0mY\033[0m"},
		Width:  1,
		Height: 2,
	}

	var buf bytes.Buffer
	err := RenderColorToWriter(&buf, result)
	if err != nil {
		t.Fatalf("RenderColorToWriter() 返回错误: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "\033[38;2;255;0;0mX\033[0m") {
		t.Error("输出应包含 ANSI 红色序列")
	}
	if !strings.Contains(output, "\033[38;2;0;255;0mY\033[0m") {
		t.Error("输出应包含 ANSI 绿色序列")
	}
}

// TestRenderColorToWriter_Nil 测试 ColorResult nil 保护
func TestRenderColorToWriter_Nil(t *testing.T) {
	var buf bytes.Buffer
	err := RenderColorToWriter(&buf, nil)
	if err == nil {
		t.Error("RenderColorToWriter(nil) 应返回错误")
	}
	if !strings.Contains(err.Error(), "nil") {
		t.Errorf("错误信息应包含 nil, got %q", err.Error())
	}
}

// TestRenderToHTML_Basic 测试 HTML 渲染
func TestRenderToHTML_Basic(t *testing.T) {
	result := &ColorResult{
		Lines:  []string{"\033[38;2;255;0;0mX\033[0m"},
		Width:  1,
		Height: 1,
	}

	html := RenderToHTML(result)

	// 验证 HTML 结构
	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("HTML 应包含 DOCTYPE")
	}
	if !strings.Contains(html, "<span style=\"color:rgb(255,0,0)\">X</span>") {
		t.Error("HTML 应包含红色 span 元素")
	}
	if !strings.Contains(html, "<pre>") || !strings.Contains(html, "</pre>") {
		t.Error("HTML 应包含 <pre> 元素")
	}
	if !strings.Contains(html, "<meta charset=\"utf-8\">") {
		t.Error("HTML 应包含 charset 声明")
	}
}

// TestRenderToHTML_Nil 测试 nil 保护
func TestRenderToHTML_Nil(t *testing.T) {
	html := RenderToHTML(nil)
	if html != "" {
		t.Errorf("RenderToHTML(nil) 应返回空字符串, got %q", html)
	}
}

// TestAnsiToHTML_ColorParsing 验证 ANSI → HTML 颜色解析
func TestAnsiToHTML_ColorParsing(t *testing.T) {
	tests := []struct {
		name string
		ansi string
		want string
	}{
		{
			name: "红色 X",
			ansi: "\033[38;2;255;0;0mX\033[0m",
			want: `<span style="color:rgb(255,0,0)">X</span>`,
		},
		{
			name: "绿色 Y",
			ansi: "\033[38;2;0;255;0mY\033[0m",
			want: `<span style="color:rgb(0,255,0)">Y</span>`,
		},
		{
			name: "蓝色 Z",
			ansi: "\033[38;2;0;0;255mZ\033[0m",
			want: `<span style="color:rgb(0,0,255)">Z</span>`,
		},
		{
			name: "纯文本无 ANSI",
			ansi: "hello",
			want: "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ansiToHTML(tt.ansi)
			if got != tt.want {
				t.Errorf("ansiToHTML() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestAnsiToHTML_ComplexLine 测试混合 ANSI 和普通文本
func TestAnsiToHTML_ComplexLine(t *testing.T) {
	// 模拟一行包含多个彩色字符
	line := "\033[38;2;255;0;0mA\033[0m \033[38;2;0;255;0mB\033[0m"
	got := ansiToHTML(line)

	if !strings.Contains(got, `<span style="color:rgb(255,0,0)">A</span>`) {
		t.Error("应包含红色 A span")
	}
	if !strings.Contains(got, `<span style="color:rgb(0,255,0)">B</span>`) {
		t.Error("应包含绿色 B span")
	}
}

// TestRenderToHTML_EndToEnd 端到端测试：ConvertColor → RenderToHTML
func TestRenderToHTML_EndToEnd(t *testing.T) {
	opts := DefaultConvertOptions()
	opts.Width = 10
	opts.AspectRatio = 0.5

	result, err := ConvertColor(testPNGPath, opts)
	if err != nil {
		t.Fatalf("ConvertColor() 返回错误: %v", err)
	}

	html := RenderToHTML(result)
	if len(html) == 0 {
		t.Fatal("RenderToHTML 返回空字符串")
	}
	if !strings.Contains(html, "</html>") {
		t.Error("HTML 应正确闭合")
	}
}
