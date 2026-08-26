# hecc-blot-db-mongo

基于 mongo-driver/v2 的 MongoDB 文档型数据库组件：集合 + 条件链式查询、文档读写，命令执行自动接入链路追踪。

## 安装

```bash
go get github.com/hecc-blot/db-mongo
```

## 接口定义

```go
import (
    dbContract "github.com/hecc-blot/db/contract"
    dbMongoContract "github.com/hecc-blot/db-mongo/contract"
)

type IDbDocument interface {
    dbContract.IDbBase
    WithContext(ctx context.Context) IDbDocument

    // 链式（返回副本）
    Collection(name string) IDbDocument
    Where(filter interface{}) IDbDocument
    Select(projection interface{}) IDbDocument
    Sort(sort interface{}) IDbDocument
    Limit(v int64) IDbDocument
    Skip(v int64) IDbDocument

    // 写入
    InsertOne(doc interface{}) (interface{}, error)
    InsertMany(docs interface{}) ([]interface{}, error)
    UpdateOne(update interface{}) error
    UpdateMany(update interface{}) error
    DeleteOne() error
    DeleteMany() error

    // 读取
    FindOne(dst interface{}) error
    Find(dst interface{}) error
    Count() (int64, error)
}
```

filter / projection / update / sort 均为 MongoDB 的 BSON 文档（`bson.M` / `bson.D`）。

## 初始化

```go
import (
    dbMongo "github.com/hecc-blot/db-mongo/service"
)

mongoDb, clearUp, err := dbMongo.NewMongo(&config.Mongo, logSvc)
if err != nil {
    panic(err)
}
defer clearUp()

container.Set(new(dbMongoContract.IDbDocument), mongoDb)
```

业务方直接注入 `IDbDocument`，每个请求用 `WithContext(ctx)` 取副本：

```go
type UserApi struct {
    Doc dbMongoContract.IDbDocument `inject:""`
}

func (a UserApi) Call(ctx *gin.Context) (interface{}, iCoreError.IError) {
    db := a.Doc.WithContext(ctx)   // 返回绑定请求上下文的副本，并发安全

    var user UserModel
    if err := db.Collection("users").Where(bson.M{"name": "tom"}).FindOne(&user); err != nil {
        return nil, errorSvc.NewError(response.Fail, err)
    }
    return user, nil
}
```

## CRUD 操作

```go
// 插入
id, err := db.Collection("users").InsertOne(&UserModel{Name: "tom"})
ids, err := db.Collection("users").InsertMany([]interface{}{
    &UserModel{Name: "a"},
    &UserModel{Name: "b"},
})

// 查询单条 / 多条
err = db.Collection("users").Where(bson.M{"name": "tom"}).FindOne(&user)
err = db.Collection("users").Where(bson.M{"age": bson.M{"$gt": 18}}).
    Sort(bson.D{{"age", -1}}).Limit(10).Find(&users)

// 更新 / 删除
err = db.Collection("users").Where(bson.M{"name": "tom"}).UpdateOne(bson.M{"$set": bson.M{"age": 20}})
err = db.Collection("users").Where(bson.M{"age": bson.M{"$lt": 18}}).DeleteMany()

// 统计
count, err := db.Collection("users").Where(bson.M{"age": bson.M{"$gte": 18}}).Count()
```

## 配置

```yaml
mongo:
  # uri 非空时优先使用完整连接串，否则由下面字段组装
  uri: ""
  ip: 127.0.0.1
  port: 27017
  username: root
  password: ""
  database: app
  auth_source: admin          # 认证库
  max_pool_size: 100          # 连接池最大连接数，0 用驱动默认
  min_pool_size: 10           # 连接池最小连接数，0 用驱动默认
  max_conn_idle_time: 60      # 连接最大空闲时间（秒），0 不限制
  connect_timeout: 10         # 建连/Ping 超时（秒），0 用默认 10s
  slow_threshold: 200         # 慢查询阈值（毫秒），0 不记录
```

## 链路追踪

通过 mongo-driver/v2 的 `CommandMonitor` 挂载 otelmongo 监控，命令执行自动生成 span，只依赖第三方 otel，不依赖 trace 模块。初始化顺序要求：先初始化 trace，再初始化 db-mongo。

## 相关模块

| 模块 | 说明 |
|------|------|
| [db](https://github.com/hecc-blot/db) | 关系型（MySQL / PostgreSQL），提供 `IDbBase` |
| [db-clickhouse](https://github.com/hecc-blot/db-clickhouse) | 分析型（ClickHouse） |
| [db-es](https://github.com/hecc-blot/db-es) | 搜索型（Elasticsearch） |
