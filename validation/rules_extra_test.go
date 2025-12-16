package validation

import "testing"

// 覆盖 Pattern/Enum 等工厂函数和 ValidateStruct 分支
func TestRuleFactoriesAndValidateStruct(t *testing.T) {
	rules := map[string][]ValidationRule{
		"Name": {Required("required"), Pattern("^a.+", "pattern mismatch")},
		"Age":  {Range("1", "10", "range")},
	}

	type sample struct {
		Name string
		Age  int
	}

	err := ValidateStruct(sample{Name: "abc", Age: 5}, rules)
	if err != nil {
		t.Fatalf("valid struct should pass, got %v", err)
	}

	if err := ValidateStruct(sample{Name: "", Age: 5}, rules); err == nil {
		t.Fatalf("required should fail")
	}

	// Min/Max/Length/Enum builder 调用仅需构造
	_ = Min("1", "")
	_ = Max("2", "")
	_ = Length("3", "")
	_ = Enum("a,b", "")
}
