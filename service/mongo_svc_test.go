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
