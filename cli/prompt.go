package cli

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/qjpcpu/common.v2/fp"
	py "github.com/qjpcpu/common.v2/pinyin"
	"github.com/qjpcpu/go-prompt"
)

const (
	ParamInputHintSymbol = ">"
	PromptTypeFile       = "FILE   "
	PromptTypeDir        = "DIR    "
	PromptTypeDefault    = "DEFAULT"
	PromptTypeHistory    = "HISTORY"
)

type SelectWidget = promptui.Select

type SelectFn func(*SelectWidget)

func FixedSelect(label string, choices []string, opt ...SelectFn) (int, string) {
	prompt := promptui.Select{
		Label: label,
		Items: choices,
	}
	for _, fn := range opt {
		fn(&prompt)
	}

	_, result, _ := prompt.Run()

	for i, v := range choices {
		if v == result {
			return i, v
		}
	}
	return -1, ""
}

// Select from menu
func Select(label string, choices []string, opt ...SelectFn) (int, string) {
	newChoices, hit := reOrderChoicesByFreq(label, choices)
	idx, str := FixedSelect(label, newChoices, opt...)
	return hit(idx), str
}

// SelectWithSearch from menu
func SelectWithSearch(label string, choices []string) int {
	newChoices, hit := reOrderChoicesByFreq(label, choices)
	searchFunction := func(s *SelectWidget) {
		s.Size = 20
		s.HideSelected = true
		s.Searcher = func(input string, index int) bool {
			_, idx := py.FuzzyContain(newChoices[index], input)
			return idx >= 0
		}
	}
	idx, _ := FixedSelect(label, newChoices, searchFunction)
	return hit(idx)
}

// Confirm with y/n
func Confirm(label string, defaultY bool) bool {
	prompt := promptui.Prompt{
		Label:     label,
		IsConfirm: true,
	}
	if defaultY {
		prompt.Default = "y"
	} else {
		prompt.Default = "n"
	}

	result, _ := prompt.Run()

	result = strings.ToLower(result)
	if defaultY {
		return result != "n"
	}
	return !(result != "y")
}

// InputPassword with mask
func InputPassword(label string, validateFunc func(string) error) string {
	prompt := promptui.Prompt{
		Label:    label,
		Validate: validateFunc,
		Mask:     '*',
	}

	result, err := prompt.Run()

	if err != nil {
		panic(fmt.Sprintf("When input password %s:%v", label, err))
	}

	return strings.TrimSpace(result)
}

type InputOption func(*inputOption)

type Suggest struct {
	Text string
	Desc string
}

func (s Suggest) GetKey() string          { return s.Text }
func (s Suggest) convert() prompt.Suggest { return prompt.Suggest{Text: s.Text, Description: s.Desc} }

type inputOption struct {
	recentBucket      string
	browseFile        bool
	includeHiddenFile bool
	suggestions       []Suggest
	showHint          bool
	validateFn        func(string) error
}

func newInputOption() *inputOption {
	p := new(inputOption)
	p.validateFn = func(string) error { return nil }
	return p
}

func WithRecentName(ns string) InputOption {
	return func(opt *inputOption) {
		opt.recentBucket = ns
	}
}

func WithFileBrowser() InputOption {
	return func(opt *inputOption) {
		opt.browseFile = true
		opt.includeHiddenFile = false
	}
}

func WithHint() InputOption {
	return func(opt *inputOption) {
		opt.showHint = true
	}
}

func WithSuggestions(list []Suggest) InputOption {
	return func(opt *inputOption) {
		opt.suggestions = list
	}
}

func WithValidator(v func(string) error) InputOption {
	return func(opt *inputOption) {
		opt.validateFn = v
	}
}

func Input(label string, fns ...InputOption) string {
	v, _ := InterruptableInput(label, fns...)
	return v
}

func InterruptableInput(label string, fns ...InputOption) (text string, interrupted bool) {
	opt := newInputOption()
	for _, fn := range fns {
		fn(opt)
	}
	cache := getSuggestCache(opt.recentBucket)
	defer func() {
		if text != "" {
			cache.InsertItem(prompt.Suggest{
				Text:        text,
				Description: PromptTypeHistory,
			})
		}
	}()
	menu := func(d prompt.Document) []prompt.Suggest {
		// get all suggestions
		suggestions := fp.ListOf(opt.suggestions).Map(func(sg Suggest) prompt.Suggest {
			return sg.convert()
		}).MustGetResult().([]prompt.Suggest)

		var filterList []sgFilter

		filterList = append(filterList, fileBrowserCompleter(opt.browseFile, opt.includeHiddenFile))
		filterList = append(filterList, getFileHistoryCompleter(cache))
		filterList = append(filterList, hintCompleter(opt.showHint))
		filterList = append(filterList, removeInvalidCompleter())

		fp.ListOf(filterList).Foreach(func(fn sgFilter) {
			suggestions = fn(d, suggestions)
		})
		return suggestions
	}
	for {
		text, interrupted = prompt.Input(
			label+" ",
			menu,
			prompt.OptionPrefixTextColor(prompt.Blue),
		)
		text = strings.TrimSpace(text)
		if err := opt.validateFn(text); err != nil {
			fmt.Printf("%s", err.Error())
		} else {
			break
		}
	}
	return
}

