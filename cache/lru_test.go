package cache

import (
	//"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type testValue struct {
	b string
}

func (t *testValue) Len() int {
	return len(t.b)
}

// 测试并发情况下是否出现问题
func TestCache_GetAndAdd(t *testing.T) {
	var wg sync.WaitGroup
	cache := NewLRUCache(100000000)
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 100000; i++ {
			cache.Add(strconv.Itoa(i), &testValue{b: "小丑"})
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 100000; i++ {
			cache.Add(strconv.Itoa(i+100000), &testValue{b: "后面"})
		}
	}()

	wg.Wait()
	a := assert.New(t)
	for i := 100000; i < 200000; i++ {
		t, _ := cache.Get(strconv.Itoa(i))
		//fmt.Println(t)
		a.Equal("后面", t.(*testValue).b)
	}
}

//测试LRU逻辑
func TestCache_FreeMomory(t *testing.T) {
	a := assert.New(t)
	//测试LRU
	cache := NewLRUCache(90)
	for i := 0; i < 10; i++ {
		cache.Add(strconv.Itoa(i), &testValue{"123456789"})
	}

	//key = "0" 淘汰
	_, f0 := cache.Get("0")
	a.False(f0)
	//"1"没有淘汰
	v1, f1 := cache.Get("1")
	a.True(f1)
	a.Equal(v1.(*testValue).b, "123456789")
	//增加一个
	cache.Add("a", &testValue{"123456789"})
	_, f2 := cache.Get("2")
	a.False(f2)

	v3, f3 := cache.Get("3")
	a.True(f3)
	a.Equal(v3.(*testValue).b, "123456789")

}

func TestCache_FreeMomory2(t *testing.T) {
	timeout := time.Now().Add(3 * time.Second)
	a := assert.New(t)
	//测试LRU
	cache := NewLRUCache(90)
	for i := 0; i < 10; i++ {
		cache.AddWithExpiration(strconv.Itoa(i), &testValue{"123456789"}, timeout)
	}

	//key = "0" 淘汰
	_, f0 := cache.Get("0")
	a.False(f0)
	//"1"没有淘汰
	v1, f1 := cache.Get("1")
	a.True(f1)
	a.Equal(v1.(*testValue).b, "123456789")
	//增加一个
	cache.AddWithExpiration("a", &testValue{"123456789"}, timeout)
	_, f2 := cache.Get("2")
	a.False(f2)

	v3, f3 := cache.Get("3")
	a.True(f3)
	a.Equal(v3.(*testValue).b, "123456789")

}

func TestCache_AddWithExpiration(t *testing.T) {
	a := assert.New(t)
	cache := NewLRUCache(100)
	cache.AddWithExpiration("1", &testValue{"123456789"}, time.Now().Add(3 * time.Second))
	time.Sleep(2 * time.Second)
	v1, _ := cache.Get("1")
	a.Equal("123456789", v1.(*testValue).b)
	time.Sleep(2 * time.Second)
	_, f := cache.Get("1")
	a.False(f)
}


