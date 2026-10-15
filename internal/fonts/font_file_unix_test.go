//go:build unix

package fonts

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestScanFontDirSkipsFIFO(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	pipe := filepath.Join(dir, "block.ttf")

	if err := syscall.Mkfifo(pipe, 0o600); err != nil {
		t.Fatal(err)
	}

	done := make(chan *Registry, 1)

	go func() {
		done <- ScanFontDirs([]string{dir})
	}()

	select {
	case reg := <-done:
		if reg == nil {
			t.Fatal("nil registry")
		}

		if got := reg.Lookup([]string{"block"}, 400, false); got != nil {
			t.Fatal("FIFO was registered as a font")
		}
	case <-time.After(time.Second):
		file, err := os.OpenFile(pipe, os.O_WRONLY|syscall.O_NONBLOCK, 0)
		if err != nil {
			t.Fatalf("opening FIFO writer to release blocked scan: %v", err)
		}

		_ = file.Close()

		<-done
		t.Fatal("opening a FIFO blocked before the regular-file check")
	}
}

func TestReadFontFileSkipsNonRegular(t *testing.T) {
	t.Parallel()

	pipe := filepath.Join(t.TempDir(), "block.ttf")
	if err := syscall.Mkfifo(pipe, 0o600); err != nil {
		t.Fatal(err)
	}

	info, err := os.Lstat(pipe)
	if err != nil {
		t.Fatal(err)
	}

	if info.Mode().IsRegular() {
		t.Fatal("FIFO stat reported a regular file")
	}

	done := make(chan error, 1)

	go func() {
		_, readErr := readFontFile(pipe, info, nil)
		done <- readErr
	}()

	select {
	case readErr := <-done:
		if readErr == nil {
			t.Fatal("non-regular font file was read")
		}
	case <-time.After(time.Second):
		file, openErr := os.OpenFile(pipe, os.O_WRONLY|syscall.O_NONBLOCK, 0)
		if openErr != nil {
			t.Fatalf("opening FIFO writer to release blocked read: %v", openErr)
		}

		_ = file.Close()

		<-done
		t.Fatal("readFontFile opened a FIFO after Info said it was not regular")
	}
}
