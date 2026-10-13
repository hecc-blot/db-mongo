package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hecc-blot/core/contract/log"
	dbMongoConf "github.com/hecc-blot/db-mongo/config"
	dbMongoContract "github.com/hecc-blot/db-mongo/contract"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/v2/mongo/otelmongo"
	"go.uber.org/zap"
)

// MongoSvc MongoDB 文档数据库服务
type MongoSvc struct {
	ctx        context.Context
	client     *mongo.Client
	database   *mongo.Database
	collection *mongo.Collection
	filter     interface{}
	projection interface{}
	sort       interface{}
	limit      int64
	skip       int64

	logger        log.ILog
	slowThreshold time.Duration
}

// 编译期断言：确保 MongoSvc 实现 IDbDocument
var _ dbMongoContract.IDbDocument = (*MongoSvc)(nil)

// NewMongo 创建单个 MongoDB 实例，返回实例与清理函数
func NewMongo(config *dbMongoConf.MongoConfig, logger log.ILog) (dbMongoContract.IDbDocument, func(), error) {
	cfg := dbMongoConf.Normalize(*config)
	connectTimeout := time.Duration(cfg.ConnectTimeout) * time.Second

	opts := options.Client().ApplyURI(buildUri(&cfg))
	// 注册 OpenTelemetry 监控，命令执行自动生成 span
	opts.SetMonitor(otelmongo.NewMonitor())
	opts.SetTimeout(connectTimeout)
	if cfg.MaxPoolSize > 0 {
		opts.SetMaxPoolSize(cfg.MaxPoolSize)
	}
	if cfg.MinPoolSize > 0 {
		opts.SetMinPoolSize(cfg.MinPoolSize)
	}
	if cfg.MaxConnIdleTime > 0 {
		opts.SetMaxConnIdleTime(time.Duration(cfg.MaxConnIdleTime) * time.Second)
	}

	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, func() {}, err
	}

	// fail-fast：启动时 Ping，地址/认证错误立即暴露
	pingCtx, cancel := context.WithTimeout(context.Background(), connectTimeout)
	defer cancel()
	if err := client.Ping(pingCtx, readpref.Primary()); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, func() {}, err
	}

	return &MongoSvc{
			client:        client,
			database:      client.Database(cfg.Database),
			logger:        logger,
			slowThreshold: time.Duration(cfg.SlowThreshold) * time.Millisecond,
		}, func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = client.Disconnect(ctx)
		}, nil
}

// buildUri 组装 MongoDB 连接串；Uri 非空时直接返回
func buildUri(config *dbMongoConf.MongoConfig) string {
	if config.Uri != "" {
		return config.Uri
	}

	var auth string
	if config.Username != "" {
		auth = config.Username
		if config.Password != "" {
			auth += ":" + config.Password
		}
		auth += "@"
	}

	uri := fmt.Sprintf("mongodb://%s%s:%d/%s", auth, config.Ip, config.Port, config.Database)
	if config.AuthSource != "" {
		uri += "?authSource=" + config.AuthSource
	}
	return uri
}

// WithContext 设置上下文 — 返回副本，不修改原实例，保证并发安全
func (m *MongoSvc) WithContext(ctx context.Context) dbMongoContract.IDbDocument {
	c := *m
	c.ctx = ctx
	return &c
}

// GetInstance 返回底层 mongo.Client 实例，供执行高级操作
func (m *MongoSvc) GetInstance() any {
	return m.client
}

// Collection 指定集合 — 返回副本，不修改原实例
func (m *MongoSvc) Collection(name string) dbMongoContract.IDbDocument {
	c := *m
	c.collection = m.database.Collection(name)
	return &c
}

// Where 过滤条件 — 返回副本，不修改原实例
func (m *MongoSvc) Where(filter interface{}) dbMongoContract.IDbDocument {
	c := *m
	c.filter = filter
	return &c
}

// Select 投影字段 — 返回副本，不修改原实例
func (m *MongoSvc) Select(projection interface{}) dbMongoContract.IDbDocument {
	c := *m
	c.projection = projection
	return &c
}

// Sort 排序 — 返回副本，不修改原实例
func (m *MongoSvc) Sort(sort interface{}) dbMongoContract.IDbDocument {
	c := *m
	c.sort = sort
	return &c
}

// Limit 限制条数 — 返回副本，不修改原实例
func (m *MongoSvc) Limit(v int64) dbMongoContract.IDbDocument {
	c := *m
	c.limit = v
	return &c
}

// Skip 跳过条数 — 返回副本，不修改原实例
func (m *MongoSvc) Skip(v int64) dbMongoContract.IDbDocument {
	c := *m
	c.skip = v
	return &c
}

