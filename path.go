package sysconf

import (
	"os"
	"path/filepath"
	"strings"
)

// WorkPath 获取工作目录
// - 空参数或 "." 返回当前工作目录
// - 绝对路径直接返回
// - 以 "~" 开头的路径展开为用户主目录
// - 相对路径基于当前工作目录展开
func WorkPath(parts ...string) string {
	// 初始化基础路径
	var basePath string

	// 如果没有参数或所有参数为空，返回当前工作目录
	if len(parts) == 0 || (len(parts) == 1 && (parts[0] == "" || parts[0] == ".")) {
		// 获取当前工作目录
		if pwd, err := os.Getwd(); err == nil {
			return pwd
		}
		// 失败时尝试用户主目录
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
		// 最后使用相对路径
		return "."
	}

	// 过滤空部分
	validParts := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			validParts = append(validParts, part)
		}
	}

	// 如果没有有效部分，返回当前工作目录
	if len(validParts) == 0 {
		if pwd, err := os.Getwd(); err == nil {
			return pwd
		}
		return "."
	}

	// 处理第一个部分
	firstPart := validParts[0]

	// 绝对路径直接使用
	if filepath.IsAbs(firstPart) {
		return filepath.Join(validParts...)
	}

	// 处理 "~" 开头的路径
	if strings.HasPrefix(firstPart, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			// 如果无法获取用户主目录，尝试使用当前目录
			if pwd, err := os.Getwd(); err == nil {
				basePath = pwd
			} else {
				basePath = "."
			}
		} else {
			// 替换 "~" 为用户主目录
			if len(firstPart) > 1 {
				if firstPart[1] == '/' || firstPart[1] == '\\' {
					// 如果是 "~/xxx" 形式
					validParts[0] = filepath.Join(home, firstPart[2:])
				} else {
					// 如果是 "~xxx" 形式，不做特殊处理
					validParts[0] = filepath.Join(home, firstPart[1:])
				}
			} else {
				// 如果只是 "~"
				validParts[0] = home
			}
		}
	} else {
		// 相对路径基于当前工作目录
		if pwd, err := os.Getwd(); err == nil {
			basePath = pwd
		} else {
			basePath = "."
		}
	}

	// 如果是以 "~" 开头且已处理过的路径，直接返回
	if strings.HasPrefix(firstPart, "~") {
		if len(validParts) == 1 {
			return validParts[0]
		}
		return filepath.Join(validParts...)
	}

	// 构建最终路径
	return filepath.Join(append([]string{basePath}, validParts...)...)
}
