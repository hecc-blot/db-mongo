package contract

import (
	"context"

	dbContract "github.com/hecc-blot/db/contract"
)

// IDbDocument 文档型数据库接口（MongoDB）。
// 复用 db 模块的 IDbBase（GetInstance），自身提供集合 + 条件链式查询 + 文档读写能力。
// filter / projection / update / sort 均为 MongoDB 的 BSON 文档（bson.M / bson.D）。
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
	Find(dst interface{}) error
	Count() (int64, error)
}
