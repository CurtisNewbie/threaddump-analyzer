package analyzer

import (
	"fmt"
	"io"
	"os"
	"regexp"
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

func (t *Thread) AddStackLine(line string) bool {
	var match string

	match = ExtractOne(`^\s+at (.*)`, line)
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

	stackFile := Stack{Content: content}
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

		if err := ParseStackLine(&stackFile, line); err != nil {
			return nil, fmt.Errorf("failed to ParseStackLine, line: %v, %v", line, err)
		}
	}

	// TODO: impl this later
	// this._identifyWaitedForSynchronizers();
	return &stackFile, nil
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
		parsed = stack.CurrentThread.AddStackLine(line)
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
