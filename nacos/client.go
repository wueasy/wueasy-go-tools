package nacosClient

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wueasy/wueasy-go-tools/utils"

	log2 "github.com/wueasy/wueasy-go-tools/log"

	config2 "github.com/wueasy/wueasy-go-tools/config"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

var (
	client naming_client.INamingClient

	// 互斥锁，保护并发访问
	mutex sync.RWMutex

	// 服务实例缓存
	serviceInstances   = make(map[string][]model.Instance)       // 服务实例缓存
	subscribedServices = make(map[string]bool)                   // 已订阅的服务
	serviceCallbacks   = make(map[string]func([]model.Instance)) // 服务回调函数

	// 平滑加权轮询的当前权重
	currentWeights = make(map[string]map[string]float64) // map[serviceKey]map[instanceKey]currentWeight

	// 轮询计数器
	roundRobinCounters = make(map[string]*int64) // map[serviceKey]*counter

	// 负载均衡策略配置
	loadBalanceType = "weighted_round_robin" // 默认使用加权轮询

	// 【优化】使用 Go 1.20+ 的本地随机数生成器，替代全局 rand.Seed
	localRand = rand.New(rand.NewSource(time.Now().UnixNano()))
)

// RegisterNacos 注册nacos服务
func RegisterNacos(config config2.NacosConfig, serverName string, rootPath string, serverPort string) {
	if !config.Discovery.Enabled {
		return
	}

	log2.Ctx(context.Background()).Info("正在注册Nacos服务...")

	defer func() {
		if err := recover(); err != nil {
			log2.Ctx(context.Background()).Error("nacos注册失败: ", err)
		}
	}()

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

	// 配置客户端注册在哪里
	cc := constant.ClientConfig{
		NamespaceId:         config.Discovery.Namespace,
		TimeoutMs:           10000,
		NotLoadCacheAtStart: true,
		Username:            config.Username,
		Password:            config.Password,
		// LogLevel:             "info",
		// LogDir:               filepath.Join(rootPath, "logs", "nacos"),

		LogDir:               "",    // 禁用日志目录
		LogLevel:             "off", // 禁用日志级别（可选）
		AppendToStdout:       false, // 不输出到控制台
		CacheDir:             filepath.Join(os.Getenv("HOME"), ".nacos", "cache"),
		UpdateThreadNum:      20,
		UpdateCacheWhenEmpty: true,
		// LogRollingConfig: &constant.ClientLogRollingConfig{
		// 	MaxSize:    100,
		// 	MaxBackups: 10,
		// 	Compress:   true,
		// },
	}

	// 创建服务发现客户端
	var err error
	client, err = clients.NewNamingClient(
		vo.NacosClientParam{
			ClientConfig:  &cc,
			ServerConfigs: sc,
		},
	)

	// 设置 Nacos 使用自定义的日志记录器
	logger.SetLogger(log2.Ctx(context.Background()))

	if err != nil {
		log2.Ctx(context.Background()).Error("创建nacos服务发现客户端失败: ", err)
		return
	}

	// 获取本地IP
	var localIp string
	if config.Discovery.Ip != "" {
		localIp = config.Discovery.Ip
	} else {
		localIp = utils.GetLocalIPv4Address()
	}

	// 获取端口
	port, err := strconv.ParseInt(serverPort, 10, 64)
	if err != nil {
		log2.Ctx(context.Background()).Error("端口转换失败: ", err)
		return
	}
	if config.Discovery.Port != "" {
		port, err = strconv.ParseInt(config.Discovery.Port, 10, 64)
		if err != nil {
			log2.Ctx(context.Background()).Error("配置端口转换失败: ", err)
			return
		}
	}

	// 注册服务实例
	success, err := client.RegisterInstance(vo.RegisterInstanceParam{
		Ip:   localIp,
		Port: uint64(port),
		Weight: func() float64 {
			if config.Discovery.Weight == 0 {
				return 1
			} else {
				return config.Discovery.Weight
			}
		}(),
		Enable:      true,
		Healthy:     true,
		Metadata:    config.Discovery.Metadata,
		ServiceName: serverName,
		GroupName:   config.Discovery.Group,
		Ephemeral:   true,
	})

	if err != nil {
		log2.Ctx(context.Background()).Error("注册nacos服务实例失败: ", err)
		return
	}

	if !success {
		log2.Ctx(context.Background()).Error("注册nacos服务实例失败")
		return
	}

	// 设置负载均衡策略
	if config.Discovery.LoadBalanceType != "" {
		loadBalanceType = config.Discovery.LoadBalanceType
	}

	// 【优化】移除已废弃的 rand.Seed，使用本地随机数生成器
	// Go 1.20+ 不再需要手动调用 Seed

	log2.Ctx(context.Background()).Infof("成功注册nacos服务: %s, 地址: %s:%d, 负载均衡策略: %s", serverName, localIp, port, loadBalanceType)
}

