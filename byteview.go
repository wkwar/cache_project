package groupcache
// []byte 和 string的封装,一个统一接口
import (
	"io"
	// "bytes"
	// "strings"
	"errors"
)
type ByteView struct {
	b []byte
	s string
}

func (v ByteView) Len() int {
	if v.b != nil {
		return len(v.b)
	}

	return len(v.s)
}

func (v ByteView) ByteSlice() []byte {
	if v.b != nil {
		return v.b
	}
	return []byte(v.s)
}

func (v ByteView) String() string {
	if v.b != nil {
		return string(v.b)
	}
	return v.s
}

func (v ByteView) At(i int) byte {
	if v.b != nil {
		return v.b[i]
	}
	return byte(v.s[i])
}

func (v ByteView) SliceFrom(from int) ByteView {
	if v.b != nil {
		return ByteView {
			b: v.b[from:],
		}
	}
	return ByteView {
		s: v.s[from:],
	}
}

func (v ByteView) Slice(from, to int) ByteView {
	if v.b != nil {
		return ByteView {
			b: v.b[from: to],
		}
	}
	return ByteView {
		s: v.s[from: to],
	}
}

func (v ByteView) Copy(b []byte) int {
	if v.b != nil {
		return copy(b, v.b)
	}	
	return copy(b, v.s)
}

func (v ByteView) Equal(bv ByteView) bool {
	if bv.b != nil {
		return v.EqualBytes(bv.b)
	}
	return v.EqualString(bv.s)
}

func (v ByteView) EqualString(s string) bool {
	if v.b != nil {
		if len(v.b) != len(s) {
			return false
		}
		// for i, v := range v.b {
		// 	if s[i] != v {
		// 		return false
		// 	}
		// }
		// return true

		return string(v.b) == s
	}
	
	return v.s == s
}

func (v ByteView) EqualBytes(b []byte) bool {
	if v.b != nil {
		if len(v.b) != len(b) {
			return false
		}
		for i, v := range v.b {
			if b[i] != v {
				return false
			}
		}
		return true
	}
	
	return v.s == string(b)
}

// func (v ByteView) Reader() *io.ReadSeeker {
// 	if v.b != nil {
// 		return bytes.NewReader(v.b)
// 	}
// 	return strings.NewReader(v.s)
// }

func (v ByteView) ReadAt(b []byte, from int) (n int, err error) {
	if from < 0 {
		return 0, errors.New("view: invild offset")
	}
	if from >= v.Len() {
		return 0, io.EOF
	}

	n = v.SliceFrom(from).Copy(b)
	if len(b) > n{
		err = io.EOF
	}
	return 
} 

