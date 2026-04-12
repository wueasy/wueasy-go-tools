package nacosClient

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/wueasy/wueasy-go-tools/utils"

	log2 "github.com/wueasy/wueasy-go-tools/log"

	config2 "github.com/wueasy/wueasy-go-tools/config"

	utils2 "github.com/wueasy/wueasy-go-tools/utils"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"gopkg.in/yaml.v3"
)

var (
	configClient config_client.IConfigClient
)

// InitConfig 初始化配置中心并加载配置
func InitConfig(ctx context.Context, encDes3Key string, encSm4Key string, nacosConfig config2.NacosConfig, serverName string, rootPath string, callback ConfigCallback, config interface{}) {
	if !nacosConfig.Config.Enabled {
		log2.Ctx(ctx).Info("nacos配置中心未启用")
		return
	}

	// 初始化配置中心客户端
	InitConfigClient(nacosConfig, rootPath)

	// 获取配置
	content, err := GetConfig(serverName, nacosConfig.Config.Group)
	if err != nil {
		log2.Ctx(ctx).Errorf("获取配置失败: %v", err)
		return
	}

	//更新
	content = UpdateConfig(ctx, encDes3Key, encSm4Key, content, config)

	// 调用配置更新回调
	if callback != nil {
		callback.OnConfigChange(content)
	}

	// 监听配置变化
	if err := ListenConfig(ctx, encDes3Key, encSm4Key, serverName, nacosConfig.Config.Group, callback, config); err != nil {
		log2.Ctx(ctx).Errorf("监听配置失败: %v", err)
		return
	}
}

// InitConfigClient 初始化配置中心客户端
func InitConfigClient(config config2.NacosConfig, rootPath string) {
	// 配置nacos的连接配置
	sc := make([]constant.ServerConfig, 0)
	nacosServerAddrs := strings.Split(config.ServerAddr, ",")
	for _, value := range nacosServerAddrs {
		serverAddrs := strings.Split(value, ":")
		if len(serverAddrs) != 2 {
			log2.Ctx(context.Background()).Error("nacos服务器地址格式错误: ", value)
			continue
		}

		port, err := strconv.ParseInt(serverAddrs[1], 10, 64)
		if err != nil {
			log2.Ctx(context.Background()).Error("nacos端口转换失败: ", err)
			continue
		}

		sc = append(sc, constant.ServerConfig{
			ContextPath: "/nacos",
			IpAddr:      serverAddrs[0],
			Port:        uint64(port),
		})
	}

	if len(sc) == 0 {
		log2.Ctx(context.Background()).Error("没有可用的nacos服务器配置")
		return
	}
	// 获取用户主目录
	userHomeDir, err2 := os.UserHomeDir()
	if err2 != nil {
		log2.Ctx(context.Background()).Errorf("获取用户主目录失败: %v", err2)
		userHomeDir = "." // 使用当前目录作为备选
	}

	// 配置客户端注册在哪里
	cc := constant.ClientConfig{
		NamespaceId:         config.Config.Namespace,
		TimeoutMs:           5000,
		NotLoadCacheAtStart: true,
		Username:            config.Username,
		Password:            config.Password,
		// LogLevel:            "info",
		// LogDir:              filepath.Join(rootPath, "logs", "nacos"),
		LogDir:         "",    // 禁用日志目录
		LogLevel:       "off", // 禁用日志级别（可选）
		AppendToStdout: false, // 不输出到控制台
		CacheDir:       filepath.Join(userHomeDir, ".nacos", "cache"),
	}

	// 创建配置中心客户端
	var err error
	configClient, err = clients.NewConfigClient(vo.NacosClientParam{
		ClientConfig:  &cc,
		ServerConfigs: sc,
	})

	// 设置 Nacos 使用自定义的日志记录器
	logger.SetLogger(log2.Ctx(context.Background()))

	if err != nil {
		log2.Ctx(context.Background()).Error("创建nacos配置中心客户端失败: ", err)
		return
	}

	log2.Ctx(context.Background()).Info("nacos配置中心客户端初始化成功")
}

// GetConfig 获取配置
func GetConfig(dataId, group string) (string, error) {
	if configClient == nil {
		return "", fmt.Errorf("配置中心客户端未初始化")
	}

	content, err := configClient.GetConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  group,
	})
	if err != nil {
		return "", err
	}

	return content, nil
}

