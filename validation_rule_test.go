package sysconf

import (
	"testing"

	"github.com/darkit/sysconf/validation"
)

func TestRuleValidatorMapping(t *testing.T) {
	cases := []struct {
		value any
		rule  string
		ok    bool
	}{
		{5, "range:1,10", true},
		{"abc123", "regex:^abc\\d+$", true},
		{"banana", "enum:apple,banana", true},
		{"bad", "regex:^abc\\d+$", false},
	}
	for _, tt := range cases {
		ok, _ := validation.ValidateValue(tt.value, tt.rule)
		if ok != tt.ok {
			t.Fatalf("rule %s expected %v", tt.rule, tt.ok)
		}
	}
}

func TestCompositeValidator(t *testing.T) {
	validator := validation.NewRuleValidator("compose")
	validator.AddRule("server.port", validation.Range("1", "10", "range"))
	validator.AddStringRule("server.host", "required")

	config := map[string]any{
		"server": map[string]any{
			"port": 5,
			"host": "example.com",
		},
	}
	if err := validator.Validate(config); err != nil {
		t.Fatalf("expected composite validator pass, got %v", err)
	}

	config["server"].(map[string]any)["port"] = 50
	if err := validator.Validate(config); err == nil {
		t.Fatalf("expected composite validator fail on range")
	}
}
