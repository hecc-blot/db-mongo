package config

// MongoConfig MongoDB 连接配置。
// Uri 非空时优先使用完整连接串，否则由 Ip/Port/Username/Password/Database 组装。
type MongoConfig struct {
	Uri      string `mapstructure:"uri"`
	Ip       string `mapstructure:"ip"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`

	AuthSource      string `mapstructure:"auth_source"`        // 认证库，默认 admin
	MaxPoolSize     uint64 `mapstructure:"max_pool_size"`      // 连接池最大连接数，0 用驱动默认
	MinPoolSize     uint64 `mapstructure:"min_pool_size"`      // 连接池最小连接数，0 用驱动默认
	MaxConnIdleTime int    `mapstructure:"max_conn_idle_time"` // 连接最大空闲时间（秒），0 不限制
	ConnectTimeout  int    `mapstructure:"connect_timeout"`    // 建连/Ping 超时（秒），0 用默认 10s
	SlowThreshold   int    `mapstructure:"slow_threshold"`     // 慢查询阈值（毫秒），0 不记录
}

// DefaultConnectTimeout 默认建连/Ping 超时（秒）。
const DefaultConnectTimeout = 10

// Normalize 补全默认值，供构造 MongoDB 实例时调用。
//
// 只回填「0 值不代表关闭」的字段：ConnectTimeout 的 0 值语义是「用默认 10s」，
// 必须显式回填。其余字段的 0 值语义是正确的，保持原样 —— MaxPoolSize/MinPoolSize
// 为「用驱动默认」，MaxConnIdleTime 为「不限制」，SlowThreshold 为「不记录」。
func Normalize(cfg MongoConfig) MongoConfig {
	if cfg.ConnectTimeout <= 0 {
		cfg.ConnectTimeout = DefaultConnectTimeout
	}
	return cfg
}
