//go:build unix

package load_test

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestLocalFileFIFORejectedWithoutBlocking(t *testing.T) {
	t.Parallel()

	pipe := filepath.Join(t.TempDir(), "page.html")
	if err := syscall.Mkfifo(pipe, 0o600); err != nil {
		t.Fatal(err)
	}

	loader := mustLoader(t)
	loader.Allow = []string{filepath.Dir(pipe)}
	done := make(chan error, 1)

	go func() {
		_, err := loader.Load(t.Context(), "file://"+pipe, defaultLP())
		done <- err
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("FIFO must be rejected")
		}
	case <-time.After(time.Second):
		// Release the blocked open before reporting the regression.
		file, err := os.OpenFile(pipe, os.O_WRONLY|syscall.O_NONBLOCK, 0)
		if err != nil {
			t.Fatalf("opening FIFO writer to release blocked load: %v", err)
		}

		_ = file.Close()

		<-done
		t.Fatal("opening a FIFO blocked before the regular-file check")
	}
}