// CloseClient 关闭nacos客户端
func CloseClient() {
	if client != nil {
		client.CloseClient()
	}
}

// ensureServiceSubscribed 确保服务已订阅（抽取的公共订阅逻辑）
func ensureServiceSubscribed(serviceKey string, serviceName string, groupName string) error {
	// 使用读锁检查是否已订阅
	mutex.RLock()
	subscribed := subscribedServices[serviceKey]
	mutex.RUnlock()

	if subscribed {
		return nil
	}

	// 使用写锁进行订阅操作
	mutex.Lock()
	// 双重检查，避免在获取锁的过程中其他goroutine已经完成订阅
	if subscribedServices[serviceKey] {
		mutex.Unlock()
		return nil
	}

	// 释放锁，在锁外进行耗时的网络操作
	mutex.Unlock()

	// 首次获取所有健康实例（耗时操作，在锁外执行）
	instances, err := client.SelectInstances(vo.SelectInstancesParam{
		ServiceName: serviceName,
		GroupName:   groupName,
		Clusters:    []string{"DEFAULT"},
		HealthyOnly: true,
	})
	if err != nil {
		return fmt.Errorf("获取服务[%s]健康实例失败: %v", serviceName, err)
	}

	// 重新获取锁进行缓存更新
	mutex.Lock()
	// 再次检查，确保在网络操作期间没有其他goroutine完成订阅
	if subscribedServices[serviceKey] {
		mutex.Unlock()
		return nil
	}

	// 定义回调函数
	callback := func(services []model.Instance) {
		// 在回调中使用局部锁，减少锁的粒度
		mutex.Lock()
		serviceInstances[serviceKey] = services
		currentWeights[serviceKey] = make(map[string]float64)
		if counter, ok := roundRobinCounters[serviceKey]; ok {
			atomic.StoreInt64(counter, 0)
		}
		mutex.Unlock()
		log2.Ctx(context.Background()).Infof("服务[%s]实例已更新，当前实例数: %d", serviceName, len(services))
	}

	// 保存回调函数
	serviceCallbacks[serviceKey] = callback

	// 释放锁，在锁外进行订阅操作
	mutex.Unlock()

	// 订阅服务变更（耗时操作，在锁外执行）
	err = client.Subscribe(&vo.SubscribeParam{
		ServiceName: serviceName,
		GroupName:   groupName,
		Clusters:    []string{"DEFAULT"},
		SubscribeCallback: func(services []model.Instance, err error) {
			if err != nil {
				log2.Ctx(context.Background()).Errorf("服务[%s]变更通知失败: %v", serviceName, err)
				return
			}
			// 调用回调函数
			mutex.RLock()
			cb, ok := serviceCallbacks[serviceKey]
			mutex.RUnlock()
			if ok {
				cb(services)
			}
		},
	})
	if err != nil {
		return fmt.Errorf("订阅服务[%s]失败: %v", serviceName, err)
	}

	// 重新获取锁，更新缓存状态
	mutex.Lock()
	subscribedServices[serviceKey] = true
	serviceInstances[serviceKey] = instances
	currentWeights[serviceKey] = make(map[string]float64)
	counter := int64(0)
	roundRobinCounters[serviceKey] = &counter
	mutex.Unlock()

	log2.Ctx(context.Background()).Infof("成功订阅服务[%s],[%s]", serviceName, groupName)
	return nil
}

// GetHealthyInstanceWithGroup 获取指定服务的健康实例（支持服务组）
func GetHealthyInstanceWithGroup(serviceName string, groupName string, grayVersion string) (*model.Instance, error) {

	instance, err := tryGetHealthyInstance(serviceName, groupName, grayVersion, nil)
	if err == nil {
		return instance, nil
	}

	log2.Ctx(context.Background()).Warnf("获取服务[%s]实例失败: %v", serviceName, err)

	return nil, fmt.Errorf("获取服务[%s]实例失败", serviceName)
}

// GetHealthyInstanceWithGroupAndMetadata 获取指定服务的健康实例（支持服务组和元数据过滤）
func GetHealthyInstanceWithGroupAndMetadata(serviceName string, groupName string, grayVersion string, metadata map[string]string) (*model.Instance, error) {

	instance, err := tryGetHealthyInstance(serviceName, groupName, grayVersion, metadata)
	if err == nil {
		return instance, nil
	}

	log2.Ctx(context.Background()).Warnf("获取服务[%s]实例失败: %v", serviceName, err)

	return nil, fmt.Errorf("获取服务[%s]实例失败", serviceName)
}

