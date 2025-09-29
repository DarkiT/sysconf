package path

import (
	"os"
	"path/filepath"
	"strings"
)

// Resolver 路径解析器结构体
type Resolver struct {
	basePath string
}

// NewResolver 创建新的路径解析器
func NewResolver() *Resolver {
	return &Resolver{}
}

// Resolve 解析路径的主要逻辑
func (r *Resolver) Resolve(parts ...string) string {
	// 预处理参数
	validParts := r.filterValidParts(parts)

	// 处理空参数情况
	if len(validParts) == 0 {
		return r.getCurrentWorkingDir()
	}

	// 处理第一个路径部分
	firstPart := validParts[0]

	// 根据路径类型进行处理
	switch {
	case filepath.IsAbs(firstPart):
		return r.handleAbsolutePath(validParts)
	case strings.HasPrefix(firstPart, "~"):
		return r.handleHomeDirPath(validParts)
	default:
		return r.handleRelativePath(validParts)
	}
}

// filterValidParts 过滤有效的路径部分
func (r *Resolver) filterValidParts(parts []string) []string {
	if len(parts) == 0 {
		return nil
	}

	// 特殊处理单个空字符串或"."的情况
	if len(parts) == 1 && (parts[0] == "" || parts[0] == ".") {
		return nil
	}

	validParts := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			validParts = append(validParts, part)
		}
	}

	return validParts
}

// getCurrentWorkingDir 获取当前工作目录，带有回退机制
func (r *Resolver) getCurrentWorkingDir() string {
	if pwd, err := os.Getwd(); err == nil {
		return pwd
	}

	// 第一级回退：尝试用户主目录
	if home, err := os.UserHomeDir(); err == nil {
		return home
	}

	// 最后回退：使用相对路径
	return "."
}

// handleAbsolutePath 处理绝对路径
func (r *Resolver) handleAbsolutePath(parts []string) string {
	return filepath.Join(parts...)
}

// handleHomeDirPath 处理以~开头的路径
func (r *Resolver) handleHomeDirPath(parts []string) string {
	firstPart := parts[0]
	home, err := os.UserHomeDir()
	if err != nil {
		// 如果无法获取用户主目录，回退到当前工作目录
		return r.handleRelativePathWithBase(parts, r.getCurrentWorkingDir())
	}

	// 处理不同的~路径格式
	expandedPath := r.expandHomePath(firstPart, home)
	if len(parts) == 1 {
		return expandedPath
	}

	// 组合扩展后的路径和剩余部分
	allParts := make([]string, 0, len(parts))
	allParts = append(allParts, expandedPath)
	allParts = append(allParts, parts[1:]...)

	return filepath.Join(allParts...)
}

// expandHomePath 展开home路径
func (r *Resolver) expandHomePath(path, home string) string {
	if len(path) == 1 {
		// 只有"~"
		return home
	}

	if path[1] == '/' || path[1] == '\\' {
		// "~/xxx" 形式
		return filepath.Join(home, path[2:])
	}

	// "~xxx" 形式，不做特殊处理，直接附加到home
	return filepath.Join(home, path[1:])
}

// handleRelativePath 处理相对路径
func (r *Resolver) handleRelativePath(parts []string) string {
	basePath := r.getCurrentWorkingDir()
	return r.handleRelativePathWithBase(parts, basePath)
}

// handleRelativePathWithBase 基于指定基础路径处理相对路径
func (r *Resolver) handleRelativePathWithBase(parts []string, basePath string) string {
	allParts := make([]string, 0, len(parts)+1)
	allParts = append(allParts, basePath)
	allParts = append(allParts, parts...)

	return filepath.Join(allParts...)
}

// IsValidPath 检查路径是否有效
func (r *Resolver) IsValidPath(path string) bool {
	if path == "" {
		return false
	}

	// 检查是否包含非法字符（根据操作系统而定）
	invalidChars := []string{"\x00"} // null字符在所有系统中都是非法的

	for _, char := range invalidChars {
		if strings.Contains(path, char) {
			return false
		}
	}

	return true
}

// Normalize 规范化路径
func (r *Resolver) Normalize(path string) string {
	if path == "" {
		return ""
	}

	// 清理路径
	cleaned := filepath.Clean(path)

	// 转换为绝对路径（如果需要）
	if !filepath.IsAbs(cleaned) {
		if abs, err := filepath.Abs(cleaned); err == nil {
			return abs
		}
	}

	return cleaned
}
