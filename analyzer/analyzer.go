package analyzer

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
)

var (
	generatedIdCounter = 1
)

func LoadStackFile(file string) (string, error) {
	var f *os.File
	var err error

	if f, err = os.Open(file); err != nil {
		return "", fmt.Errorf("failed to open file %v, %v", file, err)
	}

	var buf []byte
	if buf, err = io.ReadAll(f); err != nil {
		return "", fmt.Errorf("failed to read file %v, %v", file, err)
	}

	return string(buf), nil
}

type Stack struct {
	Content       string
	CurrentThread *Thread
	Threads       []*Thread
	Ignored       []string
}

type Thread struct {
	DontKnow string
	Nid      string
	Tid      string
	Prio     string
	OsPrio   string
	Daemon   bool
	Number   string
	Group    string
	Name     string
	State    string
	Frames   []string

	WantNotificationOn  string
	WantToAcquire       string
	LocksHeld           []string
	SynchronizerClasses map[string]string
	ThreadState         string

	// Only synchronized(){} style locks
	ClassicalLocksHeld []string
}

func AddStackLine(t *Thread, line string) bool {
	var match = ExtractOne(`^\s+at (.*)`, line)
	if match != "" {
		t.Frames = append(t.Frames, match)
		return true
	}

	match = ExtractOne(`^\s*java.lang.Thread.State: (.*)`, line)
	if match != "" {
		t.ThreadState = match
		return true
	}

	var matches = ExtractMulti(`^\s+- (.*?) +<([x0-9a-f]+)> \(a (.*)\)`, line)
	if len(matches) > 0 {
		var state = matches[0]
		var id = matches[1]
		var className = matches[2]
		t.SynchronizerClasses[id] = className

		switch state {
		case "eliminated":
			// JVM internal optimization, not sure why it's in the
			// thread dump at all
			return true

		case "waiting on":
			t.WantNotificationOn = id
			return true

		case "parking to wait for":
			t.WantNotificationOn = id
			return true

		case "waiting to lock":
			t.WantToAcquire = id
			return true

		case "locked":
			if t.WantNotificationOn == id {
				// Lock is released while waiting for the notification
				return true
			}
			// Threads can take the same lock in different frames,
			// but we just want a mapping between threads and
			// locks so we must not list any lock more than once.
			t.LocksHeld = ArrayAddUnique(t.LocksHeld, id)
			t.ClassicalLocksHeld = ArrayAddUnique(t.ClassicalLocksHeld, id)
			return true

		default:
			return false
		}
	}

	matches = ExtractMulti(`^\s+- <([x0-9a-f]+)> \(a (.*)\)`, line)
	if match != "" {
		var lockId = matches[0]
		var lockClassName = matches[1]
		t.SynchronizerClasses[lockId] = lockClassName
		// Threads can take the same lock in different frames, but
		// we just want a mapping between threads and locks so we
		// must not list any lock more than once.
		t.LocksHeld = ArrayAddUnique(t.LocksHeld, lockId)
		return true
	}

	// LOCKED_OWNABLE_SYNCHRONIZERS
	if MatchPattern(`^\s+Locked ownable synchronizers:`, line) {
		// Ignore these lines
		return true
	}

	// NONE_HELD
	if MatchPattern(`^\s+- None`, line) {
		// Ignore these lines
		return true
	}
	return false
}

func (t *Thread) IsValid() bool {
	return t.Name != ""
}

