package startup_parameter

import (
	"flag"
)

type StartupParameterConfig struct {
	Version           bool   //是否启用版本
	ConfigPath        string //配置路径
	EncValue          string //加密内容
	EncType           string //加密类型
	GenDes3Key        bool   //是否生成随机DES3密钥
	GenSm4Key         bool   //是否生成随机SM4密钥
	GenMixKey         bool   //是否生成随机MIX密钥
	RegexPattern      string //正则表达式模式
	RegexValue        string //需要验证的值
	AntPathPattern    string //AntPath模式
	AntPathValue      string //需要验证的值
	InstallPlaywright bool   //是否安装Playwright
}

// 启动参数配置
func GetStartupParameter() StartupParameterConfig {
	var v bool
	var configPath string
	var encValue string
	var encType string
	var genDes3Key bool
	var genSm4Key bool
	var genMixKey bool
	var regexPattern string
	var regexValue string
	var antPathPattern string
	var antPathValue string
	var installPlaywright bool

	flag.BoolVar(&v, "v", false, "版本号")
	flag.StringVar(&configPath, "c", "", "配置文件路径")

	flag.StringVar(&encType, "encType", "des3", "加密类型，可选 des3、sm4")
	flag.StringVar(&encValue, "encValue", "", "加密内容")
	flag.BoolVar(&genDes3Key, "genDes3Key", false, "生成随机DES3密钥")
	flag.BoolVar(&genSm4Key, "genSm4Key", false, "生成随机SM4密钥")

	flag.StringVar(&regexPattern, "regex", "", "正则表达式模式")
	flag.StringVar(&regexValue, "regexValue", "", "需要验证的值")
	flag.StringVar(&antPathPattern, "antPath", "", "AntPath模式")
	flag.StringVar(&antPathValue, "antPathValue", "", "需要验证的值")
	flag.BoolVar(&genMixKey, "genMixKey", false, "生成随机MIX密钥")
	flag.BoolVar(&installPlaywright, "installPlaywright", false, "安装Playwright")

	flag.Parse()

	return StartupParameterConfig{
		Version:           v,
		ConfigPath:        configPath,
		EncType:           encType,
		EncValue:          encValue,
		GenDes3Key:        genDes3Key,
		GenSm4Key:         genSm4Key,
		GenMixKey:         genMixKey,
		RegexPattern:      regexPattern,
		RegexValue:        regexValue,
		AntPathPattern:    antPathPattern,
		AntPathValue:      antPathValue,
		InstallPlaywright: installPlaywright,
	}
}
