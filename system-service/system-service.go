package systemService

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/wueasy/wueasy-go-tools/utils"

	startup_parameter "github.com/wueasy/wueasy-go-tools/startup-parameter"

	config2 "github.com/wueasy/wueasy-go-tools/config"

	antpathmatcher "github.com/wueasy/wueasy-go-tools/ant-path-matcher"

	"github.com/kardianos/service" //garble:no-literal
)

// generateRandomKey 生成指定长度的随机密钥（可打印字符串）
func generateRandomKey(length int) ([]byte, error) {
	// 生成由字母和数字组成的随机字符串
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	key := make([]byte, length)
	for i := 0; i < length; i++ {
		randByte := make([]byte, 1)
		if _, err := rand.Read(randByte); err != nil {
			return nil, err
		}
		key[i] = charset[randByte[0]%byte(len(charset))]
	}
	return key, nil
}

type program struct {
	svc service.Service
}

func (p *program) Start(s service.Service) error {

	// 设置当前实例为服务实例
	// p.svc = s

	// 在这里启动你的程序
	fmt.Println("Service started")
	go p.run()
	return nil
}

var callbackMain func()

func (p *program) run() {
	// Do work here
	callbackMain()
}

var callbackStop func()

func (p *program) Stop(s service.Service) error {
	// 在这里停止你的程序
	// os.Exit(1) // 强制退出程序
	callbackStop()
	return nil
}

