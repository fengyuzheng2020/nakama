package cache

import (
	"github.com/heroiclabs/nakama/v3/sample_go_module/model"
	"sync"
)

// 定义一个泛型的 UserCache 类型
type UserCache[T any] struct {
	cache sync.Map
}

// 设置用户缓存
func (uc *UserCache[T]) Set(userID string, data T) {
	uc.cache.Store(userID, data)
}

// 获取用户缓存
func (uc *UserCache[T]) Get(userID string) (T, bool) {
	val, found := uc.cache.Load(userID)
	if found {
		return val.(T), true
	}
	var zero T // 返回类型的零值
	return zero, false
}

// 删除用户缓存
func (uc *UserCache[T]) Delete(userID string) {
	uc.cache.Delete(userID)
}

var UserCacheInstance = &UserCache[model.UserCacheData]{}
