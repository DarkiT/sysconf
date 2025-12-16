package validation

import (
	"errors"
	"testing"
)

// DefaultValidator 各规则覆盖
func TestDefaultValidatorRules(t *testing.T) {
	v := NewDefaultValidator()

	cases := []struct {
		name    string
		cfg     map[string]any
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: map[string]any{
				"service": map[string]any{
					"port":    8080,
					"timeout": "5s",
					"url":     "http://example.com",
					"email":   "a@b.com",
					"host":    "example.com",
				},
			},
		},
		{
			name: "invalid port",
			cfg: map[string]any{
				"port": "bad",
			},
			wantErr: true,
		},
		{
			name: "invalid timeout/url/email/host",
			cfg: map[string]any{
				"service": map[string]any{
					"timeout":  "bad",
					"endpoint": "ftp://bad", // url 不以 http/https 开头
					"email":    "no-at",
					"host":     "bad host",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Validate(tt.cfg)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
		})
	}
}

// StructuredValidator 规则与字符串规则覆盖
func TestStructuredValidator(t *testing.T) {
	validator := NewRuleValidator("rule")
	validator.AddRule("db.port", Required("required")).AddStringRule("db.url", "url")

	err := validator.Validate(map[string]any{
		"db": map[string]any{
			"port": 3306,
			"url":  "http://ok",
		},
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	// 缺少必填触发错误
	err = validator.Validate(map[string]any{
		"db": map[string]any{
			"port": "",
			"url":  "http://ok",
		},
	})
	if err == nil {
		t.Fatalf("expected required error")
	}

	// getters
	if !validator.HasRuleForField("db.port") {
		t.Fatalf("HasRuleForField should be true")
	}
	if len(validator.GetRulesForField("db.port")) == 0 {
		t.Fatalf("GetRulesForField empty")
	}
	if len(validator.GetStringRulesForField("db.url")) == 0 {
		t.Fatalf("GetStringRulesForField empty")
	}
	if prefix := extractFieldPrefix("db.port"); prefix != "db" {
		t.Fatalf("extractFieldPrefix got %s", prefix)
	}
}

// CompositeValidator 与 ValidatorFunc 覆盖
func TestCompositeValidator(t *testing.T) {
	one := ValidatorFunc(func(map[string]any) error { return nil })
	two := ValidatorFunc(func(map[string]any) error { return errors.New("boom") })

	cv := NewCompositeValidator("combo", one, two)
	err := cv.Validate(map[string]any{})
	if err == nil || err.Error() == "" {
		t.Fatalf("expected composite error")
	}

	// 添加后再校验成功
	cv = NewCompositeValidator("combo").AddValidator(one)
	if err := cv.Validate(map[string]any{}); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	if name := cv.GetName(); name != "combo" {
		t.Fatalf("GetName mismatch")
	}
}
