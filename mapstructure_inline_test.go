package sysconf

import (
	"reflect"
	"testing"

	mapstructure "github.com/go-viper/mapstructure/v2"
	"github.com/stretchr/testify/assert"
)

// 验证多标签 TagName 下 inline/remain 仍可工作
func TestMapstructureInlineWithMultiTagName(t *testing.T) {
	type embedded struct {
		A int `yaml:"a"`
	}
	type cfg struct {
		embedded `yaml:",inline"`
		Name     string         `yaml:"name"`
		Extra    map[string]any `yaml:",remain"`
	}

	input := map[string]any{
		"a":    10,
		"name": "demo",
		"more": "keep",
	}

	var out cfg
	dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName:         "config,yaml",
		SquashTagOption: "inline",
		Result:          &out,
	})
	assert.NoError(t, err)
	t.Logf("embedded tag yaml=%q config=%q", reflect.TypeOf(cfg{}).Field(0).Tag.Get("yaml"), reflect.TypeOf(cfg{}).Field(0).Tag.Get("config"))
	assert.NoError(t, dec.Decode(input))

	assert.Equal(t, 10, out.A)
	assert.Equal(t, "demo", out.Name)
	assert.Equal(t, "keep", out.Extra["more"])
}

// 嵌套 inline + remain 组合，验证多标签（config,yaml）均能展开
func TestMapstructureNestedInlineRemainMultiTag(t *testing.T) {
	type inner struct {
		X string `config:"x" yaml:"x"`
	}

	type middle struct {
		inner `config:",inline" yaml:",inline"`
		Y     int `config:"y" yaml:"y"`
	}

	type outer struct {
		middle `config:",inline" yaml:",inline"`
		Z      bool           `config:"z" yaml:"z"`
		Remain map[string]any `config:",remain" yaml:",remain"`
	}

	input := map[string]any{
		"x":    "hello",
		"y":    42,
		"z":    true,
		"left": "over",
	}

	var out outer
	dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName:         "config,yaml",
		SquashTagOption: "inline",
		Result:          &out,
	})
	assert.NoError(t, err)
	assert.NoError(t, dec.Decode(input))

	assert.Equal(t, "hello", out.X)
	assert.Equal(t, 42, out.Y)
	assert.Equal(t, true, out.Z)
	assert.Equal(t, "over", out.Remain["left"])
}

// remain 收集与 inline 共存时，保证 unused 键进入 remain
func TestMapstructureRemainCollectsUnusedWithMultiTags(t *testing.T) {
	type embed struct {
		Name string `config:"name" yaml:"name"`
	}

	type cfg struct {
		embed  `config:",inline" yaml:",inline"`
		Remain map[string]any `config:",remain" yaml:",remain"`
	}

	input := map[string]any{
		"name": "alice",
		"age":  18,
		"city": "shanghai",
	}

	var out cfg
	dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName:         "config,yaml",
		SquashTagOption: "inline",
		Result:          &out,
	})
	assert.NoError(t, err)
	assert.NoError(t, dec.Decode(input))

	assert.Equal(t, "alice", out.Name)
	assert.Equal(t, 18, out.Remain["age"])
	assert.Equal(t, "shanghai", out.Remain["city"])
}
