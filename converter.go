// converter.go — 核心转换引擎
//
// 本文件是 asciiart 项目的核心模块，负责将图片数据转换为字符画结果。
// 它协调调用 image.go 的加载/预处理功能和 chars.go 的字符映射功能，
// 生成结构化的 ASCIIResult（灰度模式）或 ColorResult（彩色模式）。
//
// 主要功能：
//   - Convert()               : 灰度模式转换，图片路径 → ASCIIResult
//   - ConvertFromImage()      : 灰度模式转换，image.Image → ASCIIResult（复用接口）
//   - ConvertColor()          : 彩色模式转换，图片路径 → ColorResult
//   - ConvertColorFromImage() : 彩色模式转换，image.Image → ColorResult（复用接口）
//   - ConvertOptions          : 转换选项结构体（宽度/梯度/反转/宽高比）
//   - DefaultConvertOptions() : 返回合理的默认转换选项
//
// 数据流：
//
//	LoadImage → Preprocess（灰度图用于选字符）→ 遍历灰度像素
//	→ CharRamp.MapGrayToChar() → 组装 ASCIIResult
//	彩色模式额外：原始图缩放 → 逐像素提取 RGB → 拼接 ANSI 转义序列
//
// 外部依赖：golang.org/x/image/draw（彩色模式下的颜色图缩放）
package main

import (
	"fmt"
	"image"

	"golang.org/x/image/draw"

	"strings"
)

// ASCIIResult 字符画转换结果（纯灰度文本模式）。
type ASCIIResult struct {
	Lines  []string // 每行字符串（纯字符，无颜色）
	Width  int      // 字符列数
	Height int      // 字符行数
}

// ColorResult 带颜色信息的转换结果。
// Lines 中每行包含 ANSI 真彩色转义序列。
type ColorResult struct {
	Lines  []string // 每行含 ANSI 转义序列的字符串
	Width  int      // 字符列数
	Height int      // 字符行数
}

// ConvertOptions 转换选项，控制字符画输出的各种参数。
type ConvertOptions struct {
	Width       int      // 输出宽度（字符列数），默认 80
	Ramp        CharRamp // 字符梯度
	Invert      bool     // 是否反转亮度映射
	AspectRatio float64  // 字符宽高比修正系数，默认 0.5
}

// DefaultConvertOptions 返回默认转换选项。
func DefaultConvertOptions() ConvertOptions {
	return ConvertOptions{
		Width:       80,
		Ramp:        RampStandard,
		Invert:      false,
		AspectRatio: 0.5,
	}
}

// Convert 灰度模式转换：加载图片并转为 ASCII 字符画。
// 这是最常用的入口函数，内部依次调用 LoadImage → Preprocess → 字符映射。
func Convert(path string, opts ConvertOptions) (*ASCIIResult, error) {
	img, err := LoadImage(path)
	if err != nil {
		return nil, err
	}

	return ConvertFromImage(img, opts)
}

// ConvertFromImage 从已加载的 image.Image 进行灰度模式转换。
// 适用场景：调用方自行加载图片后复用此函数。
func ConvertFromImage(img image.Image, opts ConvertOptions) (*ASCIIResult, error) {
	// 参数默认值处理
	if opts.Width <= 0 {
		opts.Width = 80
	}
	if opts.AspectRatio <= 0 {
		opts.AspectRatio = 0.5
	}
	if len(opts.Ramp) == 0 {
		opts.Ramp = RampStandard
	}

	// 预处理：缩放并灰度化
	gray := Preprocess(img, opts.Width, opts.AspectRatio)
	bounds := gray.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	// 遍历灰度图每个像素，映射为字符
	lines := make([]string, h)
	for y := 0; y < h; y++ {
		var sb strings.Builder
		sb.Grow(w)
		for x := 0; x < w; x++ {
			grayVal := gray.GrayAt(x, y).Y
			ch := opts.Ramp.MapGrayToChar(grayVal, opts.Invert)
			sb.WriteRune(ch)
		}
		lines[y] = sb.String()
	}

	return &ASCIIResult{
		Lines:  lines,
		Width:  w,
		Height: h,
	}, nil
}

// ConvertColor 彩色模式转换：加载图片并转为带 ANSI 颜色的 ASCII 字符画。
func ConvertColor(path string, opts ConvertOptions) (*ColorResult, error) {
	img, err := LoadImage(path)
	if err != nil {
		return nil, err
	}

	return ConvertColorFromImage(img, opts)
}

// ConvertColorFromImage 从已加载的 image.Image 进行彩色模式转换。
// 适用场景：调用方自行加载图片后复用此函数。
func ConvertColorFromImage(img image.Image, opts ConvertOptions) (*ColorResult, error) {
	// 参数默认值处理
	if opts.Width <= 0 {
		opts.Width = 80
	}
	if opts.AspectRatio <= 0 {
		opts.AspectRatio = 0.5
	}
	if len(opts.Ramp) == 0 {
		opts.Ramp = RampStandard
	}

	// 预处理：缩放并灰度化（用于确定字符选择）
	preprocessed := Preprocess(img, opts.Width, opts.AspectRatio)
	bounds := preprocessed.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	// 同时缩放原始图像以获取对应像素的颜色
	colorScaled := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.ApproxBiLinear.Scale(colorScaled, colorScaled.Bounds(), img, img.Bounds(), draw.Over, nil)

	lines := make([]string, h)
	for y := 0; y < h; y++ {
		var sb strings.Builder
		// 每行预估约 w * 30 字节（ANSI 转义序列开销）
		sb.Grow(w * 30)
		for x := 0; x < w; x++ {
			grayVal := preprocessed.GrayAt(x, y).Y
			ch := opts.Ramp.MapGrayToChar(grayVal, opts.Invert)
			// 获取对应像素的颜色
			r, g, b, _ := colorScaled.At(x, y).RGBA()
			// ANSI 真彩色格式：\033[38;2;{R};{G};{B}m{char}\033[0m
			sb.WriteString(fmt.Sprintf("\033[38;2;%d;%d;%dm%c\033[0m",
				uint8(r>>8), uint8(g>>8), uint8(b>>8), ch))
		}
		lines[y] = sb.String()
	}

	return &ColorResult{
		Lines:  lines,
		Width:  w,
		Height: h,
	}, nil
}
