package singleflight

import (
	"testing"
	//"golang.org/x/sync/singleflight"
	"time"
	//"sync"
	"fmt"
)

func TestDo(t *testing.T) {
	fn := func() (interface{}, error) {
		//需要一点时间处理，
		time.Sleep(1 * time.Millisecond)
		return "hello", nil
	}
	g := Group{}
	n := 100
	//var wg sync.WaitGroup
	
	for i := 0; i < n; i++ {
		//wg.Add(1)
		go func(i int) {
			v, err := g.Do("www", fn)
			
			if v != "hello" {
				t.Errorf("want %v,Get %v", "hello", v)
			}
			if err != nil {
				t.Errorf("want %v,Get %v", nil, err)
			}
			fmt.Println(i, "---", v, err)
			//wg.Done()
		}(i)
		
		
	}

	time.Sleep(3 * time.Second)
	
}