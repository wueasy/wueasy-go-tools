package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

var (
	bundle *i18n.Bundle
	once   sync.Once
	// defaultLang 默认语言
	defaultLang = language.Chinese
)

// Config i18n配置
type Config struct {
	// 语言文件目录
	LocaleDir string
	// 默认语言
	DefaultLang language.Tag
}

// Init 初始化i18n
func Init(config Config) error {
	var err error
	once.Do(func() {
		if config.DefaultLang != language.Und {
			defaultLang = config.DefaultLang
		}

		// 初始化bundle
		bundle = i18n.NewBundle(defaultLang)
		bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

		// 如果配置了语言文件目录，则加载文件
		if config.LocaleDir != "" {
			err = loadMessageFiles(config.LocaleDir)
		}
	})
	return err
}

// loadMessageFiles 加载指定目录下的所有语言文件
func loadMessageFiles(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".json" {
			return nil
		}
		_, err = bundle.LoadMessageFile(path)
		return err
	})
}

// TranslateWithoutData 不带模板数据的翻译消息
func TranslateWithoutData(messageID string, lang string) string {
	return Translate(messageID, lang, nil)
}

// Translate 翻译消息
func Translate(messageID string, lang string, templateData map[string]interface{}) string {
	if bundle == nil {
		return messageID
	}

	// 如果未指定语言，使用默认语言
	if lang == "" {
		lang = defaultLang.String()
	}

	localizer := i18n.NewLocalizer(bundle, lang)
	msg, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID:    messageID,
		TemplateData: templateData,
	})

	if err != nil {
		return messageID
	}
	return msg
}

// T 快捷翻译方法
func T(messageID string, args ...interface{}) string {
	if len(args) == 0 {
		return Translate(messageID, "", nil)
	}

	// 如果只有一个参数且是map类型，则作为模板数据
	if len(args) == 1 {
		if templateData, ok := args[0].(map[string]interface{}); ok {
			return Translate(messageID, "", templateData)
		}
	}

	// 处理格式化参数
	return fmt.Sprintf(Translate(messageID, "", nil), args...)
}

// TL 指定语言的翻译方法
func TL(lang, messageID string, args ...interface{}) string {
	if len(args) == 0 {
		return Translate(messageID, lang, nil)
	}

	// 如果只有一个参数且是map类型，则作为模板数据
	if len(args) == 1 {
		if templateData, ok := args[0].(map[string]interface{}); ok {
			return Translate(messageID, lang, templateData)
		}
	}

	// 处理格式化参数
	return fmt.Sprintf(Translate(messageID, lang, nil), args...)
}

// RegisterMessage 注册单条消息到指定语言
// lang: 语言标识，如 "zh"、"en"
// msgID: 消息ID
// msgText: 消息文本
func RegisterMessage(lang string, msgID string, msgText string) {
	if bundle == nil {
		return
	}
	bundle.AddMessages(language.Make(lang), &i18n.Message{
		ID:    msgID,
		Other: msgText,
	})
}

// RegisterMessages 批量注册消息
// messages: map[语言]map[消息ID]消息文本
func RegisterMessages(messages map[string]map[string]string) {
	if bundle == nil {
		return
	}
	for lang, msgs := range messages {
		for msgID, msgText := range msgs {
			bundle.AddMessages(language.Make(lang), &i18n.Message{
				ID:    msgID,
				Other: msgText,
			})
		}
	}
}
