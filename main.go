// main.go — CLI 入口与命令行参数解析
//
// 本文件是 asciiart 项目的程序入口，负责解析命令行参数、校验输入、
// 构建转换选项、协调各模块完成转换流程，并将最终结果输出到指定目标。
//
// 支持的 CLI 参数：
//
//	-f / --file        : 输入图片路径（必填，支持 PNG/JPEG/GIF/BMP/WebP）
//	-w / --width       : 输出宽度（字符列数），默认 80
//	-c / --color       : 启用 ANSI 真彩色输出（24-bit）
//	-i / --invert      : 反转亮度映射
//	-r / --ramp        : 字符梯度预设名（short/standard/detailed/blocks），默认 standard
//	--custom-ramp      : 自定义字符梯度字符串（从暗到亮排列）
//	-o / --output      : 输出到文件路径（默认 stdout）
//	--html             : 以 HTML 格式输出（始终为彩色模式）
//	--aspect           : 字符宽高比修正系数，默认 0.5
//
// 执行流程：
//  1. parseFlags() 解析 flag → 校验图片路径必选参数
//  2. 根据 -r / --custom-ramp 构建 CharRamp
//  3. 根据 -o 确定输出目标（stdout / 文件）
//  4. 根据模式（html / color / 灰度）调用相应转换与渲染函数
//  5. 错误信息通过 log/slog 输出到 stderr，确保 stdout 不被污染
//
// 外部依赖：仅标准库 flag、fmt、log/slog、os、path/filepath。
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

// CLI 参数默认值常量
const (
	defaultWidth       = 80
	defaultAspectRatio = 0.5
	defaultRamp        = "standard"
)

func main() {
	// 日志输出到 stderr，确保 stdout 仅输出字符画内容不被污染
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// 解析命令行参数
	opts, err := parseFlags(os.Args[1:])
	if err != nil {
		// 用户请求帮助信息（-h / --help）：静默退出，帮助已由 fs.Usage 输出
		if err == flag.ErrHelp {
			os.Exit(0)
		}
		logger.Error("参数解析失败", "error", err)
		fmt.Fprintf(os.Stderr, "用法: asciiart [选项] -f <图片路径>\n")
		os.Exit(1)
	}

	// 构建转换选项
	convertOpts := DefaultConvertOptions()
	convertOpts.Width = opts.width
	convertOpts.Invert = opts.invert
	convertOpts.AspectRatio = opts.aspectRatio

	// 设置字符梯度
	if opts.customRamp != "" {
		ramp, err := NewCustomRamp(opts.customRamp)
		if err != nil {
			logger.Error("无效的自定义梯度", "error", err)
			os.Exit(1)
		}
		convertOpts.Ramp = ramp
	} else {
		ramp, err := GetRamp(opts.ramp)
		if err != nil {
			logger.Error("无效的梯度预设", "error", err)
			os.Exit(1)
		}
		convertOpts.Ramp = ramp
	}

	// 确定输出目标
	var output *os.File
	if opts.outputPath != "" {
		// 确保输出目录存在
		dir := filepath.Dir(opts.outputPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			logger.Error("无法创建输出目录", "error", err)
			os.Exit(1)
		}
		f, err := os.Create(opts.outputPath)
		if err != nil {
			logger.Error("无法创建输出文件", "error", err)
			os.Exit(1)
		}
		defer f.Close()
		output = f
	} else {
		output = os.Stdout
	}

	// 根据输出模式执行转换与渲染
	if opts.html {
		// HTML 输出模式：始终使用彩色转换
		result, err := ConvertColor(opts.imagePath, convertOpts)
		if err != nil {
			logger.Error("转换失败", "error", err)
			os.Exit(1)
		}
		htmlStr := RenderToHTML(result)
		if _, err := fmt.Fprint(output, htmlStr); err != nil {
			logger.Error("输出失败", "error", err)
			os.Exit(1)
		}
	} else if opts.color {
		// 彩色终端输出
		result, err := ConvertColor(opts.imagePath, convertOpts)
		if err != nil {
			logger.Error("转换失败", "error", err)
			os.Exit(1)
		}
		if err := RenderColorToWriter(output, result); err != nil {
			logger.Error("输出失败", "error", err)
			os.Exit(1)
		}
	} else {
		// 灰度纯文本输出
		result, err := Convert(opts.imagePath, convertOpts)
		if err != nil {
			logger.Error("转换失败", "error", err)
			os.Exit(1)
		}
		if err := RenderToWriter(output, result); err != nil {
			logger.Error("输出失败", "error", err)
			os.Exit(1)
		}
	}
}

// cliOptions 保存解析后的 CLI 参数
type cliOptions struct {
	imagePath   string // 输入图片路径（-f / --file）
	width       int
	color       bool
	invert      bool
	ramp        string
	customRamp  string
	outputPath  string
	html        bool
	aspectRatio float64
}

