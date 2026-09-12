package service

import (
	"testing"

	dbMongoConf "github.com/hecc-blot/db-mongo/config"

	"github.com/stretchr/testify/assert"
)

func TestBuildUri(t *testing.T) {
	t.Run("无认证无参数", func(t *testing.T) {
		cfg := &dbMongoConf.MongoConfig{Ip: "127.0.0.1", Port: 27017, Database: "test"}
		assert.Equal(t, "mongodb://127.0.0.1:27017/test", buildUri(cfg))
	})

	t.Run("带认证", func(t *testing.T) {
		cfg := &dbMongoConf.MongoConfig{Ip: "127.0.0.1", Port: 27017, Username: "root", Password: "secret", Database: "test"}
		assert.Equal(t, "mongodb://root:secret@127.0.0.1:27017/test", buildUri(cfg))
	})

	t.Run("带认证库", func(t *testing.T) {
		cfg := &dbMongoConf.MongoConfig{Ip: "127.0.0.1", Port: 27017, Username: "root", Password: "secret", Database: "test", AuthSource: "admin"}
		assert.Equal(t, "mongodb://root:secret@127.0.0.1:27017/test?authSource=admin", buildUri(cfg))
	})

	t.Run("完整连接串优先", func(t *testing.T) {
		cfg := &dbMongoConf.MongoConfig{Uri: "mongodb+srv://cluster.example.com/test?replicaSet=rs0", Ip: "127.0.0.1", Port: 27017, Database: "ignored"}
		assert.Equal(t, "mongodb+srv://cluster.example.com/test?replicaSet=rs0", buildUri(cfg))
	})
}

func TestMongoConfigNormalize(t *testing.T) {
	t.Run("ConnectTimeout 未设置时回填默认 10s", func(t *testing.T) {
		assert.Equal(t, dbMongoConf.DefaultConnectTimeout, dbMongoConf.Normalize(dbMongoConf.MongoConfig{}).ConnectTimeout)
	})

	t.Run("ConnectTimeout 显式设置时保持原值", func(t *testing.T) {
		assert.Equal(t, 3, dbMongoConf.Normalize(dbMongoConf.MongoConfig{ConnectTimeout: 3}).ConnectTimeout)
	})

	t.Run("0 值语义为驱动默认/不限制/不记录的字段不被回填", func(t *testing.T) {
		cfg := dbMongoConf.Normalize(dbMongoConf.MongoConfig{})
		assert.Zero(t, cfg.MaxPoolSize)
		assert.Zero(t, cfg.MinPoolSize)
		assert.Zero(t, cfg.MaxConnIdleTime)
		assert.Zero(t, cfg.SlowThreshold)
	})
}
