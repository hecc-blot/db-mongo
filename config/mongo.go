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
