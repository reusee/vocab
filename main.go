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
	dir := "."
	if args := flag.Args(); len(args) > 0 {
		dir = args[0]
	}

	// load words
	type Word struct {
		Text string
		Desc string
	}
	words := []Word{}
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
		words = append(words, Word{
			Text: text,
			Desc: desc,
		})
	}
	for i := len(words) - 1; i >= 1; i-- {
		j := rand.Intn(i + 1)
		words[i], words[j] = words[j], words[i]
	}

	// data file
	dataFilePath := filepath.Join(dir, "data")
	dataFileLockPath := filepath.Join(dir, ".data.lock")
	type HistoryEntry struct {
		Time time.Time
		What string
	}
	data := struct {
		History map[string][]HistoryEntry
	}{
		History: make(map[string][]HistoryEntry),
	}
	dataFile, err := gobfile.New(&data, dataFilePath, gobfile.NewFileLocker(dataFileLockPath))
	checkErr("open data file", err)
	defer dataFile.Close()
	defer dataFile.Save()

	// check new entry
	for _, word := range words {
		if _, ok := data.History[word.Text]; !ok {
			pt("found new word entry: %s\n", word.Text)
			data.History[word.Text] = append(data.History[word.Text], HistoryEntry{
				Time: time.Now(),
				What: "new",
			})
		}
	}

	// review
	type Reviewer func(Word) bool
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
		panic("TODO")
		return false
	}

	usageReview := func(word Word) bool {
		panic("TODO")
		return false
	}

	type LevelSpec struct {
		What     string
		Duration time.Duration
		Reviewer Reviewer
	}
	levels := []LevelSpec{
		{"new", time.Hour, audioReview},
		{"a1", time.Hour, textReview},
		{"a2", time.Hour, usageReview},
		{"a3", time.Hour * 24 * 3, audioReview},
	}
	for _, word := range words {
		history := data.History[word.Text]
		lastHistory := history[len(history)-1]
		index := 0
		for lastHistory.What != levels[index].What {
			index++
		}
		if time.Now().Sub(lastHistory.Time) > levels[index].Duration {
			ok := levels[index].Reviewer(word)
			var what string
			if ok {
				what = levels[index+1].What
			} else {
				what = levels[index].What
			}
			_ = what
			data.History[word.Text] = append(data.History[word.Text], HistoryEntry{
				What: what,
				Time: time.Now(),
			})
		}
	}
}

func checkErr(desc string, err error) {
	if err != nil {
		log.Fatalf("%s error %v", desc, err)
	}
}
