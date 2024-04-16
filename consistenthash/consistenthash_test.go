package consistenthash

import (
	"testing"
	
	"strconv"
)

func TestConsistenthash(t *testing.T) {
	//自定义虚拟节点间隔，哈希函数
	hash := New(Replicas(3), HashFunc(func(data []byte) uint32 {
		i, _ := strconv.Atoi(string(data))
		return uint32(i)
	}))

	hash.Add("6", "4", "2")

	testCases := map[string]string {
		"2": "2",
		"11": "2",
		"23": "4",
		"26": "6",
		"27": "2",
	}

	for k, v := range testCases {
		if hash.Get(k) != v {
			t.Errorf("hash.Get(%s) expected %s, but %s", k, v, hash.Get(k))
		}
	}

	hash.Add("8")
	testCases["27"] = "8"
	for k, v := range testCases {
		if hash.Get(k) != v {
			t.Errorf("hash.Get(%s) expected %s, but %s", k, v, hash.Get(k))
		}
	}

	hash.Remove("8")
	testCases["27"] = "2"
	for k, v := range testCases {
		if hash.Get(k) != v {
			t.Errorf("hash.Get(%s) expected %s, but %s", k, v, hash.Get(k))
		}
	}
}