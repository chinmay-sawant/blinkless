package load

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAllowedRootRejectsSymlinkSwappedAfterACLCheck(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	allowed := filepath.Join(dir, "public")

	if err := os.Mkdir(allowed, 0o700); err != nil {
		t.Fatal(err)
	}

	secret := filepath.Join(dir, "secret.html")
	if err := os.WriteFile(secret, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}

	candidate := filepath.Join(allowed, "page.html")
	if err := os.WriteFile(candidate, []byte("allowed"), 0o600); err != nil {
		t.Fatal(err)
	}

	controller := AccessController{AllowPrefixes: []string{allowed}}
	if !controller.Allowed(candidate) {
		t.Fatal("initial regular file must pass the ACL")
	}

	if err := os.Remove(candidate); err != nil {
		t.Fatal(err)
	}

	if err := os.Symlink(secret, candidate); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	file, err := openAllowedFile(allowed, candidate)
	if err == nil {
		_ = file.Close()

		t.Fatal("file replaced by an escaping symlink must not open")
	}
}
