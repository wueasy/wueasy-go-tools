package log

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	config2 "github.com/wueasy/wueasy-go-tools/config"
)

func TestMain(m *testing.M) {
	rootPath, err := os.MkdirTemp("", "log_sensitive_demo_*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建临时目录失败: %v\n", err)
		os.Exit(1)
	}

	Init(rootPath, config2.LogConfig{
		Level: "debug",
		Sensitive: config2.SensitiveConfig{
			FieldRules: []config2.FieldRule{
				{FieldNames: []string{"mobile", "phone"}, Type: "mobile"},
				{FieldNames: []string{"password", "pwd"}, Type: "password"},
				{FieldNames: []string{"bankcard", "cardno"}, Type: "bankcard"},
				{FieldNames: []string{"email"}, Type: "email"},
				{FieldNames: []string{"idcard", "identity"}, Type: "idcard"},
				{FieldNames: []string{"name", "realname"}, Type: "name"},
			},
			ContentRules: []config2.ContentRule{
				{Type: "mobile"},
				{Type: "email"},
				{Type: "ip"},
			},
		},
	})

	ctx := NewContext(context.Background(), "demo-trace-001")

	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║          日志脱敏演示 (Sensitive Masking Demo)          ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Println()

	fmt.Println("━━━ 1. Infof 消息体脱敏 ━━━")
	Ctx(ctx).Infof("用户%s登录成功，手机号：%s", "张三", "13800138000")

	fmt.Println()
	fmt.Println("━━━ 2. Infow 结构化字段脱敏 ━━━")
	Ctx(ctx).Infow("用户注册",
		"mobile", "13800138000",
		"password", "abc123",
		"bankcard", "6222021234567890123",
		"email", "test@example.com",
		"name", "张三丰",
		"idcard", "110101199001011234",
	)

	fmt.Println()
	fmt.Println("━━━ 3. JSON 响应体自动检测脱敏 ━━━")
	body := `{"code":0,"data":{"user":{"mobile":"13800138000","email":"test@example.com","bankcard":"6222021234567890"},"token":"abc123"},"password":"secret"}`
	Ctx(ctx).Infow("HTTP响应", "response", body)

	fmt.Println()
	fmt.Println("━━━ 4. 纯文本消息脱敏 ━━━")
	Ctx(ctx).Infof("收到短信：验证码123456，来自13800138000，IP地址192.168.1.100")

	fmt.Println()
	fmt.Println("━━━ 5. 无脱敏配置（取消脱敏后的效果）━━━")

	UpdateSensitiveConfig(config2.SensitiveConfig{})
	Ctx(ctx).Infof("关闭脱敏后，手机号明文：%s", "13800138000")

	Sync()

	logFile := filepath.Join(rootPath, "logs", "app.log")
	fmt.Println()
	fmt.Printf("  完整日志文件: %s\n", logFile)

	code := m.Run()

	Sync()
	os.RemoveAll(rootPath)
	os.Exit(code)
}

func TestDemo(t *testing.T) {
	t.Log("脱敏演示完成，请查看上方控制台输出和日志文件")
}
