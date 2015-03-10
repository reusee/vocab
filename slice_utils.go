package main

import (
	crand "crypto/rand"
	"encoding/binary"
	"math/rand"
	"sort"
)

func init() {
	var seed int64
	binary.Read(crand.Reader, binary.LittleEndian, &seed)
	rand.Seed(seed)
}

type PracticeInfos []PracticeInfo

func (s PracticeInfos) Reduce(initial interface{}, fn func(value interface{}, elem PracticeInfo) interface{}) (ret interface{}) {
	ret = initial
	for _, elem := range s {
		ret = fn(ret, elem)
	}
	return
}

func (s PracticeInfos) Map(fn func(PracticeInfo) PracticeInfo) (ret PracticeInfos) {
	for _, elem := range s {
		ret = append(ret, fn(elem))
	}
	return
}

func (s PracticeInfos) Filter(filter func(PracticeInfo) bool) (ret PracticeInfos) {
	for _, elem := range s {
		if filter(elem) {
			ret = append(ret, elem)
		}
	}
	return
}

func (s PracticeInfos) All(predict func(PracticeInfo) bool) (ret bool) {
	ret = true
	for _, elem := range s {
		ret = predict(elem) && ret
	}
	return
}

func (s PracticeInfos) Any(predict func(PracticeInfo) bool) (ret bool) {
	for _, elem := range s {
		ret = predict(elem) || ret
	}
	return
}

func (s PracticeInfos) Each(fn func(e PracticeInfo)) {
	for _, elem := range s {
		fn(elem)
	}
}

func (s PracticeInfos) Shuffle() {
	for i := len(s) - 1; i >= 1; i-- {
		j := rand.Intn(i + 1)
		s[i], s[j] = s[j], s[i]
	}
}

func (s PracticeInfos) Sort(cmp func(a, b PracticeInfo) bool) {
	sort.Sort(sliceSorter{
		l: len(s),
		less: func(i, j int) bool {
			return cmp(s[i], s[j])
		},
		swap: func(i, j int) {
			s[i], s[j] = s[j], s[i]
		},
	})
}

type sliceSorter struct {
	l    int
	less func(i, j int) bool
	swap func(i, j int)
}

func (t sliceSorter) Len() int {
	return t.l
}

func (t sliceSorter) Less(i, j int) bool {
	return t.less(i, j)
}

func (t sliceSorter) Swap(i, j int) {
	t.swap(i, j)
}
