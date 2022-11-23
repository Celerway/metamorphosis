package log

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func Test_Prefix(t *testing.T) {
	// make io.Writer buffers for stdout and stderr:
	stdoutBuffer := bytes.NewBuffer(nil)
	stderrBuffer := bytes.NewBuffer(nil)
	// Test that we can initialize the logger with a prefix
	l := NewWithPrefix(stdoutBuffer, stderrBuffer, "myprefix")
	// Test that the prefix is used in the output
	l.Info("test")
	// check that stdout buffer is not empty:
	if stdoutBuffer.Len() == 0 {
		t.Errorf("expected stdout buffer to not be empty")
	}

	logBuffer := stdoutBuffer.String()
	fmt.Println("logBuffer:", logBuffer)
	if !strings.Contains(logBuffer, "myprefix") {
		t.Errorf("expected prefix to be used in output")
	}
	// test that the message is correct:
	if !strings.Contains(logBuffer, "test") {
		t.Errorf("expected logline to contain 'test'")
	}
	// test that stderr buffer is empty:
	if stderrBuffer.Len() != 0 {
		t.Errorf("expected stderr buffer to be empty")
	}
}

func Test_Loglevel(t *testing.T) {
	// make io.Writer buffers for stdout and stderr:
	stdoutBuffer := bytes.NewBuffer(nil)
	stderrBuffer := bytes.NewBuffer(nil)
	// Test that we can initialize the logger with a prefix
	l := NewWithPrefix(stdoutBuffer, stderrBuffer, "myprefix")
	l.Trace("trace-message")
	// make sure that stdout and stderr are empty:
	if stdoutBuffer.Len() != 0 {
		t.Errorf("trace level: expected stdout buffer to be empty")
	}
	if stderrBuffer.Len() != 0 {
		t.Errorf("trace level: expected stderr buffer to be empty")
	}
	l.Debug("debug-message")
	if stdoutBuffer.Len() != 0 {
		t.Errorf("debug level: expected stdout buffer to be empty")
	}
	if stderrBuffer.Len() != 0 {
		t.Errorf("debug level: expected stderr buffer to be empty")
	}
	// now we output info and we expect it to be in stdout:
	l.Info("info-message")
	if stdoutBuffer.Len() == 0 {
		t.Errorf("info level: expected stdout buffer to not be empty")
	}
	currentStdoutSize := stdoutBuffer.Len()
	if stderrBuffer.Len() != 0 {
		t.Errorf("info level: expected stderr buffer to be empty")
	}
	// now we output warn and we expect it to be in stderr:
	l.Warn("warn-message")
	if stdoutBuffer.Len() != currentStdoutSize {
		t.Errorf("warn level: expected stdout buffer to not change")
	}
	if stderrBuffer.Len() == 0 {
		t.Errorf("warn level: expected stderr buffer to not be empty")
	}
	currentStderrSize := stderrBuffer.Len()
	// now we output error and we expect it to be in stderr:
	l.Error("error-message")
	if stdoutBuffer.Len() != currentStdoutSize {
		t.Errorf("error level: expected stdout buffer to not change")
	}
	if stderrBuffer.Len() == currentStderrSize {
		t.Errorf("error level: expected stderr buffer to change")
	}

}
