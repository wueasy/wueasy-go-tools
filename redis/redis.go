package redis

import (
	"context"
	"fmt"
	"strings"
	"time"

	log2 "github.com/wueasy/wueasy-go-tools/log"

	"github.com/wueasy/wueasy-go-tools/config"

	"github.com/redis/go-redis/v9"
)

var (
	redisClient redis.UniversalClient
)

//https://redis.uptrace.dev/zh/guide/universal.html

// InitRedis 初始化Redis客户端
func InitRedis(redisConfig config.RedisConfig) {

	ctx := context.Background()
	if redisConfig.Addrs == "" {
		log2.Ctx(ctx).Warn("未配置redis信息")
		return
	}

	// 初始化基本配置
	opts := &redis.UniversalOptions{
		Addrs:    strings.Split(redisConfig.Addrs, ","),
		Password: redisConfig.Password,
		DB:       redisConfig.DB,
		Username: redisConfig.Username,
	}

	if redisConfig.PoolSize > 0 {
		opts.PoolSize = redisConfig.PoolSize
	}
	if redisConfig.MinIdleConns > 0 {
		opts.MinIdleConns = redisConfig.MinIdleConns
	}
	if redisConfig.MaxIdleConns > 0 {
		opts.MaxIdleConns = redisConfig.MaxIdleConns
	}
	if redisConfig.MaxRetries > 0 {
		opts.MaxRetries = redisConfig.MaxRetries
	}
	if redisConfig.MaxRedirects > 0 {
		opts.MaxRedirects = redisConfig.MaxRedirects
	}

	opts.ReadOnly = redisConfig.ReadOnly
	opts.RouteRandomly = redisConfig.RouteRandomly

	if redisConfig.MasterName != "" {
		opts.MasterName = redisConfig.MasterName
	}

	// 初始化超时配置
	if redisConfig.DialTimeout != "" {
		dialTimeout, err := time.ParseDuration(redisConfig.DialTimeout)
		if err != nil {
			log2.Ctx(ctx).Errorf("解析Redis连接超时配置失败: %v", err)
		} else {
			opts.DialTimeout = dialTimeout
		}
	}

	if redisConfig.ReadTimeout != "" {
		readTimeout, err := time.ParseDuration(redisConfig.ReadTimeout)
		if err != nil {
			log2.Ctx(ctx).Errorf("解析Redis读取超时配置失败: %v", err)
		} else {
			opts.ReadTimeout = readTimeout
		}
	}

	if redisConfig.WriteTimeout != "" {
		writeTimeout, err := time.ParseDuration(redisConfig.WriteTimeout)
		if err != nil {
			log2.Ctx(ctx).Errorf("解析Redis写入超时配置失败: %v", err)
		} else {
			opts.WriteTimeout = writeTimeout
		}
	}

	if redisConfig.MinRetryBackoff != "" {
		minRetryBackoff, err := time.ParseDuration(redisConfig.MinRetryBackoff)
		if err != nil {
			log2.Ctx(ctx).Errorf("解析Redis最小重试间隔配置失败: %v", err)
		} else {
			opts.MinRetryBackoff = minRetryBackoff
		}
	}

	if redisConfig.MaxRetryBackoff != "" {
		maxRetryBackoff, err := time.ParseDuration(redisConfig.MaxRetryBackoff)
		if err != nil {
			log2.Ctx(ctx).Errorf("解析Redis最大重试间隔配置失败: %v", err)
		} else {
			opts.MaxRetryBackoff = maxRetryBackoff
		}
	}

	redisClient = redis.NewUniversalClient(opts)

	// 测试连接
	_, err := redisClient.Ping(ctx).Result()
	if err != nil {
		log2.Ctx(ctx).Error("Redis连接失败: ", err)
		return
	}
	log2.Ctx(ctx).Info("Redis连接成功")
}

// GetRedisClient 获取Redis客户端实例
func GetRedisClient() redis.UniversalClient {
	return redisClient
}

// CloseRedis 关闭Redis连接
func CloseRedis() {
	if redisClient != nil {
		err := redisClient.Close()
		ctx := context.Background()
		if err != nil {
			log2.Ctx(ctx).Error("Redis关闭失败: ", err)
			return
		}
		log2.Ctx(ctx).Info("Redis关闭成功")
	}
}

// Set 设置键值对
func Set(ctx context.Context, key string, value string, expiration time.Duration) error {
	return redisClient.Set(ctx, key, value, expiration).Err()
}

// Get 获取键值
// 如果键不存在返回空字符串和redis.Nil错误
func Get(ctx context.Context, key string) (string, error) {
	val, err := redisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	return val, err
}

// Del 删除单个键
func Del(ctx context.Context, key string) error {
	return redisClient.Del(ctx, key).Err()
}

// DelMulti 删除多个键
func DelMulti(ctx context.Context, keys ...string) error {
	return redisClient.Del(ctx, keys...).Err()
}

