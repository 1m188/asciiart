// image.go — 图片加载、解码与预处理
//
// 本文件负责 asciiart 项目中所有图片输入相关的操作，是数据流的第一环节。
// 从磁盘读取图片文件、解码为 Go 标准 image.Image 接口、缩放至目标尺寸、
// 灰度化处理，以及提取颜色信息——所有图片数据的准备和预处理均在此完成。
//
// 主要功能：
//   - LoadImage()      : 从文件路径加载图片，支持 PNG / JPEG / GIF / BMP / WebP
//   - Preprocess()     : 将图片缩放至目标宽度、按字符宽高比修正高度，输出 *image.Gray
//   - ToGrayscaleMatrix(): 将 image.Image 转换为二维 [0,255] 灰度值矩阵
//   - ToColorMatrix()  : 将 image.Image 转换为二维 color.RGBA 颜色矩阵
//
// 技术细节：
//   - 灰度化采用 ITU-R BT.601 亮度公式：Gray = 0.299*R + 0.587*G + 0.114*B
//   - 缩放使用 golang.org/x/image/draw.ApproxBiLinear 双线性插值
//   - GIF 仅解码第一帧，透明像素视为白色背景（R=G=B=255）
//   - WebP/BMP 支持依赖 golang.org/x/image 扩展库
//
// 外部依赖：golang.org/x/image（WebP/BMP 解码 + 高质量缩放）
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/bmp"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // 注册 WebP 解码器
)

// LoadImage 从文件路径加载图片，支持 PNG / JPEG / GIF / BMP / WebP 等格式。
// GIF 仅返回第一帧。
func LoadImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("无法打开图片文件 %q: %w", path, err)
	}
	defer f.Close()

	// 尝试根据扩展名推测格式，同时保留通用回退
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".png":
		img, err := png.Decode(f)
		if err != nil {
			return nil, fmt.Errorf("解码 PNG 图片 %q 失败: %w", path, err)
		}
		return img, nil
	case ".jpg", ".jpeg":
		img, err := jpeg.Decode(f)
		if err != nil {
			return nil, fmt.Errorf("解码 JPEG 图片 %q 失败: %w", path, err)
		}
		return img, nil
	case ".gif":
		img, err := gif.Decode(f)
		if err != nil {
			return nil, fmt.Errorf("解码 GIF 图片 %q 失败: %w", path, err)
		}
		return img, nil
	case ".bmp":
		img, err := bmp.Decode(f)
		if err != nil {
			return nil, fmt.Errorf("解码 BMP 图片 %q 失败: %w", path, err)
		}
		return img, nil
	case ".webp":
		// webp 解码器已在 init() 中注册到 image 包
		img, _, err := image.Decode(f)
		if err != nil {
			return nil, fmt.Errorf("解码 WebP 图片 %q 失败: %w", path, err)
		}
		return img, nil
	default:
		// 未知扩展名，尝试通用解码（image.Decode 会基于内容嗅探格式）
		img, _, err := image.Decode(f)
		if err != nil {
			return nil, fmt.Errorf("无法识别图片格式 %q: %w", path, err)
		}
		return img, nil
	}
}

// Preprocess 将图片缩放至目标宽度并转换为灰度图。
// aspectRatio 为字符宽高比修正系数（终端字符通常高约为宽的 2 倍，默认 0.5）。
// 返回 *image.Gray 灰度图。
func Preprocess(img image.Image, targetWidth int, aspectRatio float64) *image.Gray {
	if targetWidth <= 0 {
		targetWidth = 80
	}
	if aspectRatio <= 0 {
		aspectRatio = 0.5
	}

	bounds := img.Bounds()
	iw := bounds.Dx()
	ih := bounds.Dy()

	// 计算目标高度：保持宽高比，乘以字符宽高比修正系数
	targetHeight := int(float64(targetWidth) * aspectRatio * float64(ih) / float64(iw))
	if targetHeight < 1 {
		targetHeight = 1
	}

	// 缩放至目标尺寸
	scaled := image.NewGray(image.Rect(0, 0, targetWidth, targetHeight))
	draw.ApproxBiLinear.Scale(scaled, scaled.Bounds(), img, img.Bounds(), draw.Over, nil)

	return scaled
}

// ToGrayscaleMatrix 将 image.Image 转换为二维灰度值矩阵 [0,255]。
// 使用 ITU-R BT.601 亮度公式：Gray = 0.299*R + 0.587*G + 0.114*B。
func ToGrayscaleMatrix(img image.Image) [][]uint8 {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	matrix := make([][]uint8, h)
	for y := 0; y < h; y++ {
		row := make([]uint8, w)
		for x := 0; x < w; x++ {
			r, g, b, _ := img.At(x+bounds.Min.X, y+bounds.Min.Y).RGBA()
			// RGBA() 返回 [0, 65535] 范围的值，需要缩放至 [0, 255]
			rr := uint8(r >> 8)
			gg := uint8(g >> 8)
			bb := uint8(b >> 8)
			// ITU-R BT.601 亮度公式
			gray := uint8(0.299*float64(rr) + 0.587*float64(gg) + 0.114*float64(bb))
			row[x] = gray
		}
		matrix[y] = row
	}
	return matrix
}

// ToColorMatrix 将 image.Image 转换为二维 RGB 颜色矩阵。
func ToColorMatrix(img image.Image) [][]color.RGBA {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	matrix := make([][]color.RGBA, h)
	for y := 0; y < h; y++ {
		row := make([]color.RGBA, w)
		for x := 0; x < w; x++ {
			r, g, b, a := img.At(x+bounds.Min.X, y+bounds.Min.Y).RGBA()
			// 透明像素视为白色背景
			if a == 0 {
				row[x] = color.RGBA{R: 255, G: 255, B: 255, A: 255}
			} else {
				row[x] = color.RGBA{
					R: uint8(r >> 8),
					G: uint8(g >> 8),
					B: uint8(b >> 8),
					A: 255,
				}
			}
		}
		matrix[y] = row
	}
	return matrix
}
