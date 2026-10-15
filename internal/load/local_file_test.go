package load

import (
	"net/url"
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

func TestNormalizeNetworkHostStripsZone(t *testing.T) {
	t.Parallel()

	if got := normalizeNetworkHost("[FE80::1%eth0]"); got != "[fe80::1]" {
		t.Fatalf("bracketed zone host = %q", got)
	}

	if got := normalizeNetworkHost("fe80::1%25eth0"); got != "fe80::1" {
		t.Fatalf("bare zone host = %q", got)
	}

	left, err := url.Parse("https://[2001:db8::1%25eth0]/a")
	if err != nil {
		t.Fatal(err)
	}

	right, err := url.Parse("https://[2001:db8::1]/a")
	if err != nil {
		t.Fatal(err)
	}

	if !sameNetworkHost(left, right) {
		t.Fatal("an IPv6 zone id must not make the same address look like another host")
	}
}
