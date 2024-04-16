package cache

type MaxMemoryPolicy int


//不同缓存更新策略
const (
	VOLATILE_LRU    MaxMemoryPolicy = 1
	VOLATILE_RANDOM MaxMemoryPolicy = 2
	ALLKEYS_LRU     MaxMemoryPolicy = 3
	ALLKEYS_RANDOM  MaxMemoryPolicy = 4
)
