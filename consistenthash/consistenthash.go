package consistenthash

/*

 */
import (
	"hash/crc32"
	"sort"
	"strconv"
)

// 将key的值[]byte 转成 32位
var (
	GlobalReplicas  = defaultReplicas
	defaultHash     = crc32.ChecksumIEEE
	defaultReplicas = 150
)

type Hash func(data []byte) uint32

type Map struct {
	hash     Hash           //哈希函数
	replicas int            //虚拟节点倍数
	keys     []int          //哈希环
	hashMap  map[int]string //虚拟节点与真实节点的映射表
}

type ConsOptions func(*Map)

func New(opts ...ConsOptions) *Map {
	m := Map{
		hash:     defaultHash,
		replicas: defaultReplicas,
		hashMap:  make(map[int]string),
	}

	//更改项
	for _, opt := range opts {
		opt(&m)
	}
	return &m
}


func Replicas(replicas int) ConsOptions {
	return func(m *Map) {
		m.replicas = replicas
	}
}

func HashFunc(hash Hash) ConsOptions {
	return func(m *Map) {
		m.hash = hash
	}
}

func (m *Map) IsEmpty() bool {
	return len(m.keys) == 0
}

// 输入几个真实节点，然后映射虚拟节点，
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		//虚拟节点
		for i := 0; i < m.replicas; i++ {
			//输入的string类型的key ---> uint32 --> int
			hash_key := int(m.hash([]byte(strconv.Itoa(i) + key)))
			m.keys = append(m.keys, hash_key)
			m.hashMap[hash_key] = key
			
		}
	}
	//将keys 按大小排序
	sort.Ints(m.keys)
}

func (m *Map) Remove(key string) {
	for i := 0; i < m.replicas; i++ {
		//输入的string类型的key ---> uint32 --> int
		hash_key := int(m.hash([]byte(strconv.Itoa(i) + key)))
		//删除列表中的元素
		idx := sort.SearchInts(m.keys, hash_key)
		m.keys = append(m.keys[:idx], m.keys[idx+1:]...)
		//删除map映射数据
		delete(m.hashMap, hash_key)
		
	}
}

// 真实节点 --- 匹配到最近的虚拟节点
func (m *Map) Get(key string) string {
	if m.IsEmpty() {
		return ""
	}
	hash_key := int(m.hash([]byte(key)))
	idx := sort.Search(len(m.keys), func(i int) bool { return m.keys[i] >= hash_key })
	//没找到
	if idx == len(m.keys) {
		idx = 0
	}
	return m.hashMap[m.keys[idx]]
}
