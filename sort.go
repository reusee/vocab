package main

type PracticeInfoSorter struct {
	Slice []PracticeInfo
	Cmp   func(a, b PracticeInfo) bool
}

func (s PracticeInfoSorter) Len() int {
	return len(s.Slice)
}

func (s PracticeInfoSorter) Less(i, j int) bool {
	return s.Cmp(s.Slice[i], s.Slice[j])
}

func (s PracticeInfoSorter) Swap(i, j int) {
	s.Slice[i], s.Slice[j] = s.Slice[j], s.Slice[i]
}
