package main

import (
	crand "crypto/rand"
	"encoding/binary"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/reusee/gobfile"
)

var (
	pt = fmt.Printf
)

func init() {
	var seed int64
	binary.Read(crand.Reader, binary.LittleEndian, &seed)
	rand.Seed(seed)

	flag.Parse()
}

func main() {
	/*
		traceFile, err := os.Create("trace")
		checkErr("create trace file", err)
		defer traceFile.Close()
		err = pprof.StartTrace(traceFile)
		checkErr("start trace", err)
		defer pprof.StopTrace()
	*/

	dir := "."
	if args := flag.Args(); len(args) > 0 {
		dir = args[0]
	}

	// load words
	type Word struct {
		Text string
		Desc string
	}
	words := map[string]Word{}
	content, err := ioutil.ReadFile(filepath.Join(dir, "words"))
	checkErr("read words file", err)
	for _, line := range strings.Split(string(content), "\n") {
		if len(line) == 0 {
			continue
		}
		index := 0
		runes := []rune(line)
		for i, r := range runes {
			if unicode.IsSpace(r) {
				index = i
				break
			}
		}
		text := string(runes[:index])
		desc := string(runes[index+1:])
		if len(text) == 0 || len(desc) == 0 {
			log.Fatalf("invalid word entry %s\n", line)
		}
		words[text] = Word{
			Text: text,
			Desc: desc,
		}
	}

	// data file
	dataFilePath := filepath.Join(dir, "data")
	dataFileLockPath := filepath.Join(dir, ".data.lock")
	type HistoryEntry struct {
		Time time.Time
		What string
	}
	type Practice struct {
		Type string
		Text string
	}
	data := struct {
		History map[Practice][]HistoryEntry
	}{
		History: make(map[Practice][]HistoryEntry),
	}
	dataFile, err := gobfile.New(&data, dataFilePath, gobfile.NewFileLocker(dataFileLockPath))
	checkErr("open data file", err)
	defer dataFile.Close()
	defer dataFile.Save()

	// check new entry
	for _, word := range words {
		for _, t := range []string{"audio", "text", "usage"} {
			practice := Practice{
				Type: t,
				Text: word.Text,
			}
			if _, ok := data.History[practice]; !ok {
				pt("new practice: %s %s\n", t, word.Text)
				data.History[practice] = append(data.History[practice], HistoryEntry{
					Time: time.Now(),
					What: "ok",
				})
			}
		}
	}
	dataFile.Save()

	// review functions
	audioReview := func(word Word) bool {
		var reply string
		retry := 1
	play:
		pt("playing audio\n")
		err := exec.Command("mpv", filepath.Join(dir, fmt.Sprintf("%s.mp3", word.Text))).Run()
		checkErr("play audio", err)
	ask1:
		pt("'j' to show text, 'r' to replay\n")
		fmt.Scanf("%s", &reply)
		switch reply {
		case "j":
			pt("%s\n", word.Text)
		ask2:
			pt("'y' to level up, 'n' to keep\n")
			fmt.Scanf("%s", &reply)
			switch reply {
			case "y":
				return true
			case "n":
				return false
			default:
				goto ask2
			}
		case "r":
			if retry > 0 {
				retry--
				goto play
			} else {
				pt("no more replay\n")
				goto ask1
			}
		default:
			goto ask1
		}
		return false
	}

	textReview := func(word Word) bool {
		pt("showing text\n")
		pt("%s\n", word.Text)
	ask2:
		pt("'j' to play audio\n")
		var reply string
		fmt.Scanf("%s\n", &reply)
		switch reply {
		case "j":
			pt("playing audio\n")
			err := exec.Command("mpv", filepath.Join(dir, fmt.Sprintf("%s.mp3", word.Text))).Run()
			checkErr("play audio", err)
		ask:
			pt("'y' to level up, 'n' to keep\n")
			fmt.Scanf("%s\n", &reply)
			switch reply {
			case "y":
				return true
			case "n":
				return false
			default:
				goto ask
			}
		default:
			goto ask2
		}
		return false
	}

	usageReview := func(word Word) bool {
		pt("showing usage\n")
		pt("%s\n", word.Desc)
	ask:
		pt("'j' to show answer\n")
		var reply string
		fmt.Scanf("%s\n", &reply)
		switch reply {
		case "j":
			pt("playing audio\n")
			err := exec.Command("mpv", filepath.Join(dir, fmt.Sprintf("%s.mp3", word.Text))).Run()
			checkErr("play audio", err)
			pt("%s\n", word.Text)
		ask2:
			pt("'y' to level up, 'n' to keep\n")
			fmt.Scanf("%s\n", &reply)
			switch reply {
			case "y":
				return true
			case "n":
				return false
			default:
				goto ask2
			}
		default:
			goto ask
		}
		return false
	}

	reviewFuncs := map[string]func(Word) bool{
		"audio": audioReview,
		"text":  textReview,
		"usage": usageReview,
	}

	type PracticeInfo struct {
		Practice Practice
		Max      time.Duration
		Fade     time.Duration
		Ratio    float64
	}
	practices := []PracticeInfo{}
	for practice, history := range data.History {
		// calculate fade and max
		last := history[len(history)-1]
		fade := time.Now().Sub(last.Time)
		var max time.Duration
		for i := 1; i < len(history); i++ {
			if history[i].What != "ok" {
				continue
			}
			if d := history[i].Time.Sub(history[i-1].Time); d > max {
				max = d
			}
		}
		ratio := float64(fade) / float64(max+1)
		// filter
		if fade < max {
			continue
		}
		if fade < time.Second*30 { // skip newly added
			continue
		}
		// collect
		practices = append(practices, PracticeInfo{
			Practice: practice,
			Fade:     fade,
			Max:      max,
			Ratio:    ratio,
		})
	}
	pt("%d practices\n", len(practices))

	// sort
	for i := len(practices) - 1; i >= 1; i-- {
		j := rand.Intn(i + 1)
		practices[i], practices[j] = practices[j], practices[i]
	}
	Sort(practices, func(left, right PracticeInfo) bool {
		return left.Ratio > right.Ratio
	})

	// unique words
	practicedWords := map[string]struct{}{}
	infos := []PracticeInfo{}
	for _, info := range practices {
		if _, ok := practicedWords[info.Practice.Text]; ok {
			continue
		}
		infos = append(infos, info)
		practicedWords[info.Practice.Text] = struct{}{}
	}
	practices = infos

	// practice
	for _, practice := range practices {
		pt("practice %s fade %v max %v ratio %f\n", practice.Practice.Type, practice.Fade, practice.Max, practice.Ratio)
		var what string
		if reviewFuncs[practice.Practice.Type](words[practice.Practice.Text]) {
			what = "ok"
		} else {
			what = "fail"
		}
		data.History[practice.Practice] = append(data.History[practice.Practice], HistoryEntry{
			What: what,
			Time: time.Now(),
		})
		dataFile.Save()
	}
}

func checkErr(desc string, err error) {
	if err != nil {
		log.Fatalf("%s error %v", desc, err)
	}
}

func Sort(slice interface{}, cmp interface{}) {
	sort.Sort(sliceSorter{reflect.ValueOf(slice), reflect.ValueOf(cmp)})
}

type sliceSorter struct {
	slice, cmp reflect.Value
}

func (t sliceSorter) Len() int {
	return t.slice.Len()
}

func (t sliceSorter) Less(i, j int) bool {
	return t.cmp.Call([]reflect.Value{
		t.slice.Index(i),
		t.slice.Index(j),
	})[0].Bool()
}

func (t sliceSorter) Swap(i, j int) {
	tmp := t.slice.Index(i).Interface()
	t.slice.Index(i).Set(t.slice.Index(j))
	t.slice.Index(j).Set(reflect.ValueOf(tmp))
}