// tryGetHealthyInstance 尝试获取健康实例（优化版本）
func tryGetHealthyInstance(serviceName string, groupName string, grayVersion string, metadata map[string]string) (*model.Instance, error) {
	if client == nil {
		return nil, fmt.Errorf("nacos客户端未初始化")
	}

	// 生成服务标识（使用 strings.Builder 优化字符串拼接）
	var sb strings.Builder
	sb.Grow(len(serviceName) + len(groupName) + 1)
	sb.WriteString(serviceName)
	sb.WriteByte(':')
	sb.WriteString(groupName)
	serviceKey := sb.String()

	// 【优化】使用抽取的公共订阅逻辑
	err := ensureServiceSubscribed(serviceKey, serviceName, groupName)
	if err != nil {
		return nil, err
	}

	// 【优化1】使用读锁直接访问实例切片，避免完整拷贝
	// 只在需要时拷贝指针，而不是整个结构体
	mutex.RLock()
	cachedInstances := serviceInstances[serviceKey]
	instanceCount := len(cachedInstances)
	mutex.RUnlock()

	if instanceCount == 0 {
		return nil, fmt.Errorf("服务[%s]没有可用的健康实例", serviceName)
	}

	// 【优化2】预分配切片容量，避免多次扩容
	// 使用指针切片而不是值切片，减少内存拷贝
	filteredInstances := make([]*model.Instance, 0, instanceCount)

	// 【优化3】合并过滤逻辑，减少循环次数
	hasMetadataFilter := len(metadata) > 0
	hasGrayFilter := grayVersion != ""

	mutex.RLock()
	for i := range cachedInstances {
		instance := &cachedInstances[i]

		// metadata 过滤
		if hasMetadataFilter {
			match := true
			for key, value := range metadata {
				if instanceValue, exists := instance.Metadata[key]; !exists || instanceValue != value {
					match = false
					break
				}
			}
			if !match {
				continue
			}
		}

		// 版本过滤
		if hasGrayFilter {
			// 灰度版本：必须匹配指定版本
			if version, ok := instance.Metadata["version"]; !ok || version != grayVersion {
				continue
			}
		} else {
			// 正常版本：version不存在或不以gray开头
			if version, exists := instance.Metadata["version"]; exists && strings.HasPrefix(version, "gray") {
				continue
			}
		}

		// 通过所有过滤条件
		filteredInstances = append(filteredInstances, instance)
	}
	mutex.RUnlock()

	// 如果找到匹配版本的实例,使用过滤后的实例列表
	if len(filteredInstances) == 0 {
		// 如果没有找到匹配版本的实例,返回错误
		return nil, fmt.Errorf("服务[%s]没有找到可用实例", serviceName)
	}

	// 根据负载均衡策略选择实例（在锁外进行）
	selectedInstance := selectInstanceByLoadBalanceTypeOptimized(serviceKey, filteredInstances)
	if selectedInstance == nil {
		return filteredInstances[0], nil
	}

	return selectedInstance, nil
}

// GetAllHealthyInstances 获取所有健康实例（优化版本）
func GetAllHealthyInstances(serviceName string, groupName string) ([]model.Instance, error) {
	if client == nil {
		return nil, fmt.Errorf("nacos客户端未初始化")
	}

	// 使用 strings.Builder 优化字符串拼接
	var sb strings.Builder
	sb.Grow(len(serviceName) + len(groupName) + 1)
	sb.WriteString(serviceName)
	sb.WriteByte(':')
	sb.WriteString(groupName)
	serviceKey := sb.String()

	// 确保服务已订阅
	err := ensureServiceSubscribed(serviceKey, serviceName, groupName)
	if err != nil {
		return nil, err
	}

	// 使用读锁获取实例缓存
	mutex.RLock()
	cachedInstances := serviceInstances[serviceKey]
	// 【优化】只在需要返回副本时才拷贝，否则直接返回切片引用
	// 如果调用方不会修改返回的切片，可以直接返回引用以提高性能
	instances := make([]model.Instance, len(cachedInstances))
	copy(instances, cachedInstances)
	mutex.RUnlock()

	return instances, nil
}

