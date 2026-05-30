package log

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	config2 "github.com/wueasy/wueasy-go-tools/config"
)

func resetLogForTest() {
	sugarLogger = nil
	UpdateSensitiveConfig(config2.SensitiveConfig{})
}

func readLogFile(rootPath string) string {
	logFile := filepath.Join(rootPath, "logs", "app.log")
	data, err := os.ReadFile(logFile)
	if err != nil {
		return ""
	}
	return string(data)
}

func setupSensitiveConfig(fieldRules []config2.FieldRule, contentRules []config2.ContentRule, maxLen int) {
	UpdateSensitiveConfig(config2.SensitiveConfig{
		FieldRules:   fieldRules,
		ContentRules: contentRules,
		MaxLength:    maxLen,
	})
}

func resetSensitiveConfig() {
	UpdateSensitiveConfig(config2.SensitiveConfig{})
}

func TestApplyMaskBorder(t *testing.T) {
	cfg := config2.MaskConfig{Strategy: config2.MaskBorder, PrefixKeep: 3, SuffixKeep: 4, MaskChar: "*"}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"mobile", "13800138000", "138****8000"},
		{"short_prefix_suffix_gt_length", "12345", "12345"},
		{"very_short", "12", "12"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyMask(tt.input, cfg)
			if got != tt.expected {
				t.Errorf("applyMask(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestApplyMaskReplace(t *testing.T) {
	cfg := config2.MaskConfig{Strategy: config2.MaskReplace, MaskChar: "*"}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"password", "abc123", "******"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyMask(tt.input, cfg)
			if got != tt.expected {
				t.Errorf("applyMask(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestApplyMaskPrefix(t *testing.T) {
	cfg := config2.MaskConfig{Strategy: config2.MaskPrefix, PrefixKeep: 3, MaskChar: "*"}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"wechat", "wxid_abcdef", "wxi********"},
		{"qq", "12345678", "123*****"},
		{"short", "ab", "ab"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyMask(tt.input, cfg)
			if got != tt.expected {
				t.Errorf("applyMask(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestApplyMaskSuffix(t *testing.T) {
	cfg := config2.MaskConfig{Strategy: config2.MaskSuffix, SuffixKeep: 3, MaskChar: "*"}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"normal", "12345678", "*****678"},
		{"short", "12", "12"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyMask(tt.input, cfg)
			if got != tt.expected {
				t.Errorf("applyMask(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestApplyMaskCustom(t *testing.T) {
	cfg := config2.MaskConfig{Strategy: config2.MaskBorder, PrefixKeep: 2, SuffixKeep: 3, MaskChar: "#"}

	got := applyMask("1234567890", cfg)
	expected := "12#####890"
	if got != expected {
		t.Errorf("applyMask = %q, want %q", got, expected)
	}
}

func TestDesensitizePresetTypes(t *testing.T) {
	defer resetSensitiveConfig()

	tests := []struct {
		name     string
		typ      SensitiveType
		input    string
		expected string
	}{
		{"mobile", Mobile, "13800138000", "138****8000"},
		{"idcard", IDCard, "110101199001011234", "110101********1234"},
		{"bankcard", BankCard, "6222021234567890123", "6222***********0123"},
		{"email", Email, "test@example.com", "tes***@example.com"},
		{"email_short", Email, "a@b.com", "a***@b.com"},
		{"password", Password, "abc123", "******"},
		{"name_3chars", Name, "张三丰", "张*丰"},
		{"name_2chars", Name, "张三", "张三"},
		{"ip", IP, "192.168.1.1", "19*********"},
		{"creditcode", CreditCode, "91110108MA01ABCD2X", "91110108********2X"},
		{"carnumber", CarNumber, "京A12345", "京A1**45"},
		{"wechatid", WeChatID, "wxid_abcdef", "wxi********"},
		{"qq", QQ, "12345678", "1234****"},
		{"empty", Mobile, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Desensitize(tt.input, tt.typ)
			if got != tt.expected {
				t.Errorf("Desensitize(%q, %s) = %q, want %q", tt.input, tt.typ, got, tt.expected)
			}
		})
	}
}

func TestDesensitizeJSONFlat(t *testing.T) {
	defer resetSensitiveConfig()

	setupSensitiveConfig(
		[]config2.FieldRule{
			{FieldNames: []string{"mobile"}, Type: "mobile"},
			{FieldNames: []string{"password"}, Type: "password"},
		}, nil, 0,
	)

	input := `{"mobile":"13800138000","password":"abc123","name":"test"}`
	got := DesensitizeJSON(input)

	if !containsAll(got, `"mobile":"138****8000"`, `"password":"******"`) {
		t.Errorf("DesensitizeJSON = %q", got)
	}

	if !contains(got, `"name":"test"`) {
		t.Errorf("non-sensitive field should not be masked: %q", got)
	}
}

func TestDesensitizeJSONNested(t *testing.T) {
	defer resetSensitiveConfig()

	setupSensitiveConfig(
		[]config2.FieldRule{
			{FieldNames: []string{"mobile"}, Type: "mobile"},
			{FieldNames: []string{"password"}, Type: "password"},
			{FieldNames: []string{"bankcard"}, Type: "bankcard"},
		}, nil, 0,
	)

	input := `{"user":{"mobile":"13800138000","cards":[{"bankcard":"6222021234567890"}]},"password":"abc123"}`
	got := DesensitizeJSON(input)

	cheks := []string{
		`"mobile":"138****8000"`,
		`"password":"******"`,
		`"bankcard":"6222********7890"`,
	}
	for _, c := range cheks {
		if !contains(got, c) {
			t.Errorf("expected %q in result, got %q", c, got)
		}
	}
}

func TestDesensitizeJSONArray(t *testing.T) {
	defer resetSensitiveConfig()

	setupSensitiveConfig(
		[]config2.FieldRule{
			{FieldNames: []string{"mobile"}, Type: "mobile"},
		}, nil, 0,
	)

	input := `[{"mobile":"13800138000"},{"mobile":"13900139000"}]`
	got := DesensitizeJSON(input)

	if !contains(got, `"mobile":"138****8000"`) {
		t.Errorf("array element 1 not masked: %q", got)
	}
	if !contains(got, `"mobile":"139****9000"`) {
		t.Errorf("array element 2 not masked: %q", got)
	}
}

func TestDesensitizeJSONInvalid(t *testing.T) {
	defer resetSensitiveConfig()

	setupSensitiveConfig(
		[]config2.FieldRule{{FieldNames: []string{"mobile"}, Type: "mobile"}},
		nil, 0,
	)

	input := `not valid json`
	got := DesensitizeJSON(input)
	if got != input {
		t.Errorf("invalid JSON should return unchanged, got %q", got)
	}
}

func TestDesensitizeQuery(t *testing.T) {
	defer resetSensitiveConfig()

	setupSensitiveConfig(
		[]config2.FieldRule{
			{FieldNames: []string{"mobile"}, Type: "mobile"},
			{FieldNames: []string{"name"}, Type: "name"},
		}, nil, 0,
	)

	input := "mobile=13800138000&name=%E5%BC%A0%E4%B8%89&page=1"
	got := DesensitizeQuery(input)

	if !contains(got, "138****8000") {
		t.Errorf("expected mobile masked, got %q", got)
	}
	if !contains(got, "page=1") {
		t.Errorf("non-sensitive query param should be preserved, got %q", got)
	}
}

func TestDesensitizeText(t *testing.T) {
	defer resetSensitiveConfig()

	setupSensitiveConfig(nil, []config2.ContentRule{
		{Type: "mobile"},
		{Type: "email"},
		{Type: "qq"},
	}, 0)

	input := "用户13800138000的邮箱test@example.com，QQ号12345678"
	got := DesensitizeText(input)

	cheks := []string{"138****8000", "tes***@example.com", "1234****"}
	for _, c := range cheks {
		if !contains(got, c) {
			t.Errorf("expected %q in result, got %q", c, got)
		}
	}
}

func TestDesensitizeTextNoConfig(t *testing.T) {
	defer resetSensitiveConfig()

	input := "用户13800138000登录"
	got := DesensitizeText(input)
	if got != input {
		t.Errorf("without content-rules config, text should not change: %q", got)
	}
}

func TestDesensitizeTextCustomPattern(t *testing.T) {
	defer resetSensitiveConfig()

	setupSensitiveConfig(nil, []config2.ContentRule{
		{
			Type:    "custom",
			Pattern: `NO\.\d{8}`,
			Mask:    &config2.MaskConfig{Strategy: config2.MaskBorder, PrefixKeep: 5, SuffixKeep: 4, MaskChar: "*"},
		},
	}, 0)

	got := DesensitizeText("订单号NO.12345678已发货")
	if !contains(got, "NO.12**5678") {
		t.Errorf("custom pattern not masked: got %q", got)
	}
}

func TestDesensitizeFieldValue_FieldNameMatch(t *testing.T) {
	defer resetSensitiveConfig()

	setupSensitiveConfig(
		[]config2.FieldRule{
			{FieldNames: []string{"mobile"}, Type: "mobile"},
		}, nil, 0,
	)

	got := desensitizeFieldValue("mobile", "13800138000")
	if got != "138****8000" {
		t.Errorf("desensitizeFieldValue by field name = %q, want 138****8000", got)
	}
}

func TestDesensitizeFieldValue_FieldNameCaseInsensitive(t *testing.T) {
	defer resetSensitiveConfig()

	setupSensitiveConfig(
		[]config2.FieldRule{
			{FieldNames: []string{"mobile"}, Type: "mobile"},
		}, nil, 0,
	)

	got := desensitizeFieldValue("Mobile", "13800138000")
	if got != "138****8000" {
		t.Errorf("case insensitive: got %q, want 138****8000", got)
	}
}

func TestDesensitizeFieldValue_JSONAutoDetect(t *testing.T) {
	defer resetSensitiveConfig()

	setupSensitiveConfig(
		[]config2.FieldRule{
			{FieldNames: []string{"mobile"}, Type: "mobile"},
			{FieldNames: []string{"password"}, Type: "password"},
		}, nil, 0,
	)

	value := `{"user":{"mobile":"13800138000"},"password":"abc123"}`
	got := desensitizeFieldValue("body", value)

	cheks := []string{`"mobile":"138****8000"`, `"password":"******"`}
	for _, c := range cheks {
		if !contains(got, c) {
			t.Errorf("JSON auto-detect: expected %q in result, got %q", c, got)
		}
	}
}

func TestDesensitizeFieldValue_JSONAutoDetectArray(t *testing.T) {
	defer resetSensitiveConfig()

	setupSensitiveConfig(
		[]config2.FieldRule{
			{FieldNames: []string{"mobile"}, Type: "mobile"},
		}, nil, 0,
	)

	value := `[{"mobile":"13800138000"},{"mobile":"13900139000"}]`
	got := desensitizeFieldValue("list", value)

	if !contains(got, `"mobile":"138****8000"`) {
		t.Errorf("array auto-detect failed, got %q", got)
	}
}

func TestDesensitizeFieldValue_ContentRegexFallback(t *testing.T) {
	defer resetSensitiveConfig()

	setupSensitiveConfig(nil, []config2.ContentRule{
		{Type: "mobile"},
	}, 0)

	got := desensitizeFieldValue("unknown_field", "用户13800138000登录")
	if !contains(got, "138****8000") {
		t.Errorf("content regex fallback: expected masked mobile, got %q", got)
	}
}

func TestDesensitizeFieldValue_NoMatch(t *testing.T) {
	defer resetSensitiveConfig()

	setupSensitiveConfig(
		[]config2.FieldRule{
			{FieldNames: []string{"mobile"}, Type: "mobile"},
		}, []config2.ContentRule{}, 0,
	)

	input := "12345"
	got := desensitizeFieldValue("other", input)
	if got != input {
		t.Errorf("unmatched value should be unchanged: got %q", got)
	}
}

func TestLooksLikeJSON(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{`{"a":1}`, true},
		{`[1,2,3]`, true},
		{`  {"a":1}  `, true},
		{`  [1,2,3]  `, true},
		{`not json`, false},
		{``, false},
		{`{`, false},
		{`{}`, true},
		{`[]`, true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := looksLikeJSON(tt.input)
			if got != tt.expected {
				t.Errorf("looksLikeJSON(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestGetMaskConfig_CustomOverridesPreset(t *testing.T) {
	custom := &config2.MaskConfig{Strategy: config2.MaskPrefix, PrefixKeep: 5, MaskChar: "#"}
	got := getMaskConfig("mobile", custom)

	if got.Strategy != config2.MaskPrefix || got.PrefixKeep != 5 || got.MaskChar != "#" {
		t.Errorf("custom mask should override preset, got %+v", got)
	}
}

func TestGetMaskConfig_CustomEmptyStrategy(t *testing.T) {
	custom := &config2.MaskConfig{}
	got := getMaskConfig("mobile", custom)

	if got.Strategy != config2.MaskBorder || got.PrefixKeep != 3 || got.SuffixKeep != 4 {
		t.Errorf("empty custom should fall back to preset, got %+v", got)
	}
}

func TestGetMaskConfig_UnknownType(t *testing.T) {
	got := getMaskConfig("unknown", nil)
	if got.Strategy != config2.MaskReplace || got.MaskChar != "*" {
		t.Errorf("unknown type should default to replace, got %+v", got)
	}
}

func TestIsSensitiveEnabled(t *testing.T) {
	defer resetSensitiveConfig()

	if IsSensitiveEnabled() {
		t.Error("should be false with no config")
	}

	setupSensitiveConfig([]config2.FieldRule{
		{FieldNames: []string{"a"}, Type: "mobile"},
	}, nil, 0)

	if !IsSensitiveEnabled() {
		t.Error("should be true with field rules")
	}

	resetSensitiveConfig()
	setupSensitiveConfig(nil, []config2.ContentRule{{Type: "mobile"}}, 0)

	if !IsSensitiveEnabled() {
		t.Error("should be true with content rules")
	}
}

func TestDesensitizeJSONMaxLength(t *testing.T) {
	defer resetSensitiveConfig()

	setupSensitiveConfig(nil, nil, 10)

	input := `{"data":"this is a very long string that exceeds the limit"}`
	got := DesensitizeJSON(input)

	if len(got) >= len(input) {
		t.Errorf("long string should be truncated, got %q", got)
	}
}

func TestDesensitizeJSON2NoTruncate(t *testing.T) {
	defer resetSensitiveConfig()

	setupSensitiveConfig(nil, nil, 10)

	input := `{"data":"this is a very long string that exceeds the limit"}`
	got := DesensitizeJSON2(input)

	if got != input {
		t.Errorf("DesensitizeJSON2 should not truncate, got %q", got)
	}
}

// ========================
// 集成测试：log.Init() 写入日志文件，读文件验证
// ========================

func tempLogDir(t *testing.T) string {
	rootPath, err := os.MkdirTemp("", "log_sensitive_test_*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		Sync()
		os.RemoveAll(rootPath)
	})
	return rootPath
}

func TestLogFile_InfofMasking(t *testing.T) {
	rootPath := tempLogDir(t)
	resetLogForTest()

	Init(rootPath, config2.LogConfig{
		Level: "debug",
		Sensitive: config2.SensitiveConfig{
			FieldRules: []config2.FieldRule{
				{FieldNames: []string{"mobile"}, Type: "mobile"},
				{FieldNames: []string{"password"}, Type: "password"},
			},
			ContentRules: []config2.ContentRule{
				{Type: "mobile"},
			},
		},
	})

	ctx := NewContext(context.Background(), "trace-001")
	Ctx(ctx).Infof("用户%s登录，手机号%s", "张三", "13800138000")
	Sync()

	content := readLogFile(rootPath)
	if !contains(content, "138****8000") {
		t.Errorf("Infof 消息中手机号应脱敏，日志内容:\n%s", content)
	}
	if !contains(content, "张三") {
		t.Errorf("非敏感内容不应脱敏，日志内容:\n%s", content)
	}
}

func TestLogFile_StructuredFieldMasking(t *testing.T) {
	rootPath := tempLogDir(t)
	resetLogForTest()

	Init(rootPath, config2.LogConfig{
		Level: "debug",
		Sensitive: config2.SensitiveConfig{
			FieldRules: []config2.FieldRule{
				{FieldNames: []string{"mobile"}, Type: "mobile"},
				{FieldNames: []string{"password"}, Type: "password"},
			},
		},
	})

	ctx := NewContext(context.Background(), "trace-001")
	Ctx(ctx).Infow("用户操作",
		"mobile", "13800138000",
		"password", "abc123",
		"name", "张三",
	)
	Sync()

	content := readLogFile(rootPath)
	if !contains(content, "138****8000") {
		t.Errorf("结构化字段 mobile 应脱敏，日志内容:\n%s", content)
	}
	if !contains(content, "******") {
		t.Errorf("结构化字段 password 应脱敏，日志内容:\n%s", content)
	}
	if !contains(content, "张三") {
		t.Errorf("非敏感字段 name 应保留，日志内容:\n%s", content)
	}
}

func TestLogFile_JSONBodyAutoDetect(t *testing.T) {
	rootPath := tempLogDir(t)
	resetLogForTest()

	Init(rootPath, config2.LogConfig{
		Level: "debug",
		Sensitive: config2.SensitiveConfig{
			FieldRules: []config2.FieldRule{
				{FieldNames: []string{"mobile"}, Type: "mobile"},
				{FieldNames: []string{"password"}, Type: "password"},
			},
		},
	})

	ctx := NewContext(context.Background(), "trace-001")
	body := `{"user":{"mobile":"13800138000"},"password":"abc123"}`
	Ctx(ctx).Infow("请求响应", "body", body)
	Sync()

	content := readLogFile(rootPath)
	if !contains(content, "138****8000") {
		t.Errorf("JSON body 字段应自动检测并脱敏 mobile，日志内容:\n%s", content)
	}
	if !contains(content, "******") {
		t.Errorf("JSON body 字段应自动检测并脱敏 password，日志内容:\n%s", content)
	}
}

func TestLogFile_PlainTextMasking(t *testing.T) {
	rootPath := tempLogDir(t)
	resetLogForTest()

	Init(rootPath, config2.LogConfig{
		Level: "debug",
		Sensitive: config2.SensitiveConfig{
			ContentRules: []config2.ContentRule{
				{Type: "mobile"},
				{Type: "email"},
			},
		},
	})

	ctx := NewContext(context.Background(), "trace-001")
	Ctx(ctx).Infof("用户13800138000的邮箱是test@example.com")
	Sync()

	content := readLogFile(rootPath)
	if !contains(content, "138****8000") {
		t.Errorf("纯文本消息中手机号应脱敏，日志内容:\n%s", content)
	}
	if !contains(content, "tes***@example.com") {
		t.Errorf("纯文本消息中邮箱应脱敏，日志内容:\n%s", content)
	}
}

func TestLogFile_WithoutSensitiveConfig(t *testing.T) {
	rootPath := tempLogDir(t)
	resetLogForTest()

	Init(rootPath, config2.LogConfig{
		Level: "debug",
	})

	ctx := NewContext(context.Background(), "trace-001")
	Ctx(ctx).Infof("用户13800138000登录")
	Sync()

	content := readLogFile(rootPath)
	if !contains(content, "13800138000") {
		t.Errorf("未配置脱敏时不应修改内容，日志内容:\n%s", content)
	}
	if contains(content, "****") {
		t.Errorf("未配置脱敏时不应出现掩码，日志内容:\n%s", content)
	}
}

func TestLogFile_MixedScenario(t *testing.T) {
	rootPath := tempLogDir(t)
	resetLogForTest()

	Init(rootPath, config2.LogConfig{
		Level: "debug",
		Sensitive: config2.SensitiveConfig{
			FieldRules: []config2.FieldRule{
				{FieldNames: []string{"mobile"}, Type: "mobile"},
				{FieldNames: []string{"password"}, Type: "password"},
				{FieldNames: []string{"bankcard"}, Type: "bankcard"},
			},
			ContentRules: []config2.ContentRule{
				{Type: "mobile"},
				{Type: "ip"},
			},
			MaxLength: 50,
		},
	})

	ctx := NewContext(context.Background(), "trace-001")
	Ctx(ctx).Infof("1. Infof 消息体: 手机号%s, 姓名%s", "13800138000", "张三")
	Ctx(ctx).Infow("2. 结构化字段",
		"mobile", "13800138000",
		"password", "abc123",
		"bankcard", "6222021234567890",
	)
	Ctx(ctx).Infow("3. JSON 响应体(自动检测)",
		"response", `{"code":0,"data":{"mobile":"13800138000","bankcard":"6222021234567890"},"sign":"abc"}`,
	)
	Ctx(ctx).Infof("4. 纯文本: IP=192.168.1.100, 手机=13900139000")
	Sync()

	content := readLogFile(rootPath)

	t.Logf("日志文件内容:\n%s", content)

	checks := []string{
		"138****8000",
		"张三",
		"******",
		"6222********7890",
	}
	for _, c := range checks {
		if !contains(content, c) {
			t.Errorf("日志中应包含 %q，日志内容:\n%s", c, content)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func containsAll(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if !contains(s, sub) {
			return false
		}
	}
	return true
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