// parseFlags 解析命令行参数，返回选项。
// 所有参数（包括图片路径）均通过标志位指定，顺序无关。
func parseFlags(args []string) (*cliOptions, error) {
	opts := &cliOptions{}

	fs := flag.NewFlagSet("asciiart", flag.ContinueOnError)

	// 自定义帮助信息：在默认 flag 列表后追加使用样例
	fs.Usage = func() {
		// 打印标准用法头部
		fmt.Fprintf(fs.Output(), "asciiart — 将图片转换为终端字符画\n\n")
		fmt.Fprintf(fs.Output(), "用法: asciiart [选项] -f <图片路径>\n\n")
		fmt.Fprintf(fs.Output(), "选项:\n")
		fs.PrintDefaults()
		fmt.Fprintf(fs.Output(), "\n")
		fmt.Fprintf(fs.Output(), "使用样例:\n")
		fmt.Fprintf(fs.Output(), "\n")
		fmt.Fprintf(fs.Output(), "  # 基本用法：将图片转换为 80 列宽度的灰度字符画\n")
		fmt.Fprintf(fs.Output(), "  asciiart -f photo.png\n")
		fmt.Fprintf(fs.Output(), "\n")
		fmt.Fprintf(fs.Output(), "  # 指定输出宽度为 120 列\n")
		fmt.Fprintf(fs.Output(), "  asciiart -w 120 -f photo.jpg\n")
		fmt.Fprintf(fs.Output(), "\n")
		fmt.Fprintf(fs.Output(), "  # 参数顺序自由，以下两种写法等价\n")
		fmt.Fprintf(fs.Output(), "  asciiart -f photo.png -c -w 120\n")
		fmt.Fprintf(fs.Output(), "  asciiart -c -w 120 -f photo.png\n")
		fmt.Fprintf(fs.Output(), "\n")
		fmt.Fprintf(fs.Output(), "  # 启用真彩色 ANSI 输出（终端需支持 24-bit 颜色）\n")
		fmt.Fprintf(fs.Output(), "  asciiart -c -f photo.png\n")
		fmt.Fprintf(fs.Output(), "\n")
		fmt.Fprintf(fs.Output(), "  # 反转亮度映射（亮色背景适用）\n")
		fmt.Fprintf(fs.Output(), "  asciiart -i -c -f photo.png\n")
		fmt.Fprintf(fs.Output(), "\n")
		fmt.Fprintf(fs.Output(), "  # 使用详细字符梯度（70 级）获得更丰富层次\n")
		fmt.Fprintf(fs.Output(), "  asciiart -r detailed -f photo.png\n")
		fmt.Fprintf(fs.Output(), "\n")
		fmt.Fprintf(fs.Output(), "  # 使用 Unicode 块元素梯度\n")
		fmt.Fprintf(fs.Output(), "  asciiart -r blocks -w 60 -f photo.png\n")
		fmt.Fprintf(fs.Output(), "\n")
		fmt.Fprintf(fs.Output(), "  # 自定义字符梯度（从暗到亮）\n")
		fmt.Fprintf(fs.Output(), "  asciiart --custom-ramp \"@#+=. \" -f photo.png\n")
		fmt.Fprintf(fs.Output(), "\n")
		fmt.Fprintf(fs.Output(), "  # 输出到文件\n")
		fmt.Fprintf(fs.Output(), "  asciiart -o output.txt -f photo.png\n")
		fmt.Fprintf(fs.Output(), "  asciiart -c -o colored.txt -f photo.png\n")
		fmt.Fprintf(fs.Output(), "\n")
		fmt.Fprintf(fs.Output(), "  # 生成 HTML 格式（可在浏览器中查看彩色字符画）\n")
		fmt.Fprintf(fs.Output(), "  asciiart --html -o art.html -f photo.png\n")
		fmt.Fprintf(fs.Output(), "\n")
		fmt.Fprintf(fs.Output(), "  # 调整字符宽高比修正系数（默认 0.5 适配大多数终端字体）\n")
		fmt.Fprintf(fs.Output(), "  asciiart --aspect 0.4 -f photo.png\n")
		fmt.Fprintf(fs.Output(), "\n")
		fmt.Fprintf(fs.Output(), "  # 组合多个选项\n")
		fmt.Fprintf(fs.Output(), "  asciiart -w 100 -c -r detailed -i -f photo.png\n")
		fmt.Fprintf(fs.Output(), "  asciiart -f photo.jpg -w 60 -c --html -o art.html\n")
		fmt.Fprintf(fs.Output(), "\n")
		fmt.Fprintf(fs.Output(), "支持的图片格式: PNG, JPEG, GIF（第一帧）, BMP, WebP\n")
	}

	fs.IntVar(&opts.width, "w", defaultWidth, "输出宽度（字符列数）")
	fs.IntVar(&opts.width, "width", defaultWidth, "输出宽度（字符列数）")

	fs.BoolVar(&opts.color, "c", false, "启用真彩色输出（24-bit ANSI）")
	fs.BoolVar(&opts.color, "color", false, "启用真彩色输出（24-bit ANSI）")

	fs.BoolVar(&opts.invert, "i", false, "反转亮度映射")
	fs.BoolVar(&opts.invert, "invert", false, "反转亮度映射")

	fs.StringVar(&opts.ramp, "r", defaultRamp, "字符梯度预设名 (short/standard/detailed/blocks)")
	fs.StringVar(&opts.ramp, "ramp", defaultRamp, "字符梯度预设名 (short/standard/detailed/blocks)")

	fs.StringVar(&opts.customRamp, "custom-ramp", "", "自定义字符梯度字符串（从暗到亮）")

	fs.StringVar(&opts.outputPath, "o", "", "输出到文件路径（默认 stdout）")
	fs.StringVar(&opts.outputPath, "output", "", "输出到文件路径（默认 stdout）")

	fs.BoolVar(&opts.html, "html", false, "以 HTML 格式输出")

	fs.Float64Var(&opts.aspectRatio, "aspect", defaultAspectRatio, "字符宽高比修正系数")

	fs.StringVar(&opts.imagePath, "f", "", "输入图片路径（必填，支持 PNG/JPEG/GIF/BMP/WebP）")
	fs.StringVar(&opts.imagePath, "file", "", "输入图片路径（必填，支持 PNG/JPEG/GIF/BMP/WebP）")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	// 校验：图片路径为必填参数
	if opts.imagePath == "" {
		return nil, fmt.Errorf("必须通过 -f / --file 指定输入图片路径")
	}

	return opts, nil
}
