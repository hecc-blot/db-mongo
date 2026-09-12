# hecc-blot-db-mongo

文档型数据库模块（MongoDB）：集合选择、条件链式查询、文档读写。

## 安装

```bash
go get github.com/hecc-blot/db-mongo
```

## 快速开始

```go
import (
	"context"

	logContract "github.com/hecc-blot/core/contract/log"
	dbMongoContract "github.com/hecc-blot/db-mongo/contract"
	dbMongo "github.com/hecc-blot/db-mongo/service"
)

func main() {
	// 1. 构造（返回 实例 / 清理函数 / 错误）；构造时会 Ping，地址或认证错误立即暴露
	mongoDb, clearUp, err := dbMongo.NewMongo(&config.Mongo, logSvc)
	if err != nil {
		panic(err)
	}
	defer clearUp() // 一定非 nil，可直接 defer

	// 2. 注册到容器（须早于路由注册）
	container.Set(new(dbMongoContract.IDbDocument), mongoDb)

	// 3. 使用：先 Collection 指定集合，再读写
	ctx := context.Background()
	_, err = mongoDb.WithContext(ctx).Collection("users").InsertOne(bson.M{"name": "tom"})
}
```

完整的应用组装顺序（日志 → trace → db → 容器 → 路由）见[组织主页](https://github.com/hecc-blot)。

## 接口定义

```go
import (
	"context"

	dbContract "github.com/hecc-blot/db/contract"
	dbMongoContract "github.com/hecc-blot/db-mongo/contract"
)

// 复用 db 模块的 IDbBase（GetInstance）。
// filter / projection / update / sort 均为 BSON 文档（bson.M / bson.D）。
type IDbDocument interface {
	dbContract.IDbBase
	WithContext(ctx context.Context) IDbDocument

	// 链式（返回副本，不修改原实例）
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
	Find(dst interface{}) error // dst 为 *[]T
	Count() (int64, error)
}
```

## 使用

### 定义文档

普通结构体 + `bson` tag：

```go
type UserModel struct {
	Name string `bson:"name"`
	Age  int    `bson:"age"`
}
```

### 写入与读取

```go
import "go.mongodb.org/mongo-driver/v2/bson"

users := mongoDb.WithContext(ctx).Collection("users") // 每次读写前先指定集合

// 插入单条，返回 _id
id, err := users.InsertOne(bson.M{"name": "tom", "age": 20})

// 批量插入，返回 _id 列表
ids, err := users.InsertMany([]interface{}{
	bson.M{"name": "jerry", "age": 18},
	bson.M{"name": "spike", "age": 3},
})

// 条件查询（链式返回副本）
var one UserModel
err = users.Where(bson.M{"name": "tom"}).FindOne(&one)

var list []UserModel
err = users.
	Where(bson.M{"age": bson.M{"$gte": 18}}).
	Select(bson.M{"name": 1}).
	Sort(bson.D{{Key: "age", Value: -1}}).
	Limit(10).
	Skip(0).
	Find(&list)

// 更新
err = users.Where(bson.M{"name": "tom"}).
	UpdateOne(bson.M{"$set": bson.M{"age": 21}})

// 删除
err = users.Where(bson.M{"name": "tom"}).DeleteOne()

// 计数
n, err := users.Where(bson.M{"age": bson.M{"$gte": 18}}).Count()
```

> ⚠️ **`UpdateMany` / `DeleteMany` 不传 `Where` 会作用于整个集合。** 未调用 `Where` 时 filter 为 nil，驱动会把它当成「匹配全部文档」。批量操作前务必先 `Where`。

> `Find` 的 `dst` 必须是 `*[]T`；`FindOne` 的 `dst` 是 `*T`。

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

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `uri` | string | — | 完整连接串，非空时**忽略**其余连接字段 |
| `ip` / `port` | string / int | — | 地址与端口 |
| `username` / `password` | string | — | 认证信息，`username` 非空才拼进 URI |
| `database` | string | — | 库名 |
| `auth_source` | string | — | 认证库；非空才追加 `?authSource=` |
| `max_pool_size` / `min_pool_size` | uint64 | 0 | 0 表示用驱动默认（不回填） |
| `max_conn_idle_time` | int | 0 | 秒，0 表示不限制（不回填） |
| `connect_timeout` | int | 10 | 建连/Ping 超时（秒），0 值由 `Normalize` 回填为 10 |
| `slow_threshold` | int | 0 | 慢查询阈值（毫秒），0 表示不记录（不回填） |

> `Normalize` **只回填 `connect_timeout`**。其余字段的 0 值语义本身就是「用驱动默认 / 不限制 / 不记录」，不会被改写。

## 语义与约定

- **构造即校验连通性**：`NewMongo` 会 Ping 主节点，地址/认证错误在初始化阶段就返回 error，不会拖到第一次查询。
- **集合必须先指定**：所有读写都要求先 `Collection(name)`。未指定时返回 `mongo: 未指定集合，请先调用 Collection(name)`。
- **链式方法返回副本**：`Collection`/`Where`/`Select`/`Sort`/`Limit`/`Skip`/`WithContext` 都返回新实例，不修改原实例；实例可安全并发共享。
- **`UpdateMany` / `DeleteMany` 的 filter 为 nil 时匹配全集合**（见上文警告）。
- **释放语义**：`clearUp()` 断开客户端连接；失败路径返回的也是非 nil 的空函数，可以无条件 `defer`。

## 相关模块

| 模块 | 说明 |
|------|------|
| [core](https://github.com/hecc-blot/core) | `ILog` 日志契约 |
| [db](https://github.com/hecc-blot/db) | `IDbBase` 契约来源 |
| [trace](https://github.com/hecc-blot/trace) | 必须先于本模块初始化，否则命令 span 丢失 |
| [db-clickhouse](https://github.com/hecc-blot/db-clickhouse) / [db-es](https://github.com/hecc-blot/db-es) | 平级的专用驱动模块 |
