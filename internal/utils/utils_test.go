package utils

import (
	"testing"
)

func TestCamelToSnake(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"驼峰转蛇形", "userName", "user_name"},
		{"大写字母", "XMLHttpRequest", "xml_http_request"},
		{"连续大写", "HTTPStatusCode", "http_status_code"},
		{"首字母大写", "UserName", "user_name"},
		{"已是蛇形", "user_name", "user_name"},
		{"单个字母", "a", "a"},
		{"空字符串", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CamelToSnake(tt.in); got != tt.want {
				t.Errorf("camelToSnake() = %v, 期望 %v", got, tt.want)
			}
		})
	}
}

func TestSnakeToCamel(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"蛇形转驼峰", "user_name", "userName"},
		{"多个下划线", "xml_http_request", "xmlHttpRequest"},
		{"首尾下划线", "_user_name_", "userName"},
		{"连续下划线", "user__name", "userName"},
		{"已是驼峰", "userName", "username"},
		{"单个字母", "a", "a"},
		{"空字符串", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SnakeToCamel(tt.in); got != tt.want {
				t.Errorf("snakeToCamel() = %v, 期望 %v", got, tt.want)
			}
		})
	}
}

func TestIsAllLower(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"全小写", "hello", true},
		{"包含大写", "Hello", false},
		{"数字字母", "hello123", true},
		{"特殊字符", "hello_world", true},
		{"空字符串", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsAllLower(tt.in); got != tt.want {
				t.Errorf("isAllLower() = %v, 期望 %v", got, tt.want)
			}
		})
	}
}

func TestSetDefaultValues(t *testing.T) {
	type TestConfig struct {
		String string `default:"默认字符串"`
		Int    int    `default:"42"`
		Bool   bool   `default:"true"`
	}

	config := &TestConfig{}
	SetDefaultValues(config)

	if config.String != "默认字符串" {
		t.Errorf("String 默认值错误，期望=%s, 实际=%s", "默认字符串", config.String)
	}

	if config.Int != 42 {
		t.Errorf("Int 默认值错误，期望=%d, 实际=%d", 42, config.Int)
	}

	if !config.Bool {
		t.Errorf("Bool 默认值错误，期望=%v, 实际=%v", true, config.Bool)
	}
}