func PressEnterToContinue() {
	fmt.Print("Press Enter to continue...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}

func PressEnterToContinueWithHint(hint string) {
	fmt.Print(hint)
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}

func fileDesc(file os.FileInfo) string {
	tp := PromptTypeFile
	if file.IsDir() {
		tp = PromptTypeDir
	}
	return fmt.Sprintf("%s mod:%s", tp, file.ModTime().Format("2006-01-02 15:04:05"))
}

func fileBrowserCompleter(browseFile, includeHiddenFile bool) sgFilter {
	getDir := func(s string) string {
		if strings.HasPrefix(s, `~`) {
			hd, _ := os.UserHomeDir()
			s = strings.Replace(s, `~`, hd, 1)
		}
		s, _ = filepath.Abs(s)
		if _, err := os.Stat(s); err != nil {
			return filepath.Dir(s)
		}
		return s
	}
	return func(d prompt.Document, sgIn []prompt.Suggest) []prompt.Suggest {
		if !browseFile {
			return sgIn
		}
		var sgList []prompt.Suggest
		dir := getDir(d.Text)
		fileMap := make(map[string]string)
		files, _ := ioutil.ReadDir(dir)
		for _, file := range files {
			if strings.HasPrefix(file.Name(), `.`) && !includeHiddenFile {
				continue
			}
			sgList = append(sgList, prompt.Suggest{Text: file.Name()})
			fileMap[file.Name()] = filepath.Join(dir, file.Name())
		}
		if !strings.HasSuffix(d.GetWordBeforeCursor(), `/`) {
			sgList = prompt.FilterContains(sgList, filepath.Base(d.GetWordBeforeCursor()), true)
		}
		for i := range sgList {
			sgList[i].Text = fileMap[sgList[i].Text]
		}
		return append(sgIn, sgList...)
	}
}

func hintCompleter(showHint bool) sgFilter {
	return func(d prompt.Document, suggestions []prompt.Suggest) []prompt.Suggest {
		if !showHint {
			for i := range suggestions {
				suggestions[i].Description = ""
			}
		}
		return suggestions
	}
}

func getFileHistoryCompleter(cache sgCache) sgFilter {
	return func(d prompt.Document, suggestions []prompt.Suggest) []prompt.Suggest {
		var history []prompt.Suggest
		cache.ListItem(&history)
		dup := make(map[string]*prompt.Suggest)
		if len(history) > 0 {
			for i, item := range history {
				dup[item.Text] = &history[i]
			}
			for _, sug := range suggestions {
				if v, ok := dup[sug.Text]; !ok {
					history = append(history, sug)
				} else {
					v.Description = sug.Description
				}
			}
			history = prompt.FilterContains(history, d.GetWordBeforeCursor(), true)
		}
		return append(history, suggestions...)
	}
}

func removeInvalidCompleter() sgFilter {
	return func(d prompt.Document, suggestions []prompt.Suggest) []prompt.Suggest {
		return fp.ListOf(suggestions).Reject(func(sg prompt.Suggest) bool {
			return strings.TrimSpace(sg.Text) == ``
		}).UniqBy(func(sg prompt.Suggest) string {
			return sg.Text
		}).MustGetResult().([]prompt.Suggest)
	}
}

type sgFilter func(prompt.Document, []prompt.Suggest) []prompt.Suggest

type sgCache interface {
	InsertItem(interface{}) error
	ListItem(interface{}) error
}

type nilCache int

func (nilCache) InsertItem(interface{}) error { return nil }
func (nilCache) ListItem(interface{}) error   { return nil }

func getSuggestCache(bucket string) sgCache {
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		return nilCache(0)
	}
	return MustNewHomeFileDB("cli-input").GetItemHistoryBucket(bucket, 5)
}

func reOrderChoicesByFreq(name string, choices []string) (newChoices []string, hit func(int) int) {
	key := strings.Join(choices, "-")
	db := MustNewHomeFileDB("cli-select" + name).GetBucketKV(key)
	counter := make(map[string]int)
	db.Get(key, &counter)

	newChoices = make([]string, len(choices))
	copy(newChoices, choices)
	sort.SliceStable(newChoices, func(i, j int) bool {
		return counter[newChoices[i]] > counter[newChoices[j]]
	})

	hit = func(idx int) int {
		if idx >= 0 && idx < len(newChoices) {
			counter[newChoices[idx]]++
			db.Put(key, counter)
			for i, v := range choices {
				if v == newChoices[idx] {
					return i
				}
			}
		}
		return -1
	}
	return
}
