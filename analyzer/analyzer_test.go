package analyzer

import (
	"os"
	"strings"
	"testing"
)

var (
	testFile = "../dump.txt"
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

func TestThreadBrief(t *testing.T) {
	s := `"JobExecutor[org.camunda.bpm.engine.spring.components.jobexecutor.SpringJobExecutor]" #225 prio=5 os_prio=0 tid=0x00007f724f2b0800 nid=0xe8 in Object.wait() [0x00007f7025fea000]
   java.lang.Thread.State: TIMED_WAITING (on object monitor)
	at java.lang.Object.wait(Native Method)
	at org.camunda.bpm.engine.impl.jobexecutor.AcquireJobsRunnable.suspendAcquisition(AcquireJobsRunnable.java:51)
	- locked <0x00000007134b90c8> (a java.lang.Object)
	at org.camunda.bpm.engine.impl.jobexecutor.SequentialJobAcquisitionRunnable.run(SequentialJobAcquisitionRunnable.java:104)
	- locked <0x00000007134b90d8> (a org.camunda.bpm.engine.impl.jobexecutor.SequentialJobAcquisitionRunnable)
	at java.lang.Thread.run(Thread.java:750)`

	lines := strings.Split(s, "\n")
	thread, err := NewThread(lines[0])
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	for i, l := range lines {
		if i > 0 {
			if ok := AddStackLine(thread, l); !ok {
				t.Log(l)
				t.FailNow()
			}
		}
	}
	IdentifyWaitedForSynchronizers(thread)

	t.Logf("%+v", thread)
	brief := ThreadBrief(thread)
	t.Log(brief)
}

func TestStackOutput(t *testing.T) {
	f, _ := LoadStackFile(testFile)
	stack, err := NewStack(f)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	out := StackOutput(stack)
	// t.Log(out)
	os.WriteFile("out.txt", []byte(out), os.ModePerm)
}

func TestThreadFactoryName(t *testing.T) {
	name := "redisson-3-3"
	factName := ThreadFactoryName(name)
	if factName != "redisson-3" {
		t.Log(factName)
		t.FailNow()
	}
}