func UpdateConfig(ctx context.Context, encDes3Key string, encSm4Key string, configStr string, config interface{}) string {
	if "" != configStr {
		log2.Ctx(ctx).Info("开始更新配置信息...")

		// 解密配置内容
		encryptionConfigs := utils.CreateEncryptionConfigs(encDes3Key, encSm4Key)
		configStr = utils.DecryptEncryptedContent(ctx, configStr, encryptionConfigs)

		// 替换环境变量
		configStr = utils2.ExpandEnv(configStr)

		// log2.Ctx(ctx).Debug(configStr)

		// 使用反射获取config的指针
		configValue := reflect.ValueOf(config)
		if configValue.Kind() != reflect.Ptr {
			log2.Ctx(ctx).Error("config must be a pointer")
			return configStr
		}

		// 获取指针指向的值
		configElem := configValue.Elem()
		if !configElem.CanSet() {
			log2.Ctx(ctx).Error("config is not settable")
			return configStr
		}

		// 创建新的配置对象
		newConfig := reflect.New(configElem.Type()).Interface()

		// 解析配置
		if err := yaml.Unmarshal([]byte(configStr), newConfig); err != nil {
			log2.Ctx(ctx).Errorf("解析新配置失败: %v", err)
			return configStr
		}

		// 更新配置，只更新存在的字段
		newConfigValue := reflect.ValueOf(newConfig).Elem()
		for i := 0; i < newConfigValue.NumField(); i++ {
			field := newConfigValue.Field(i)
			fieldType := field.Type().Kind()

			// 如果是结构体类型,递归更新字段
			if fieldType == reflect.Struct {
				oldField := configElem.Field(i)
				for j := 0; j < field.NumField(); j++ {
					subField := field.Field(j)
					// 如果新配置中的字段不为零值，则更新
					if !subField.IsZero() {
						oldField.Field(j).Set(subField)
					}
				}
			} else {
				// 如果新配置中的字段不为零值，则更新
				if !field.IsZero() {
					configElem.Field(i).Set(field)
				}
			}
		}
		log2.Ctx(ctx).Info("配置更新成功")

		// 获取并更新日志级别
		if configValue.Elem().FieldByName("Log").IsValid() {
			logConfig := configValue.Elem().FieldByName("Log")
			if logConfig.IsValid() && logConfig.FieldByName("Level").IsValid() {
				logLevel := logConfig.FieldByName("Level").String()
				if logLevel != "" {
					log2.UpdateLogLevel(logLevel)
				}
			}

			// 更新日志轮转配置
			maxSizeField := logConfig.FieldByName("MaxSize")
			maxBackupsField := logConfig.FieldByName("MaxBackups")
			maxAgeField := logConfig.FieldByName("MaxAge")

			if maxSizeField.IsValid() || maxBackupsField.IsValid() || maxAgeField.IsValid() {
				var maxSize, maxBackups, maxAge int

				if maxSizeField.IsValid() {
					maxSize = int(maxSizeField.Int())
				}
				if maxBackupsField.IsValid() {
					maxBackups = int(maxBackupsField.Int())
				}
				if maxAgeField.IsValid() {
					maxAge = int(maxAgeField.Int())
				}

				log2.UpdateLogRotation(maxSize, maxBackups, maxAge)
			}
		}

	}
	return configStr
}

// ListenConfig 监听配置变化
func ListenConfig(ctx context.Context, encDes3Key string, encSm4Key string, dataId, group string, callback ConfigCallback, config interface{}) error {
	if configClient == nil {
		return fmt.Errorf("配置中心客户端未初始化")
	}

	// 监听配置变化
	err := configClient.ListenConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  group,
		OnChange: func(namespace, group, dataId, data string) {
			log2.Ctx(ctx).Infof("配置发生变化: namespace=%s, group=%s, dataId=%s", namespace, group, dataId)
			log2.Ctx(ctx).Debugf("新配置内容: %s", data)

			//更新
			UpdateConfig(ctx, encDes3Key, encSm4Key, data, config)

			// 调用配置更新回调
			if callback != nil {
				callback.OnConfigChange(data)
			}
		},
	})

	if err != nil {
		return err
	}

	log2.Ctx(ctx).Infof("开始监听配置: dataId=%s, group=%s", dataId, group)
	return nil
}

// ConfigCallback 配置更新回调接口
type ConfigCallback interface {
	OnConfigChange(configStr string)
}

// CloseClient 关闭nacos客户端
func CloseConfigClient() {
	if configClient != nil {
		configClient.CloseClient()
	}
}
