// chars_test.go — chars.go 的单元测试
//
// 本文件对字符梯度集模块进行表驱动测试，覆盖：
//   - MapGrayToChar() 边界值 (0/128/255) 及各预设梯度的映射正确性
//   - MapGrayToChar() invert 反转映射
//   - MapGrayToChar() 空梯度保护
//   - GetRamp() 所有预设梯度名称
//   - NewCustomRamp() 正常与空字符串边界情况
package main

import (
	"strings"
	"testing"
)

// TestMapGrayToChar_Boundaries 测试灰度边界值 0、128、255 在标准梯度下的映射
func TestMapGrayToChar_Boundaries(t *testing.T) {
	tests := []struct {
		name   string
		ramp   CharRamp
		gray   uint8
		invert bool
		want   rune
	}{
		// 不反转：灰度 0 → 最暗字符（索引 0），255 → 最亮字符（索引末尾）
		{name: "gray=0 standard", ramp: RampStandard, gray: 0, invert: false, want: '@'},
		{name: "gray=128 standard", ramp: RampStandard, gray: 128, invert: false, want: '+'},
		{name: "gray=255 standard", ramp: RampStandard, gray: 255, invert: false, want: ' '},

		// 反转：灰度 0 → 最亮字符，255 → 最暗字符
		{name: "gray=0 invert", ramp: RampStandard, gray: 0, invert: true, want: ' '},
		{name: "gray=128 invert", ramp: RampStandard, gray: 128, invert: true, want: '='},
		{name: "gray=255 invert", ramp: RampStandard, gray: 255, invert: true, want: '@'},

		// Blocks 梯度 (5 级: █▓▒░ )
		{name: "gray=0 blocks", ramp: RampBlocks, gray: 0, invert: false, want: '█'},
		{name: "gray=255 blocks", ramp: RampBlocks, gray: 255, invert: false, want: ' '},
		{name: "gray=0 blocks invert", ramp: RampBlocks, gray: 0, invert: true, want: ' '},
		{name: "gray=255 blocks invert", ramp: RampBlocks, gray: 255, invert: true, want: '█'},

		// Detailed 梯度 70 级
		{name: "gray=0 detailed", ramp: RampDetailed, gray: 0, invert: false, want: '$'},
		{name: "gray=255 detailed", ramp: RampDetailed, gray: 255, invert: false, want: ' '},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ramp.MapGrayToChar(tt.gray, tt.invert)
			if got != tt.want {
				t.Errorf("MapGrayToChar(%d, %v) = %q, want %q", tt.gray, tt.invert, got, tt.want)
			}
		})
	}
}

// TestMapGrayToChar_EmptyRamp 测试空梯度保护
func TestMapGrayToChar_EmptyRamp(t *testing.T) {
	var emptyRamp CharRamp
	got := emptyRamp.MapGrayToChar(128, false)
	if got != ' ' {
		t.Errorf("空梯度应返回空格，got %q", got)
	}
}

// TestGetRamp 测试预设梯度名称解析
func TestGetRamp(t *testing.T) {
	tests := []struct {
		name      string
		wantFirst rune // 梯度第一个字符（最暗）
		wantLen   int
	}{
		{name: "short", wantFirst: '@', wantLen: 10},
		{name: "SHORT", wantFirst: '@', wantLen: 10}, // 大小写不敏感
		{name: "standard", wantFirst: '@', wantLen: 10},
		{name: "detailed", wantFirst: '$', wantLen: 70},
		{name: "blocks", wantFirst: '█', wantLen: 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ramp, err := GetRamp(tt.name)
			if err != nil {
				t.Fatalf("GetRamp(%q) 返回错误: %v", tt.name, err)
			}
			if len(ramp) != tt.wantLen {
				t.Errorf("GetRamp(%q) 长度 = %d, want %d", tt.name, len(ramp), tt.wantLen)
			}
			if len(ramp) > 0 && ramp[0] != tt.wantFirst {
				t.Errorf("GetRamp(%q) 首字符 = %q, want %q", tt.name, ramp[0], tt.wantFirst)
			}
		})
	}
}

// TestGetRamp_InvalidName 测试无效预设名称
func TestGetRamp_InvalidName(t *testing.T) {
	_, err := GetRamp("nonexistent")
	if err == nil {
		t.Error("GetRamp(\"nonexistent\") 应返回错误")
	}
	if !strings.Contains(err.Error(), "未知的字符梯度预设") {
		t.Errorf("错误信息应包含中文提示，got %q", err.Error())
	}
}

// TestNewCustomRamp 测试自定义梯度创建
func TestNewCustomRamp(t *testing.T) {
	// 正常创建
	ramp, err := NewCustomRamp("ABC")
	if err != nil {
		t.Fatalf("NewCustomRamp(\"ABC\") 返回错误: %v", err)
	}
	if len(ramp) != 3 {
		t.Errorf("NewCustomRamp(\"ABC\") 长度 = %d, want 3", len(ramp))
	}
	if ramp[0] != 'A' || ramp[1] != 'B' || ramp[2] != 'C' {
		t.Errorf("NewCustomRamp(\"ABC\") 内容错误: %v", ramp)
	}

	// 空字符串应返回错误
	_, err = NewCustomRamp("")
	if err == nil {
		t.Error("NewCustomRamp(\"\") 应返回错误")
	}
	if !strings.Contains(err.Error(), "不能为空") {
		t.Errorf("错误信息应包含中文提示，got %q", err.Error())
	}

	// Unicode 字符支持
	ramp, err = NewCustomRamp("暗亮")
	if err != nil {
		t.Fatalf("NewCustomRamp(\"暗亮\") 返回错误: %v", err)
	}
	if len(ramp) != 2 {
		t.Errorf("NewCustomRamp(\"暗亮\") 长度 = %d, want 2", len(ramp))
	}
}

// TestMapGrayToChar_LinearMapping 验证线性映射的灰阶梯次
func TestMapGrayToChar_LinearMapping(t *testing.T) {
	// 在 10 级梯度上验证每个灰阶的映射是单调的
	ramp := RampStandard
	prevIndex := -1
	for gray := 0; gray <= 255; gray += 25 {
		ch := ramp.MapGrayToChar(uint8(gray), false)
		// 查找字符在梯度中的位置
		idx := -1
		for i, r := range ramp {
			if r == ch {
				idx = i
				break
			}
		}
		if idx < prevIndex {
			t.Errorf("灰度 %d 映射到索引 %d (字符 %q)，但前一灰度映射到索引 %d，非单调递增",
				gray, idx, ch, prevIndex)
		}
		prevIndex = idx
	}
}