// UnsubscribeService 取消订阅服务（优化版本）
func UnsubscribeService(serviceName string, groupName string) error {
	if client == nil {
		return fmt.Errorf("nacos客户端未初始化")
	}

	// 使用 strings.Builder 优化字符串拼接
	var sb strings.Builder
	sb.Grow(len(serviceName) + len(groupName) + 1)
	sb.WriteString(serviceName)
	sb.WriteByte(':')
	sb.WriteString(groupName)
	serviceKey := sb.String()

	// 使用读锁检查是否已订阅
	mutex.RLock()
	subscribed := subscribedServices[serviceKey]
	mutex.RUnlock()

	if !subscribed {
		return nil
	}

	// 在锁外进行耗时的取消订阅操作
	err := client.Unsubscribe(&vo.SubscribeParam{
		ServiceName: serviceName,
		GroupName:   groupName,
	})
	if err != nil {
		return fmt.Errorf("取消订阅服务[%s]失败: %v", serviceName, err)
	}

	// 使用写锁清理缓存
	mutex.Lock()
	delete(subscribedServices, serviceKey)
	delete(serviceInstances, serviceKey)
	delete(serviceCallbacks, serviceKey)
	delete(currentWeights, serviceKey)
	delete(roundRobinCounters, serviceKey)
	mutex.Unlock()

	log2.Ctx(context.Background()).Infof("成功取消订阅服务[%s]", serviceName)
	return nil
}

// selectInstanceByLoadBalanceType 根据负载均衡策略选择实例
func selectInstanceByLoadBalanceType(serviceKey string, instances []model.Instance) *model.Instance {
	if len(instances) == 0 {
		return nil
	}

	switch loadBalanceType {
	case "round_robin":
		return selectRoundRobin(serviceKey, instances)
	case "random":
		return selectRandom(instances)
	case "weighted_round_robin":
		return selectWeightedRoundRobin(serviceKey, instances)
	default:
		// 默认使用加权轮询
		return selectWeightedRoundRobin(serviceKey, instances)
	}
}

// selectInstanceByLoadBalanceTypeOptimized 根据负载均衡策略选择实例（优化版本，接受指针切片）
func selectInstanceByLoadBalanceTypeOptimized(serviceKey string, instances []*model.Instance) *model.Instance {
	if len(instances) == 0 {
		return nil
	}

	switch loadBalanceType {
	case "round_robin":
		return selectRoundRobinOptimized(serviceKey, instances)
	case "random":
		return selectRandomOptimized(instances)
	case "weighted_round_robin":
		return selectWeightedRoundRobinOptimized(serviceKey, instances)
	default:
		// 默认使用加权轮询
		return selectWeightedRoundRobinOptimized(serviceKey, instances)
	}
}

// selectRoundRobin 轮询选择
func selectRoundRobin(serviceKey string, instances []model.Instance) *model.Instance {
	if len(instances) == 0 {
		return nil
	}

	// 使用读锁检查计数器是否存在
	mutex.RLock()
	counter, exists := roundRobinCounters[serviceKey]
	mutex.RUnlock()

	if !exists {
		// 使用写锁初始化计数器
		mutex.Lock()
		if _, exists := roundRobinCounters[serviceKey]; !exists {
			newCounter := int64(0)
			roundRobinCounters[serviceKey] = &newCounter
			counter = &newCounter
		} else {
			counter = roundRobinCounters[serviceKey]
		}
		mutex.Unlock()
	}

	// 使用原子操作，无需额外加锁
	index := atomic.AddInt64(counter, 1) % int64(len(instances))
	return &instances[index]
}

// selectRoundRobinOptimized 轮询选择（优化版本，接受指针切片）
func selectRoundRobinOptimized(serviceKey string, instances []*model.Instance) *model.Instance {
	if len(instances) == 0 {
		return nil
	}

	// 使用读锁检查计数器是否存在
	mutex.RLock()
	counter, exists := roundRobinCounters[serviceKey]
	mutex.RUnlock()

	if !exists {
		// 使用写锁初始化计数器
		mutex.Lock()
		if _, exists := roundRobinCounters[serviceKey]; !exists {
			newCounter := int64(0)
			roundRobinCounters[serviceKey] = &newCounter
			counter = &newCounter
		} else {
			counter = roundRobinCounters[serviceKey]
		}
		mutex.Unlock()
	}

	// 使用原子操作，无需额外加锁
	index := atomic.AddInt64(counter, 1) % int64(len(instances))
	return instances[index]
}

// selectRandom 随机选择
func selectRandom(instances []model.Instance) *model.Instance {
	if len(instances) == 0 {
		return nil
	}

	// 【优化】使用本地随机数生成器，避免全局锁竞争
	index := localRand.Intn(len(instances))
	return &instances[index]
}

