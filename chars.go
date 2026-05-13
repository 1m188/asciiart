// chars.go — 字符梯度集定义与映射逻辑
//
// 本文件是 asciiart 项目的基础设施模块，负责定义和管理字符画的核心视觉元素
// —— 字符梯度（Character Ramp）。字符梯度是一组按视觉密度从暗到亮排列的
// 字符集合，用于将图片的灰度值映射为对应的字符表示。
//
// 主要功能：
//   - 定义 CharRamp 类型及其预设梯度（Short / Standard / Detailed / Blocks）
//   - 提供 GetRamp() 按名称获取预设梯度
//   - 提供 NewCustomRamp() 从用户自定义字符串创建梯度
//   - 提供 MapGrayToChar() 将 [0,255] 灰度值线性映射为梯度中的字符
//
// 映射公式：index = gray * (len(ramp) - 1) / 255，invert 为 true 时反转索引。
//
// 本文件无外部依赖，仅使用标准库 fmt 和 strings。
package main

import (
	"fmt"
	"strings"
)

// CharRamp 字符梯度（从暗到亮排列），暗色字符视觉密度高，亮色字符视觉密度低。
type CharRamp []rune

// 预设字符梯度
var (
	// RampShort 短梯度（10 级），从暗到亮
	RampShort CharRamp = []rune("@%#*+=-:. ")

	// RampStandard 标准梯度，等价于 RampShort，常用简单梯度
	RampStandard CharRamp = []rune("@%#*+=-:. ")

	// RampDetailed 详细梯度（70 级），提供更丰富的明暗层次
	RampDetailed CharRamp = []rune("$@B%8&WM#*oahkbdpqwmZO0QLCJUYXzcvunxrjft/\\|()1{}[]?-_+~<>i!lI;:,\"^`'. ")

	// RampBlocks Unicode 块元素梯度（5 级），从暗到亮
	RampBlocks CharRamp = []rune("█▓▒░ ")
)

// rampRegistry 预设梯度名称到梯度的映射
var rampRegistry = map[string]CharRamp{
	"short":    RampShort,
	"standard": RampStandard,
	"detailed": RampDetailed,
	"blocks":   RampBlocks,
}

// GetRamp 根据名称获取预设梯度。支持的名称：short, standard, detailed, blocks。
// 若名称不存在，返回 error。
func GetRamp(name string) (CharRamp, error) {
	ramp, ok := rampRegistry[strings.ToLower(name)]
	if !ok {
		names := make([]string, 0, len(rampRegistry))
		for k := range rampRegistry {
			names = append(names, k)
		}
		return nil, fmt.Errorf("未知的字符梯度预设 %q，支持的预设: %s", name, strings.Join(names, ", "))
	}
	return ramp, nil
}

// NewCustomRamp 从字符串创建自定义梯度。字符从左到右应为从暗到亮排列。
// 若字符串为空，返回 error。
func NewCustomRamp(s string) (CharRamp, error) {
	if s == "" {
		return nil, fmt.Errorf("自定义梯度字符串不能为空")
	}
	return CharRamp([]rune(s)), nil
}

// MapGrayToChar 将灰度值 [0,255] 映射为梯度中的字符。
// 灰度值 0 表示最暗（纯黑），255 表示最亮（纯白）。
// 当 invert 为 true 时反转映射（亮→暗）。
func (r CharRamp) MapGrayToChar(gray uint8, invert bool) rune {
	if len(r) == 0 {
		return ' '
	}
	// 将灰度值 [0,255] 线性映射到梯度索引 [0, len(r)-1]
	index := int(gray) * (len(r) - 1) / 255
	// 反转索引
	if invert {
		index = (len(r) - 1) - index
	}
	return r[index]
}
