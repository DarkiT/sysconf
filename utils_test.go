package sysconf

import (
	"testing"
)

func TestCamelToSnake(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"简单驼峰", "testName", "test_name"},
		{"首字母大写", "TestName", "test_name"},
		{"多个单词", "thisIsATest", "this_is_a_test"},
		{"包含数字", "test123Name", "test123_name"},
		{"全大写", "URL", "url"},
		{"全大写与小写混合", "JSONData", "json_data"},
		{"连续大写后小写", "APIKey", "api_key"},
		{"空字符串", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := camelToSnake(tt.in); got != tt.want {
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
		{"简单下划线", "test_name", "testName"},
		{"多个单词", "this_is_a_test", "thisIsATest"},
		{"包含数字", "test_123_name", "test123Name"},
		{"以下划线开始", "_test_name", "testName"},
		{"以下划线结束", "test_name_", "testName"},
		{"连续下划线", "test__name", "testName"},
		{"全下划线", "___", ""},
		{"空字符串", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := snakeToCamel(tt.in); got != tt.want {
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
		{"全小写", "test", true},
		{"包含大写", "Test", false},
		{"全大写", "TEST", false},
		{"混合大小写", "testName", false},
		{"空字符串", "", true},
		{"包含非字母", "test123", true},
		{"包含非字母和大写", "Test123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isAllLower(tt.in); got != tt.want {
				t.Errorf("isAllLower() = %v, 期望 %v", got, tt.want)
			}
		})
	}
}

type DefaultTestConfig struct {
	String string `default:"默认字符串"`
	Int    int    `default:"42"`
	Bool   bool   `default:"true"`
}

func TestSetDefaultValues(t *testing.T) {
	// 创建一个测试配置
	config := &DefaultTestConfig{}

	// 设置默认值
	setDefaultValues(config)

	// 验证默认值设置
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