// selectRandomOptimized 随机选择（优化版本，接受指针切片）
func selectRandomOptimized(instances []*model.Instance) *model.Instance {
	if len(instances) == 0 {
		return nil
	}

	// 【优化】使用本地随机数生成器，避免全局锁竞争
	index := localRand.Intn(len(instances))
	return instances[index]
}

// selectWeightedRoundRobin 加权轮询选择（平滑加权轮询算法）
func selectWeightedRoundRobin(serviceKey string, instances []model.Instance) *model.Instance {
	if len(instances) == 0 {
		return nil
	}

	var selectedInstance *model.Instance
	var maxCurrentWeight float64 = -1
	var selectedInstanceKey string

	// 使用读锁获取当前权重
	mutex.RLock()
	weights, exists := currentWeights[serviceKey]
	if !exists {
		mutex.RUnlock()
		// 如果权重不存在，使用写锁初始化
		mutex.Lock()
		if _, exists := currentWeights[serviceKey]; !exists {
			currentWeights[serviceKey] = make(map[string]float64)
		}
		weights = currentWeights[serviceKey]
		mutex.Unlock()
		mutex.RLock()
	}

	// 复制当前权重到本地变量，减少锁持有时间
	localWeights := make(map[string]float64)
	for k, v := range weights {
		localWeights[k] = v
	}
	mutex.RUnlock()

	// 第一步：更新每个实例的当前权重（在锁外计算）
	for i := range instances {
		instanceKey := fmt.Sprintf("%s:%d", instances[i].Ip, instances[i].Port)
		localWeights[instanceKey] += instances[i].Weight
		if localWeights[instanceKey] > maxCurrentWeight {
			maxCurrentWeight = localWeights[instanceKey]
			selectedInstance = &instances[i]
			selectedInstanceKey = instanceKey
		}
	}

	if selectedInstance == nil {
		// 如果没有选中实例（理论上不会发生），返回第一个实例
		return &instances[0]
	}

	// 第二步：计算总权重（在锁外计算）
	totalWeight := 0.0
	for _, instance := range instances {
		totalWeight += instance.Weight
	}
	localWeights[selectedInstanceKey] -= totalWeight

	// 使用写锁更新权重缓存
	mutex.Lock()
	currentWeights[serviceKey] = localWeights
	mutex.Unlock()

	return selectedInstance
}

// selectWeightedRoundRobinOptimized 加权轮询选择（优化版本，接受指针切片）
func selectWeightedRoundRobinOptimized(serviceKey string, instances []*model.Instance) *model.Instance {
	if len(instances) == 0 {
		return nil
	}

	var selectedInstance *model.Instance
	var maxCurrentWeight float64 = -1
	var selectedInstanceKey string

	// 【优化】使用 strings.Builder 构建 instanceKey，减少内存分配
	var keyBuilder strings.Builder
	keyBuilder.Grow(32) // 预分配足够的空间

	// 使用读锁获取当前权重
	mutex.RLock()
	weights, exists := currentWeights[serviceKey]
	if !exists {
		mutex.RUnlock()
		// 如果权重不存在，使用写锁初始化
		mutex.Lock()
		if _, exists := currentWeights[serviceKey]; !exists {
			currentWeights[serviceKey] = make(map[string]float64, len(instances))
		}
		weights = currentWeights[serviceKey]
		mutex.Unlock()
		mutex.RLock()
	}

	// 【优化】预分配 map 容量，避免扩容
	localWeights := make(map[string]float64, len(weights))
	for k, v := range weights {
		localWeights[k] = v
	}
	mutex.RUnlock()

	// 第一步：更新每个实例的当前权重（在锁外计算）
	totalWeight := 0.0
	for _, instance := range instances {
		// 使用 strings.Builder 构建 key
		keyBuilder.Reset()
		keyBuilder.WriteString(instance.Ip)
		keyBuilder.WriteByte(':')
		keyBuilder.WriteString(strconv.FormatUint(instance.Port, 10))
		instanceKey := keyBuilder.String()

		localWeights[instanceKey] += instance.Weight
		totalWeight += instance.Weight

		if localWeights[instanceKey] > maxCurrentWeight {
			maxCurrentWeight = localWeights[instanceKey]
			selectedInstance = instance
			selectedInstanceKey = instanceKey
		}
	}

	if selectedInstance == nil {
		// 如果没有选中实例（理论上不会发生），返回第一个实例
		return instances[0]
	}

	// 第二步：减去总权重
	localWeights[selectedInstanceKey] -= totalWeight

	// 使用写锁更新权重缓存
	mutex.Lock()
	currentWeights[serviceKey] = localWeights
	mutex.Unlock()

	return selectedInstance
}
