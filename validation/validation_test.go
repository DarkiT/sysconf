package validation

import (
	"testing"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		value   interface{}
		rule    ValidationRule
		wantErr bool
		errMsg  string
	}{
		{
			name:    "必填字段-空字符串",
			value:   "",
			rule:    Required("field cannot be empty"),
			wantErr: true,
			errMsg:  "field cannot be empty",
		},
		{
			name:    "必填字段-非空字符串",
			value:   "test",
			rule:    Required("field cannot be empty"),
			wantErr: false,
		},
		{
			name:    "最小值-数字",
			value:   5,
			rule:    Min("10", "value must be greater than or equal to 10"),
			wantErr: true,
			errMsg:  "value must be greater than or equal to 10",
		},
		{
			name:    "最大值-数字",
			value:   15,
			rule:    Max("10", "value must be less than or equal to 10"),
			wantErr: true,
			errMsg:  "value must be less than or equal to 10",
		},
		{
			name:    "范围-数字",
			value:   5,
			rule:    Range("1", "10", "value must be between 1 and 10"),
			wantErr: false,
		},
		{
			name:    "长度-字符串",
			value:   "test",
			rule:    Length("5", "length must be equal to 5"),
			wantErr: true,
			errMsg:  "length must be equal to 5",
		},
		{
			name:    "枚举-字符串",
			value:   "apple",
			rule:    Enum("apple,banana,orange", "invalid fruit type"),
			wantErr: false,
		},
		{
			name:    "IPv4-有效地址",
			value:   "192.168.1.1",
			rule:    ValidationRule{Type: "ipv4", Message: "invalid IPv4 address"},
			wantErr: false,
		},
		{
			name:    "IPv4-无效地址",
			value:   "256.1.2.3",
			rule:    ValidationRule{Type: "ipv4", Message: "invalid IPv4 address"},
			wantErr: true,
			errMsg:  "invalid IPv4 address",
		},
		{
			name:    "IPv6-有效地址",
			value:   "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
			rule:    ValidationRule{Type: "ipv6", Message: "invalid IPv6 address"},
			wantErr: false,
		},
		{
			name:    "IPv6-无效地址",
			value:   "2001:0db8:85a3",
			rule:    ValidationRule{Type: "ipv6", Message: "invalid IPv6 address"},
			wantErr: true,
			errMsg:  "invalid IPv6 address",
		},
		{
			name:    "端口-有效端口",
			value:   8080,
			rule:    ValidationRule{Type: "port", Message: "invalid port number"},
			wantErr: false,
		},
		{
			name:    "端口-无效端口",
			value:   70000,
			rule:    ValidationRule{Type: "port", Message: "invalid port number"},
			wantErr: true,
			errMsg:  "invalid port number",
		},
		{
			name:    "主机名-有效域名",
			value:   "example.com",
			rule:    ValidationRule{Type: "hostname", Message: "invalid hostname"},
			wantErr: false,
		},
		{
			name:    "主机名-无效域名",
			value:   "example..com",
			rule:    ValidationRule{Type: "hostname", Message: "invalid hostname"},
			wantErr: true,
			errMsg:  "invalid hostname",
		},
		{
			name:    "字母数字-有效",
			value:   "abc123",
			rule:    ValidationRule{Type: "alphanum", Message: "invalid alphanumeric"},
			wantErr: false,
		},
		{
			name:    "字母数字-无效",
			value:   "abc123!@#",
			rule:    ValidationRule{Type: "alphanum", Message: "invalid alphanumeric"},
			wantErr: true,
			errMsg:  "invalid alphanumeric",
		},
		{
			name:    "UUID-有效",
			value:   "123e4567-e89b-12d3-a456-426614174000",
			rule:    ValidationRule{Type: "uuid", Message: "invalid UUID"},
			wantErr: false,
		},
		{
			name:    "UUID-无效",
			value:   "123e4567",
			rule:    ValidationRule{Type: "uuid", Message: "invalid UUID"},
			wantErr: true,
			errMsg:  "invalid UUID",
		},
		{
			name:    "JSON-有效",
			value:   `{"name": "test"}`,
			rule:    ValidationRule{Type: "json", Message: "invalid JSON"},
			wantErr: false,
		},
		{
			name:    "JSON-无效",
			value:   `{"name": test}`,
			rule:    ValidationRule{Type: "json", Message: "invalid JSON"},
			wantErr: true,
			errMsg:  "invalid JSON",
		},
		{
			name:    "Base64-有效",
			value:   "SGVsbG8gV29ybGQ=",
			rule:    ValidationRule{Type: "base64", Message: "invalid Base64"},
			wantErr: false,
		},
		{
			name:    "Base64-无效",
			value:   "SGVsbG8gV29ybGQ==@",
			rule:    ValidationRule{Type: "base64", Message: "invalid Base64"},
			wantErr: true,
			errMsg:  "invalid Base64",
		},
		{
			name:    "DateTime-有效",
			value:   "2024-03-14 15:30:00",
			rule:    ValidationRule{Type: "datetime", Message: "invalid datetime"},
			wantErr: false,
		},
		{
			name:    "DateTime-无效",
			value:   "2024-13-14 15:30:00",
			rule:    ValidationRule{Type: "datetime", Message: "invalid datetime"},
			wantErr: true,
			errMsg:  "invalid datetime",
		},
		{
			name:    "Timezone-有效",
			value:   "Asia/Shanghai",
			rule:    ValidationRule{Type: "timezone", Message: "invalid timezone"},
			wantErr: false,
		},
		{
			name:    "Timezone-无效",
			value:   "Invalid/Timezone",
			rule:    ValidationRule{Type: "timezone", Message: "invalid timezone"},
			wantErr: true,
			errMsg:  "invalid timezone",
		},
		{
			name:    "CreditCard-有效",
			value:   "4532015112830366",
			rule:    ValidationRule{Type: "creditcard", Message: "invalid credit card"},
			wantErr: false,
		},
		{
			name:    "CreditCard-无效",
			value:   "1234567890123456",
			rule:    ValidationRule{Type: "creditcard", Message: "invalid credit card"},
			wantErr: true,
			errMsg:  "invalid credit card",
		},
		{
			name:    "PhoneNumber-有效",
			value:   "+86 123 4567 8901",
			rule:    ValidationRule{Type: "phonenumber", Message: "invalid phone number"},
			wantErr: false,
		},
		{
			name:    "PhoneNumber-无效",
			value:   "123-456",
			rule:    ValidationRule{Type: "phonenumber", Message: "invalid phone number"},
			wantErr: true,
			errMsg:  "invalid phone number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.value, tt.rule)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && err.Error() != tt.errMsg {
				t.Errorf("Validate() error message = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestValidateStruct(t *testing.T) {
	type ServerConfig struct {
		Host     string `json:"host"`
		Port     int    `json:"port"`
		IPv4     string `json:"ipv4"`
		IPv6     string `json:"ipv6"`
		Domain   string `json:"domain"`
		Username string `json:"username"`
		UUID     string `json:"uuid"`
		TimeZone string `json:"timezone"`
	}

	rules := map[string][]ValidationRule{
		"Host": {
			Required("hostname cannot be empty"),
			ValidationRule{Type: "hostname", Message: "invalid hostname"},
		},
		"Port": {
			Required("port cannot be empty"),
			ValidationRule{Type: "port", Message: "invalid port"},
		},
		"IPv4": {
			ValidationRule{Type: "ipv4", Message: "invalid IPv4 address"},
		},
		"IPv6": {
			ValidationRule{Type: "ipv6", Message: "invalid IPv6 address"},
		},
		"Domain": {
			ValidationRule{Type: "hostname", Message: "invalid domain"},
		},
		"Username": {
			Required("username cannot be empty"),
		},
		"UUID": {
			ValidationRule{Type: "uuid", Message: "invalid UUID"},
		},
		"TimeZone": {
			ValidationRule{Type: "timezone", Message: "invalid timezone"},
		},
	}

	tests := []struct {
		name    string
		config  ServerConfig
		wantErr bool
	}{
		{
			name: "有效配置",
			config: ServerConfig{
				Host:     "example.com",
				Port:     8080,
				IPv4:     "192.168.1.1",
				IPv6:     "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
				Domain:   "api.example.com",
				Username: "admin",
				UUID:     "123e4567-e89b-12d3-a456-426614174000",
				TimeZone: "Asia/Shanghai",
			},
			wantErr: false,
		},
		{
			name: "无效配置",
			config: ServerConfig{
				Host:     "",
				Port:     70000,
				IPv4:     "256.1.1.1",
				IPv6:     "invalid",
				Domain:   "example..com",
				Username: "",
				UUID:     "invalid-uuid",
				TimeZone: "Invalid/Zone",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStruct(tt.config, rules)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateStruct() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
