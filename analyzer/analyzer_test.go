package analyzer

import "testing"

var (
	testFile = ""
)

func TestLoadStackFile(t *testing.T) {
	f, err := LoadStackFile(testFile)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	t.Log(f)
}

func TestParseStackFile(t *testing.T) {
	f, _ := LoadStackFile(testFile)
	stack, err := NewStack(f)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	for i, th := range stack.Threads {
		t.Logf("%d - thread: %+v", i, th)
	}
}

func TestNewThread(t *testing.T) {
	line := "\"Attach Listener\" #2463 daemon prio=9 os_prio=0 tid=0x00007fda04035000 nid=0x804 waiting on condition [0x0000000000000000]"
	thread, err := NewThread(line)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	t.Logf("thread: %+v", thread)
}

func TestLineExtract(t *testing.T) {
	m := ExtractOne(` nid=([0-9a-fx,]+)`, " nid=123")
	if m != "123" {
		t.Log(m)
		t.FailNow()
	}
}
