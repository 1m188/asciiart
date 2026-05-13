// render.go — 输出渲染
//
// 本文件负责 asciiart 项目中的输出渲染环节，将 converter.go 生成的
// ASCIIResult 或 ColorResult 结构体转化为最终的输出形式。支持三种输出
// 目标：纯文本终端（stdout / 文件）、ANSI 真彩色终端、HTML 文档。
//
// 主要功能：
//   - RenderToWriter()      : 将 ASCIIResult 纯文本逐行写入 io.Writer
//   - RenderColorToWriter() : 将 ColorResult（含 ANSI 序列）逐行写入 io.Writer
//   - RenderToHTML()        : 将 ColorResult 渲染为完整的 HTML 文档
//     自动解析 ANSI 38;2;R;G;B 序列 → <span style="color:rgb(...)">
//   - ansiToHTML()          : 内部函数，将一行 ANSI 字符串转换为 HTML 片段
//
// ANSI 真彩色格式：\033[38;2;{R};{G};{B}m{char}\033[0m
// HTML 输出格式：完整 HTML5 文档，monospace 字体，内联颜色样式
//
// 本文件仅依赖标准库 fmt、html、io、strings。
package main

import (
	"fmt"
	"html"
	"io"
	"strings"
)

// RenderToWriter 将 ASCIIResult（纯文本结果）渲染到 io.Writer。
// 每行末尾添加换行符。
func RenderToWriter(w io.Writer, result *ASCIIResult) error {
	if result == nil {
		return fmt.Errorf("ASCIIResult 不能为 nil")
	}
	for _, line := range result.Lines {
		if _, err := fmt.Fprintln(w, line); err != nil {
			return fmt.Errorf("写入输出失败: %w", err)
		}
	}
	return nil
}

// RenderColorToWriter 将 ColorResult（带 ANSI 颜色的结果）渲染到 io.Writer。
// Lines 中已包含 ANSI 转义序列，直接输出即可。
func RenderColorToWriter(w io.Writer, result *ColorResult) error {
	if result == nil {
		return fmt.Errorf("ColorResult 不能为 nil")
	}
	for _, line := range result.Lines {
		if _, err := fmt.Fprintln(w, line); err != nil {
			return fmt.Errorf("写入输出失败: %w", err)
		}
	}
	return nil
}

// RenderToHTML 将 ColorResult 渲染为完整的 HTML 文档字符串。
// 每个字符作为 <span> 元素，使用内联 style="color:rgb(R,G,B)" 着色。
func RenderToHTML(result *ColorResult) string {
	if result == nil {
		return ""
	}

	var sb strings.Builder

	// HTML 头部
	sb.WriteString(`<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<style>
  pre { line-height: 1; font-family: monospace; font-size: 8px; }
  span { display: inline; }
</style>
</head>
<body>
<pre>
`)

	for _, line := range result.Lines {
		// 解析 ANSI 转义序列并转换为 HTML <span> 元素
		htmlLine := ansiToHTML(line)
		sb.WriteString(htmlLine)
		sb.WriteByte('\n')
	}

	sb.WriteString(`</pre>
</body>
</html>
`)

	return sb.String()
}

// ansiToHTML 将包含 ANSI 真彩色转义序列的字符串转换为 HTML。
// 格式：\033[38;2;{R};{G};{B}m{char}\033[0m → <span style="color:rgb(R,G,B)">char</span>
func ansiToHTML(line string) string {
	var sb strings.Builder
	sb.Grow(len(line) * 2)

	i := 0
	runes := []rune(line)
	n := len(runes)

	for i < n {
		// 查找 ANSI 转义序列起始 \033[
		if runes[i] == '\033' && i+1 < n && runes[i+1] == '[' {
			// 解析 ANSI 序列
			seqEnd := i + 2
			var seqContent strings.Builder
			for seqEnd < n && runes[seqEnd] != 'm' {
				seqContent.WriteRune(runes[seqEnd])
				seqEnd++
			}
			if seqEnd < n {
				seqEnd++ // 跳过 'm'
			}

			seq := seqContent.String()

			// 检查是否为重置序列 [0m
			if seq == "0" {
				i = seqEnd
				continue
			}

			// 解析 38;2;R;G;B 格式
			if strings.HasPrefix(seq, "38;2;") {
				parts := strings.Split(seq, ";")
				if len(parts) >= 5 {
					r := parts[2]
					g := parts[3]
					b := parts[4]

					// 找到下一个字符（跳过可能的 0m 结束序列）
					if seqEnd < n {
						ch := runes[seqEnd]
						// 转义 HTML 敏感字符
						escapedChar := html.EscapeString(string(ch))
						// 生成 span 元素
						fmt.Fprintf(&sb, `<span style="color:rgb(%s,%s,%s)">%s</span>`,
							r, g, b, escapedChar)
						seqEnd++ // 跳过字符
					}
				}
			}
			i = seqEnd
		} else {
			// 普通字符，HTML 转义后直接写入
			sb.WriteString(html.EscapeString(string(runes[i])))
			i++
		}
	}

	return sb.String()
}