func NewThread(line string) (*Thread, error) {
	t := Thread{}
	t.SynchronizerClasses = make(map[string]string)
	t.DontKnow = ExtractOne(`\[([0-9a-fx,]+)\]$`, line)
	t.Nid = ExtractOne(` nid=([0-9a-fx,]+)`, line)
	t.Tid = ExtractOne(` tid=([0-9a-fx,]+)`, line)
	if t.Tid == "" {
		t.Tid = ExtractOne(` - Thread t@([0-9a-fx]+)`, line)
	}
	t.Prio = ExtractOne(` prio=([0-9]+)`, line)
	t.OsPrio = ExtractOne(` os_prio=([0-9a-fx,]+)`, line)
	t.Daemon = ExtractOne(` (daemon)`, line) != ""
	t.Number = ExtractOne(` #([0-9]+)`, line)
	t.Group = ExtractOne(` group="(.*)"`, line)
	t.Name = ExtractOne(`^"(.*)" `, line)
	if t.Name == "" {
		t.Name = ExtractOne(`^"(.*)":?$`, line)
	}
	t.State = strings.TrimSpace(line)
	if t.Tid == "" {
		t.Tid = "generated-id-" + fmt.Sprintf("%d", generatedIdCounter)
		generatedIdCounter++
	}
	t.Frames = []string{}

	return &t, nil
}

func MatchPattern(pat string, ctn string) bool {
	rg := regexp.MustCompile(pat)
	return rg.MatchString(ctn)
}

func ExtractMulti(pat string, ctn string) []string {
	rg := regexp.MustCompile(pat)
	matched := rg.FindStringSubmatch(ctn)
	if len(matched) < 2 {
		return []string{}
	}
	return matched[1:]
}

func ExtractOne(pat string, ctn string) string {
	rg := regexp.MustCompile(pat)
	matched := rg.FindStringSubmatch(ctn)
	if len(matched) < 2 {
		return ""
	}
	return matched[1]
}

func NewStack(content string) (*Stack, error) {

	stack := Stack{Content: content}
	lines := strings.Split(content, "\n")

	for i := 0; i < len(lines); i++ {
		var line = lines[i]

		for IsIncompleteThreadHeader(line) {
			// Multi line thread name
			i++
			if i >= len(lines) {
				break
			}
			// Replace thread name newline with ", "
			line += ", " + lines[i]
		}

		if err := ParseStackLine(&stack, line); err != nil {
			return nil, fmt.Errorf("failed to ParseStackLine, line: %v, %v", line, err)
		}
	}

	for _, t := range stack.Threads {
		IdentifyWaitedForSynchronizers(t)
	}

	return &stack, nil
}

func ParseStackLine(stack *Stack, line string) error {
	var thread *Thread
	var err error
	var parsed = false

	if thread, err = NewThread(line); err != nil {
		return fmt.Errorf("failed to create new *Thread, %v", err)
	}
	if thread.IsValid() {
		stack.Threads = append(stack.Threads, thread)
		stack.CurrentThread = thread
		parsed = true
	} else if matched, err := regexp.MatchString(`^\s*$`, line); err == nil && matched {
		// We ignore empty lines, and lines containing only whitespace
		parsed = true
	} else if stack.CurrentThread != nil {
		parsed = AddStackLine(stack.CurrentThread, line)
	}
	if !parsed {
		stack.Ignored = append(stack.Ignored, line)
	}
	return nil
}

func IsIncompleteThreadHeader(line string) bool {
	rline := []rune(line)

	if len(rline) < 1 {
		return false
	}

	if rline[0] != '"' {
		// Thread headers start with ", this is not it
		return false
	}
	if strings.Contains(line, "prio=") {
		// Thread header contains "prio=" => we think it's complete
		return false
	}
	if strings.Contains(line, "Thread t@") {
		// Thread header contains a thread ID => we think it's complete
		return false
	}
	if string(rline[len(rline)-2:2]) == `":` {
		// Thread headers ending in ": are complete as seen in the example here:
		// https://github.com/spotify/threaddump-analyzer/issues/12
		return false
	}
	return true
}

func ArrayAddUnique(array []string, toAdd string) []string {
	for _, v := range array {
		if v == toAdd {
			return array
		}
	}
	array = append(array, toAdd)
	return array
}

func StackOutput(stack *Stack) string {
	if stack == nil || len(stack.Threads) < 1 {
		return ""
	}

	// stack summary
	var output = "Summary:\n\n"
	output += StackSummary(stack)
	output += "------------------------------------\n\n"

	// thread briefs
	SortThreads(stack.Threads)
	output += "Threads:\n\n"
	for i, t := range stack.Threads {
		output += fmt.Sprintf("\t%-4d ", i+1)
		output += ThreadBrief(t)
		output += "\n"
	}
	return output
}

