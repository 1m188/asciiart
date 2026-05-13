// image_test.go — image.go 的单元测试
//
// 本文件测试图片加载与预处理模块，覆盖：
//   - LoadImage() 正常加载 PNG + 文件不存在错误
//   - Preprocess() 输出尺寸正确性 + 边界值保护（width<=0, aspectRatio<=0）
//   - ToGrayscaleMatrix() 灰度值 [0,255] 范围
//   - ToColorMatrix() 颜色矩阵维度正确性
package main

import (
	"image"
	"image/color"
	"testing"
)

// testPNGPath 测试用 PNG 图片路径
const testPNGPath = "test.png"

// TestLoadImage_PNG 测试正常加载 PNG 图片
func TestLoadImage_PNG(t *testing.T) {
	img, err := LoadImage(testPNGPath)
	if err != nil {
		t.Fatalf("LoadImage(%q) 返回错误: %v", testPNGPath, err)
	}
	if img == nil {
		t.Fatal("LoadImage 返回 nil")
	}
	bounds := img.Bounds()
	if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
		t.Errorf("图片尺寸异常: %dx%d", bounds.Dx(), bounds.Dy())
	}
	t.Logf("成功加载图片: %dx%d", bounds.Dx(), bounds.Dy())
}

// TestLoadImage_FileNotFound 测试文件不存在
func TestLoadImage_FileNotFound(t *testing.T) {
	_, err := LoadImage("nonexistent_file.png")
	if err == nil {
		t.Error("LoadImage 对不存在的文件应返回错误")
	}
}

// TestPreprocess_OutputSize 测试预处理输出尺寸正确性
func TestPreprocess_OutputSize(t *testing.T) {
	img, err := LoadImage(testPNGPath)
	if err != nil {
		t.Fatalf("加载测试图片失败: %v", err)
	}

	tests := []struct {
		name        string
		width       int
		aspectRatio float64
		wantWidth   int
	}{
		{name: "default 80 col 0.5 ratio", width: 80, aspectRatio: 0.5, wantWidth: 80},
		{name: "custom 120 col 0.5 ratio", width: 120, aspectRatio: 0.5, wantWidth: 120},
		{name: "custom 40 col 1.0 ratio", width: 40, aspectRatio: 1.0, wantWidth: 40},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gray := Preprocess(img, tt.width, tt.aspectRatio)
			bounds := gray.Bounds()
			if bounds.Dx() != tt.wantWidth {
				t.Errorf("宽度 = %d, want %d", bounds.Dx(), tt.wantWidth)
			}
			if bounds.Dy() < 1 {
				t.Error("高度应 >= 1")
			}
			t.Logf("预处理结果: %dx%d", bounds.Dx(), bounds.Dy())
		})
	}
}

// TestPreprocess_EdgeCases 测试 Preprocess 边界情况
func TestPreprocess_EdgeCases(t *testing.T) {
	img, err := LoadImage(testPNGPath)
	if err != nil {
		t.Fatalf("加载测试图片失败: %v", err)
	}

	// width <= 0 应回退为 80
	gray := Preprocess(img, 0, 0.5)
	if gray.Bounds().Dx() != 80 {
		t.Errorf("width=0 应回退为 80, got %d", gray.Bounds().Dx())
	}

	// aspectRatio <= 0 应回退为 0.5
	gray2 := Preprocess(img, 40, 0)
	if gray2.Bounds().Dy() < 1 {
		t.Error("aspectRatio=0 时高度应 >= 1")
	}

	// width=1 的最小宽度
	gray3 := Preprocess(img, 1, 0.5)
	if gray3.Bounds().Dx() != 1 {
		t.Errorf("width=1 时宽度应为 1, got %d", gray3.Bounds().Dx())
	}
	if gray3.Bounds().Dy() < 1 {
		t.Error("width=1 时高度应 >= 1")
	}
}

// TestPreprocess_GrayValues 验证灰度值在 [0,255] 范围内
func TestPreprocess_GrayValues(t *testing.T) {
	img, err := LoadImage(testPNGPath)
	if err != nil {
		t.Fatalf("加载测试图片失败: %v", err)
	}

	gray := Preprocess(img, 40, 0.5)
	bounds := gray.Bounds()

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			v := gray.GrayAt(x, y).Y
			// *image.Gray 的 Y 值天然在 [0,255]
			_ = v
		}
	}
	// 若能遍历完成即无 panic
}

// TestToGrayscaleMatrix_Dimensions 测试灰度矩阵维度
func TestToGrayscaleMatrix_Dimensions(t *testing.T) {
	img, err := LoadImage(testPNGPath)
	if err != nil {
		t.Fatalf("加载测试图片失败: %v", err)
	}

	bounds := img.Bounds()
	matrix := ToGrayscaleMatrix(img)

	if len(matrix) != bounds.Dy() {
		t.Errorf("矩阵高度 = %d, want %d", len(matrix), bounds.Dy())
	}
	if len(matrix) > 0 && len(matrix[0]) != bounds.Dx() {
		t.Errorf("矩阵宽度 = %d, want %d", len(matrix[0]), bounds.Dx())
	}

	// 验证灰度值范围 [0,255]
	for y := 0; y < len(matrix); y++ {
		for x := 0; x < len(matrix[y]); x++ {
			_ = matrix[y][x] // uint8 天然在 [0,255]
		}
	}
}

// TestToColorMatrix_Dimensions 测试颜色矩阵维度
func TestToColorMatrix_Dimensions(t *testing.T) {
	img, err := LoadImage(testPNGPath)
	if err != nil {
		t.Fatalf("加载测试图片失败: %v", err)
	}

	bounds := img.Bounds()
	matrix := ToColorMatrix(img)

	if len(matrix) != bounds.Dy() {
		t.Errorf("矩阵高度 = %d, want %d", len(matrix), bounds.Dy())
	}
	if len(matrix) > 0 && len(matrix[0]) != bounds.Dx() {
		t.Errorf("矩阵宽度 = %d, want %d", len(matrix[0]), bounds.Dx())
	}
}

// TestPreprocess_PreservesAspectRatio 测试宽高比修正
func TestPreprocess_PreservesAspectRatio(t *testing.T) {
	// 创建一个 200x100 的纯色图片
	src := image.NewRGBA(image.Rect(0, 0, 200, 100))
	// 填充蓝色
	for y := 0; y < 100; y++ {
		for x := 0; x < 200; x++ {
			src.Set(x, y, color.RGBA{R: 0, G: 0, B: 255, A: 255})
		}
	}

	// aspectRatio=0.5: targetWidth=100 → height=100*0.5*100/200=25
	gray := Preprocess(src, 100, 0.5)
	bounds := gray.Bounds()
	if bounds.Dx() != 100 {
		t.Errorf("宽度 = %d, want 100", bounds.Dx())
	}
	if bounds.Dy() != 25 {
		t.Errorf("高度 = %d, want 25 (aspectRatio=0.5, 100*0.5*100/200)", bounds.Dy())
	}

	// aspectRatio=1.0: height=100*1.0*100/200=50
	gray2 := Preprocess(src, 100, 1.0)
	if gray2.Bounds().Dy() != 50 {
		t.Errorf("aspectRatio=1.0 时高度 = %d, want 50", gray2.Bounds().Dy())
	}
}