// Exists 检查键是否存在
// 如果键不存在返回false和nil
// 如果发生错误返回false和错误信息
func Exists(ctx context.Context, key string) (bool, error) {
	n, err := redisClient.Exists(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// Expire 设置键的过期时间
func Expire(ctx context.Context, key string, expiration time.Duration) error {
	return redisClient.Expire(ctx, key, expiration).Err()
}

// TTL 获取键的剩余过期时间
// 如果键不存在返回-2和redis.Nil错误
// 如果键存在但没有过期时间返回-1和nil错误
func GetExpire(ctx context.Context, key string) (time.Duration, error) {
	ttl, err := redisClient.TTL(ctx, key).Result()
	if err == redis.Nil {
		return -2, nil
	}
	return ttl, err
}

// SetIfAbsent 仅当键不存在时设置键值对
// 如果键不存在则设置成功返回true和nil
// 如果键已存在则设置失败返回false和nil
// 如果发生错误返回false和错误信息
func SetIfAbsent(ctx context.Context, key string, value string, expiration time.Duration) (bool, error) {
	result, err := redisClient.SetNX(ctx, key, value, expiration).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return result, nil
}

// Increment 将 key 中储存的数字值增一
// 如果 key 不存在，那么 key 的值会先被初始化为 0 ，然后再执行 INCR 操作
// 如果发生错误返回0和错误信息
func Increment(ctx context.Context, key string) (int64, error) {
	result, err := redisClient.Incr(ctx, key).Result()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return result, nil
}

// NewScript 创建一个新的Redis脚本
func NewScript(script string) *redis.Script {
	return redis.NewScript(script)
}

// RunScript 执行Redis脚本
// 如果脚本执行失败返回nil和错误信息
func RunScript(ctx context.Context, script *redis.Script, keys []string, args ...interface{}) (interface{}, error) {
	result, err := script.Run(ctx, redisClient, keys, args...).Result()
	if err == redis.Nil {
		return nil, nil
	}
	return result, err
}

// RequestRateLimiter 限流器
// business: 业务标识
// rate: 每秒生成令牌数
// capacity: 令牌桶容量
// requested: 请求令牌数
// 返回值:
// - bool: 是否允许请求
// - error: 错误信息
func RequestRateLimiter(ctx context.Context, business string, rate float64, capacity float64, requested float64) ([]interface{}, error) {
	// 构建Redis key，使用hash tags确保在Redis集群中的key路由
	tokensKey := fmt.Sprintf("wueasy:ratelimiter:{%s}.tokens", business)
	timestampKey := fmt.Sprintf("wueasy:ratelimiter:{%s}.timestamp", business)

	// 执行限流脚本
	result, err := RunScript(ctx, requestRateLimiterScript, []string{tokensKey, timestampKey}, rate, capacity, "", requested)
	if err != nil {
		return nil, fmt.Errorf("执行限流脚本失败: %v", err)
	}

	// 解析结果
	resultArray, ok := result.([]interface{})
	if !ok || len(resultArray) != 2 {
		return resultArray, fmt.Errorf("限流脚本返回结果格式错误")
	}
	return resultArray, nil
}

// 限流计数器脚本
var requestRateLimiterScript = redis.NewScript(`
redis.replicate_commands()

local tokens_key = KEYS[1]
local timestamp_key = KEYS[2]
--redis.log(redis.LOG_WARNING, "tokens_key " .. tokens_key)

local rate = tonumber(ARGV[1])
local capacity = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local requested = tonumber(ARGV[4])

local fill_time = capacity / rate
local ttl = math.floor(fill_time * 2)

-- for testing, it should use redis system time in production
if now == nil then
  now = redis.call('TIME')[1]
end

--redis.log(redis.LOG_WARNING, "rate " .. ARGV[1])
--redis.log(redis.LOG_WARNING, "capacity " .. ARGV[2])
--redis.log(redis.LOG_WARNING, "now " .. now)
--redis.log(redis.LOG_WARNING, "requested " .. ARGV[4])
--redis.log(redis.LOG_WARNING, "filltime " .. fill_time)
--redis.log(redis.LOG_WARNING, "ttl " .. ttl)

local last_tokens = tonumber(redis.call("get", tokens_key))
if last_tokens == nil then
  last_tokens = capacity
end
--redis.log(redis.LOG_WARNING, "last_tokens " .. last_tokens)

local last_refreshed = tonumber(redis.call("get", timestamp_key))
if last_refreshed == nil then
  last_refreshed = 0
end
--redis.log(redis.LOG_WARNING, "last_refreshed " .. last_refreshed)

local delta = math.max(0, now-last_refreshed)
local filled_tokens = math.min(capacity, last_tokens+(delta*rate))
local allowed = filled_tokens >= requested
local new_tokens = filled_tokens
local allowed_num = 0
if allowed then
  new_tokens = filled_tokens - requested
  allowed_num = 1
end

--redis.log(redis.LOG_WARNING, "delta " .. delta)
--redis.log(redis.LOG_WARNING, "filled_tokens " .. filled_tokens)
--redis.log(redis.LOG_WARNING, "allowed_num " .. allowed_num)
--redis.log(redis.LOG_WARNING, "new_tokens " .. new_tokens)

if ttl > 0 then
  redis.call("setex", tokens_key, ttl, new_tokens)
  redis.call("setex", timestamp_key, ttl, now)
end

-- return { allowed_num, new_tokens, capacity, filled_tokens, requested, new_tokens }
return { allowed_num, new_tokens }
`)
