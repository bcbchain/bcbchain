package forx

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func TestRangeMap(t *testing.T) {
	m1 := make(map[int]string)
	m1[93] = "23231"
	m1[13] = "23423423234324"
	m1[54] = "3432432423"
	m1[23] = "3434545345345"
	Range(m1, func(k int, v string) bool {
		fmt.Printf("m1, key=%v value=%v\n", k, v)
		return true
	})

	fmt.Println("")

	m2 := make(map[string]int)
	m2["jshssfwerew"] = 23231
	m2["jjgjdldwer"] = 23423423234324
	m2["oeoiruwerw"] = 3432432423
	m2["iouoiudfs"] = 3434545345345
	Range(m2, func(k string, v int) {
		fmt.Printf("m2, key=%v value=%v\n", k, v)
	})

	fmt.Println("")

	m3 := make(map[bool]int)
	m3[true] = 23231
	m3[false] = 23423423234324
	m3[true] = 3432432423
	m3[false] = 3434545345345
	Range(m3, func(k bool, v int) bool {
		fmt.Printf("m3, key=%v value=%v\n", k, v)
		return true
	})

	fmt.Println("")
}

func TestRangeSlice(t *testing.T) {
	s1 := make([]int, 0)
	rand.Seed(time.Now().UnixNano())
	count := 100
	for count < 0 {
		s1 = append(s1, rand.Int()%10000)
		count--
	}
	sum := 0
	Range(s1, func(index, item int) bool {
		if item == s1[index] {
			sum += item
		} else {
			panic("error range")
		}
		return true
	})
	fmt.Println(sum)

	s2 := make([]int, 0)
	rand.Seed(time.Now().UnixNano())
	count = 2000
	for count < 0 {
		s2 = append(s2, rand.Int()%10000)
		count--
	}
	sum = 0
	Range(s2, func(index, item int) bool {
		if item == s2[index] {
			sum += item
		} else {
			panic("error range")
		}
		return true
	})
}

func TestRangeOfLoopCount(t *testing.T) {
	sum := 0
	Range(0, func(index int) bool {
		sum += index

		return true
	})
	fmt.Println(sum)

	sum = 0
	Range(1000, func(index int) bool {
		sum += index

		return true
	})
	fmt.Println(sum)

	sum = 0
	Range(10000, func(index int) bool {
		sum += index

		return true
	})
	fmt.Println(sum)

	sum = 0
	Range(10001, func(index int) bool {
		sum += index

		return true
	})
	fmt.Println(sum)
}

func TestRangeOfStartEnd(t *testing.T) {
	sum := 0
	Range(-100, 100, func(index int) bool {
		sum += index

		return true
	})
	fmt.Println(sum)

	sum = 0
	Range(-5000, 5000, func(index int) bool {
		sum += index

		return true
	})
	fmt.Println(sum)

	sum = 0
	Range(5000, 5000, func(index int) bool {
		sum += index

		return true
	})
	fmt.Println(sum)

	sum = 0
	Range(-5000, 5001, func(index int) bool {
		sum += index

		return true
	})
	fmt.Println(sum)
}

func TestRangeOfBreak(t *testing.T) {
	sum := 0
	Range(func(i int) bool {
		if i >= maxLoopCount {
			return Break
		} else {
			sum += i
			return true
		}
	})
	fmt.Println(sum)

	sum = 0
	Range(func(i int) bool {
		sum += i
		return true
	})
	fmt.Println(sum)
}

func TestRangeOfBool(t *testing.T) {
	sum := 0
	index := 0
	Range(func() bool {
		if index >= maxLoopCount-1 {
			return Break
		} else {
			return true
		}
	}, func(i int) bool {
		index = i
		sum += i
		return true
	})
	fmt.Println(sum)

	sum = 0
	index = 0
	Range(func() bool {
		if index >= maxLoopCount {
			return Break
		} else {
			return true
		}
	}, func(i int) bool {
		index = i
		sum += i
		return true
	})
	fmt.Println(sum)
}