func SortThreads(threads []*Thread) {
	sort.SliceStable(threads, func(i, j int) bool {
		t1, t2 := threads[i], threads[j]
		return strings.Compare(t1.Name, t2.Name) < 0
	})
}

func ThreadBrief(t *Thread) string {
	// var brief = ""
	// if t.Group != "" {
	// 	brief += t.Group
	// }
	// brief += "\""
	// brief += t.Name
	// brief += "\": "
	// brief += ThreadStatusBrief(t)
	// return brief

	return fmt.Sprintf("%s %-40s : %s", t.Group, t.Name, ThreadStatusBrief(t))
}

func ThreadStatusBrief(t *Thread) string {
	var s = ""

	if t.WantNotificationOn != "" {
		s += "awaiting notification on ["
		s += t.WantNotificationOn
		s += "]"
	} else if t.WantToAcquire != "" {
		s += "waiting to acquire ["
		s += t.WantToAcquire
		s += "]"
	} else if t.ThreadState == "TIMED_WAITING (sleeping)" {
		s += "sleeping"
	} else if t.ThreadState == "NEW" {
		s += "not started"
	} else if t.ThreadState == "TERMINATED" {
		s += "terminated"
	} else if t.ThreadState == "RUNNABLE" {
		s += "running"
	} else if t.ThreadState == "" {
		s += "non-Java thread"
	} else if len(t.Frames) < 1 {
		s += "non-Java thread"
	} else {
		// FIXME: Write something in the warnings section (that
		// doesn't exist yet)
		s += "Thread is "
		s += t.ThreadState
		s += " without waiting for anything?"
	}

	if len(t.LocksHeld) > 0 {
		s += ", holding ["
		for i, l := range t.LocksHeld {
			if i > 0 {
				s += ", "
			}
			s += l
		}
		s += "]"
	}

	return s
}

func IdentifyWaitedForSynchronizers(thread *Thread) {
	if !strings.Contains(thread.ThreadState, "TIMED_WAITING (on object monitor)") &&
		!strings.Contains(thread.ThreadState, "WAITING (on object monitor)") { // Not waiting for notification
		return
	}

	if thread.WantNotificationOn != "" {
		return
	}

	if len(thread.ClassicalLocksHeld) != 1 {
		return
	}

	thread.WantNotificationOn = thread.ClassicalLocksHeld[0]
}

func StackSummary(stack *Stack) string {
	out := fmt.Sprintf("In total %d threads found\n\n", len(stack.Threads))
	if len(stack.Threads) < 1 {
		return out
	}

	group := map[string][]*Thread{}
	for _, t := range stack.Threads {
		factName := ThreadFactoryName(t.Name)
		if l, ok := group[factName]; ok {
			group[factName] = append(l, t)
		} else {
			group[factName] = []*Thread{t}
		}
	}

	type PercentGroup struct {
		Percent float64
		Desc    string
	}

	grouped := []PercentGroup{}
	for k, v := range group {
		cnt := len(v)
		if cnt > 1 {
			percent := float64(cnt) / float64(len(stack.Threads)) * 100
			grouped = append(grouped, PercentGroup{
				Percent: percent,
				Desc:    fmt.Sprintf("\t%-40s: has %-3d threads with similar names (%.2f%%)", k, cnt, percent),
			})
		}
	}

	sort.SliceStable(grouped, func(i, j int) bool {
		return grouped[i].Percent > grouped[j].Percent
	})

	for _, g := range grouped {
		out += g.Desc
		out += "\n"
	}
	if len(grouped) > 0 {
		out += "\n"
	}
	return out
}

func ThreadFactoryName(name string) string {
	r := []rune(name)
	for i := len(r) - 1; i >= 0; i-- {
		if r[i] == '-' {
			return string(r[:i])
		}
		if r[i] < '0' || r[i] > '9' {
			return name
		}
	}
	return name
}