// InsertOne 插入单条，返回 _id
func (m *MongoSvc) InsertOne(doc interface{}) (interface{}, error) {
	if err := m.checkCollection(); err != nil {
		return nil, err
	}
	begin := time.Now()
	defer m.logSlow(begin, "insert_one")

	res, err := m.collection.InsertOne(m.ctx, doc)
	if err != nil {
		return nil, err
	}
	return res.InsertedID, nil
}

// InsertMany 批量插入，返回 _id 列表
func (m *MongoSvc) InsertMany(docs interface{}) ([]interface{}, error) {
	if err := m.checkCollection(); err != nil {
		return nil, err
	}
	begin := time.Now()
	defer m.logSlow(begin, "insert_many")

	res, err := m.collection.InsertMany(m.ctx, docs)
	if err != nil {
		return nil, err
	}
	return res.InsertedIDs, nil
}

// UpdateOne 更新单条
func (m *MongoSvc) UpdateOne(update interface{}) error {
	if err := m.checkCollection(); err != nil {
		return err
	}
	begin := time.Now()
	defer m.logSlow(begin, "update_one")

	_, err := m.collection.UpdateOne(m.ctx, m.filter, update)
	return err
}

// UpdateMany 更新多条
func (m *MongoSvc) UpdateMany(update interface{}) error {
	if err := m.checkCollection(); err != nil {
		return err
	}
	begin := time.Now()
	defer m.logSlow(begin, "update_many")

	_, err := m.collection.UpdateMany(m.ctx, m.filter, update)
	return err
}

// DeleteOne 删除单条
func (m *MongoSvc) DeleteOne() error {
	if err := m.checkCollection(); err != nil {
		return err
	}
	begin := time.Now()
	defer m.logSlow(begin, "delete_one")

	_, err := m.collection.DeleteOne(m.ctx, m.filter)
	return err
}

// DeleteMany 删除多条
func (m *MongoSvc) DeleteMany() error {
	if err := m.checkCollection(); err != nil {
		return err
	}
	begin := time.Now()
	defer m.logSlow(begin, "delete_many")

	_, err := m.collection.DeleteMany(m.ctx, m.filter)
	return err
}

// FindOne 查询单条
func (m *MongoSvc) FindOne(dst interface{}) error {
	if err := m.checkCollection(); err != nil {
		return err
	}
	begin := time.Now()
	defer m.logSlow(begin, "find_one")

	return m.collection.FindOne(m.ctx, m.filter, m.buildFindOneOptions()).Decode(dst)
}

// Find 查询多条
func (m *MongoSvc) Find(dst interface{}) error {
	if err := m.checkCollection(); err != nil {
		return err
	}
	begin := time.Now()
	defer m.logSlow(begin, "find")

	cursor, err := m.collection.Find(m.ctx, m.filter, m.buildFindOptions())
	if err != nil {
		return err
	}
	defer cursor.Close(m.ctx)
	return cursor.All(m.ctx, dst)
}

// Count 统计数量
func (m *MongoSvc) Count() (int64, error) {
	if err := m.checkCollection(); err != nil {
		return 0, err
	}
	begin := time.Now()
	defer m.logSlow(begin, "count")

	return m.collection.CountDocuments(m.ctx, m.filter)
}

// buildFindOptions 组装 Find 选项
func (m *MongoSvc) buildFindOptions() *options.FindOptionsBuilder {
	b := options.Find()
	if m.projection != nil {
		b.SetProjection(m.projection)
	}
	if m.sort != nil {
		b.SetSort(m.sort)
	}
	if m.limit > 0 {
		b.SetLimit(m.limit)
	}
	if m.skip > 0 {
		b.SetSkip(m.skip)
	}
	return b
}

// buildFindOneOptions 组装 FindOne 选项
func (m *MongoSvc) buildFindOneOptions() *options.FindOneOptionsBuilder {
	b := options.FindOne()
	if m.projection != nil {
		b.SetProjection(m.projection)
	}
	if m.sort != nil {
		b.SetSort(m.sort)
	}
	if m.skip > 0 {
		b.SetSkip(m.skip)
	}
	return b
}

func (m *MongoSvc) checkCollection() error {
	if m.collection == nil {
		return errors.New("mongo: collection is not specified, call Collection(name) first")
	}
	return nil
}

func (m *MongoSvc) logSlow(begin time.Time, op string) {
	if m.slowThreshold <= 0 {
		return
	}
	if elapsed := time.Since(begin); elapsed > m.slowThreshold {
		m.logger.Warn(m.ctx, "Slow Mongo", zap.Duration("elapsed", elapsed), zap.String("op", op))
	}
}