func Run(startupParameterConfig startup_parameter.StartupParameterConfig, rootPath string, serviceConfig config2.SystemServiceConfig, encDes3Key string, encSm4Key string, callback func(), stopCallback func()) {

	callbackMain = callback
	callbackStop = stopCallback

	if startupParameterConfig.Version {
		fmt.Println("版本号：" + serviceConfig.Version)
		return
	}

	// 正则表达式验证功能
	if startupParameterConfig.RegexPattern != "" && startupParameterConfig.RegexValue != "" {
		matched, err := utils.RegexMatch(startupParameterConfig.RegexPattern, startupParameterConfig.RegexValue)
		if err != nil {
			fmt.Printf("正则表达式验证失败！错误信息：%v\n", err)
			return
		}
		if matched {
			fmt.Printf("正则表达式验证成功！值 '%s' 匹配模式 '%s'\n", startupParameterConfig.RegexValue, startupParameterConfig.RegexPattern)
		} else {
			fmt.Printf("正则表达式验证失败！值 '%s' 不匹配模式 '%s'\n", startupParameterConfig.RegexValue, startupParameterConfig.RegexPattern)
		}
		return
	}

	if startupParameterConfig.RegexPattern != "" || startupParameterConfig.RegexValue != "" {
		fmt.Println("正则表达式验证需要同时提供 -regex 和 -regexValue 参数")
		return
	}

	// AntPath验证功能
	if startupParameterConfig.AntPathPattern != "" && startupParameterConfig.AntPathValue != "" {
		matched := antpathmatcher.Match(startupParameterConfig.AntPathPattern, startupParameterConfig.AntPathValue)
		if matched {
			fmt.Printf("AntPath验证成功！值 '%s' 匹配模式 '%s'\n", startupParameterConfig.AntPathValue, startupParameterConfig.AntPathPattern)
		} else {
			fmt.Printf("AntPath验证失败！值 '%s' 不匹配模式 '%s'\n", startupParameterConfig.AntPathValue, startupParameterConfig.AntPathPattern)
		}
		return
	}

	if startupParameterConfig.AntPathPattern != "" || startupParameterConfig.AntPathValue != "" {
		fmt.Println("AntPath验证需要同时提供 -antPath 和 -antPathValue 参数")
		return
	}

	// 生成随机密钥功能
	if startupParameterConfig.GenDes3Key || startupParameterConfig.GenSm4Key || startupParameterConfig.GenMixKey {
		des3Key, err := generateRandomKey(24) // DES3需要24字节密钥
		if startupParameterConfig.GenSm4Key {
			des3Key, err = generateRandomKey(16) // SM4需要16字节密钥
		} else if startupParameterConfig.GenMixKey {
			des3Key, err = generateRandomKey(32) // SM4需要16字节密钥
		}

		if err != nil {
			fmt.Printf("生成DES3密钥失败！错误信息：%v\n", err)
			return
		}

		// 确定加密类型，默认为des3
		encType := startupParameterConfig.EncType
		if encType == "" {
			encType = "des3"
		}

		// 根据EncType对密钥进行加密
		var encryptedKey []byte
		var encryptErr error

		if encType == "sm4" {
			encryptedKey, encryptErr = utils.EncryptSM4(des3Key, []byte(encSm4Key))
		} else {
			// 默认使用des3加密
			encryptedKey, encryptErr = utils.Encrypt3DES(des3Key, []byte(encDes3Key))
		}

		if encryptErr != nil {
			fmt.Printf("DES3密钥加密失败！错误信息：%v\n", encryptErr)
			return
		}

		// 将加密后的数据转为base64编码
		data2 := base64.StdEncoding.EncodeToString(encryptedKey)

		encPrefix := "ENCDES3"
		if startupParameterConfig.EncType == "sm4" {
			encPrefix = "ENCSM4"
		}

		if startupParameterConfig.GenDes3Key {
			fmt.Printf("生成的随机DES3密钥： %s(%s)\n", encPrefix, data2)
		}
		if startupParameterConfig.GenSm4Key {
			fmt.Printf("生成的随机SM4密钥： %s(%s)\n", encPrefix, data2)
		}
		if startupParameterConfig.GenMixKey {
			fmt.Printf("生成的随机MIX密钥： %s(%s)\n", encPrefix, data2)
		}
		return
	}

	if startupParameterConfig.EncType != "" && startupParameterConfig.EncValue != "" {

		var encryptedData []byte
		var encryptErr error
		if startupParameterConfig.EncType == "sm4" {
			encryptedData, encryptErr = utils.EncryptSM4([]byte(startupParameterConfig.EncValue), []byte(encSm4Key))
		} else {
			encryptedData, encryptErr = utils.Encrypt3DES([]byte(startupParameterConfig.EncValue), []byte(encDes3Key))
		}
		if encryptErr != nil {
			fmt.Printf("加密失败！错误信息：%v\n", encryptErr)
			return
		}

		// 将加密后的数据转为base64编码
		data2 := base64.StdEncoding.EncodeToString(encryptedData)

		encPrefix := "ENCDES3"
		if startupParameterConfig.EncType == "sm4" {
			encPrefix = "ENCSM4"
		}
		fmt.Printf("加密成功！加密后内容为： %s(%s)\n", encPrefix, data2)

		return
	}

	svcConfig := &service.Config{
		Name:        serviceConfig.Name,
		DisplayName: serviceConfig.DisplayName,
		Description: serviceConfig.Description,
		EnvVars: map[string]string{
			serviceConfig.EnvRootPath: rootPath,
		},
	}

	ss := &program{}
	s, err := service.New(ss, svcConfig)
	if err != nil {
		fmt.Printf("service New failed, err: %v\n", err)
		os.Exit(1)
	}

	if len(os.Args) > 1 {
		serviceAction := os.Args[1]
		switch serviceAction {
		case "install":
			err := s.Install()
			if err != nil {
				fmt.Println("安装服务失败: ", err.Error())
			} else {
				fmt.Println("安装服务成功")
			}
			return
		case "uninstall":
			err := s.Uninstall()
			if err != nil {
				fmt.Println("卸载服务失败: ", err.Error())
			} else {
				fmt.Println("卸载服务成功")
			}
			return
		case "start":
			err := s.Start()
			if err != nil {
				fmt.Println("运行服务失败: ", err.Error())
			} else {
				fmt.Println("运行服务成功")
			}
			return
		case "stop":
			err := s.Stop()
			if err != nil {
				fmt.Println("停止服务失败: ", err.Error())
			} else {
				fmt.Println("停止服务成功")
			}
			return
		case "restart":
			err := s.Restart()
			if err != nil {
				fmt.Println("重启服务失败: ", err.Error())
			} else {
				fmt.Println("重启服务成功")
			}
			return
		case "status":
			status, err := s.Status()
			if err != nil {
				fmt.Println("服务状态获取失败: ", err.Error())
			} else {
				if status == service.StatusStopped {
					fmt.Println("服务已停止")
				} else if status == service.StatusRunning {
					fmt.Println("服务已运行")
				} else if status == service.StatusUnknown {
					fmt.Println("服务状态未知")
				}
			}
			return
		}
	}

	// 默认 运行 Run
	err = s.Run()
	if err != nil {
		fmt.Printf("service Control  failed, err: %v\n", err)
		os.Exit(1)
	}
}
