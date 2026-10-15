package load_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/load"
	"github.com/chinmay-sawant/blinkless/internal/settings"
)

// TestRequestMetadataCaps proves unbounded operator metadata is rejected
// before any request is built.
func TestRequestMetadataCaps(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	loader := mustLoader(t)

	headers := map[string]string{}
	for i := range load.MaxCustomHeaders + 1 {
		headers["X-Test-Header-"+strconv.Itoa(i)] = "v"
	}

	pageLoad := defaultLP()
	pageLoad.CustomHeaders = headers

	if _, err := loader.Load(t.Context(), srv.URL, pageLoad); err == nil {
		t.Fatal("over-limit custom headers must be rejected")
	}

	cookies := map[string]string{}
	for i := range load.MaxCookies + 1 {
		cookies["c"+strconv.Itoa(i)] = "v"
	}

	pageLoad = defaultLP()
	pageLoad.Cookies = cookies

	if _, err := loader.Load(t.Context(), srv.URL, pageLoad); err == nil {
		t.Fatal("over-limit cookies must be rejected")
	}
}

// TestRangeMismatch proves a 206 response whose Content-Length disagrees
// with its Content-Range span is rejected instead of treated as a full body.
func TestRangeMismatch(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Range", "bytes 0-99/200")
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write([]byte("short"))
	}))
	defer srv.Close()

	loader := mustLoader(t)

	if _, err := loader.Load(t.Context(), srv.URL, defaultLP()); err == nil {
		t.Fatal("Content-Length/Range mismatch must be rejected")
	}
}

func TestContentRangeValidation(t *testing.T) {
	t.Parallel()

	for name, testCase := range map[string]struct {
		header string
		valid  bool
	}{
		"known total":   {header: "bytes 0-4/10", valid: true},
		"unknown total": {header: "bytes 0-4/*", valid: true},
		"past total":    {header: "bytes 0-4/4"},
		"trailing junk": {header: "bytes 0-4/10junk"},
		"overflow":      {header: "bytes 0-9223372036854775807/*"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Range", testCase.header)
				w.WriteHeader(http.StatusPartialContent)
				_, _ = w.Write([]byte("range"))
			}))
			defer server.Close()

			_, err := mustLoader(t).Load(t.Context(), server.URL, defaultLP())
			if (err == nil) != testCase.valid {
				t.Fatalf("Content-Range %q: error %v, want valid=%v", testCase.header, err, testCase.valid)
			}
		})
	}
}

func TestContentRangeChecksChunkedBodyLength(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, _ *http.Request) {
		responseWriter.Header().Set("Content-Range", "bytes 0-9/20")
		responseWriter.WriteHeader(http.StatusPartialContent)

		if err := http.NewResponseController(responseWriter).Flush(); err != nil {
			t.Error(err)

			return
		}

		_, _ = responseWriter.Write([]byte("short"))
	}))
	defer server.Close()

	if _, err := mustLoader(t).Load(t.Context(), server.URL, defaultLP()); err == nil {
		t.Fatal("chunked body length must match its Content-Range span")
	}
}

func TestSetMaxRedirectsConcurrentWithFetch(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/start" {
			http.Redirect(responseWriter, request, "/end", http.StatusFound)

			return
		}

		_, _ = responseWriter.Write([]byte("ok"))
	}))
	defer server.Close()

	loader := mustLoader(t)

	done := make(chan struct{})
	go func() {
		defer close(done)

		for index := range 1000 {
			if err := loader.SetMaxRedirects(1 + index%2); err != nil {
				t.Error(err)

				return
			}
		}
	}()

	for range 16 {
		if _, err := loader.Load(t.Context(), server.URL+"/start", defaultLP()); err != nil {
			t.Error(err)
		}
	}

	<-done
}

// TestProxyErrorRedactsCredentials proves proxy configuration errors never
// echo the configured password.
func TestProxyErrorRedactsCredentials(t *testing.T) {
	t.Parallel()

	for _, proxy := range []string{
		"ftp://operator:supersecretpw@proxy.example.com:8080",
		"http://operator:supersecretpw@proxy.example.com:bad",
		"http://operator:supersecretpw@proxy.example.com/%zz",
	} {
		global := settings.LoadGlobal{Proxy: proxy}

		_, err := load.NewLoaderWithError(global)
		if err == nil {
			t.Fatalf("invalid proxy %q must be rejected", proxy)
		}

		if strings.Contains(err.Error(), "supersecretpw") {
			t.Fatalf("proxy error leaks credentials: %v", err)
		}
	}
}

// TestFileHostCaseVariants proves host canonicalization (case, trailing
// dot) cannot smuggle a remote file host past the guard.
func TestFileHostCaseVariants(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "page.html")

	if err := os.WriteFile(filePath, []byte("<html>ok</html>"), 0o600); err != nil {
		t.Fatal(err)
	}

	pageLoad := defaultLP()
	pageLoad.BlockLocalFileAccess = false
	loader := mustLoader(t)
	loader.EnableLocalFileAccess = true

	for _, url := range []string{
		"file://LOCALHOST" + filePath,
		"file://localhost." + filePath,
	} {
		if _, err := loader.Load(t.Context(), url, pageLoad); err != nil {
			t.Errorf("local file host variant %q must load: %v", url, err)
		}
	}

	if _, err := loader.Load(t.Context(), "file://EVIL.example.com"+filePath, pageLoad); err == nil {
		t.Error("remote file host must be refused regardless of case")
	}
}

// TestRestrictedRedirectTrailingDotSameHost proves redirect host comparison
// canonicalizes trailing dots: 127.0.0.1.:port is the same host as
// 127.0.0.1:port, not a cross-host redirect. The redirect stays on one
// server (same port); only the hostname gains the trailing FQDN dot.
func TestRestrictedRedirectTrailingDotSameHost(t *testing.T) {
	t.Parallel()

	var srv *httptest.Server

	srv = httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/redirect" {
			parsed, err := url.Parse(srv.URL)
			if err != nil {
				t.Error(err)
				http.Error(respWriter, "bad server URL", http.StatusInternalServerError)

				return
			}

			dotted := "http://" + parsed.Hostname() + ".:" + parsed.Port() + "/"
			http.Redirect(respWriter, req, dotted, http.StatusFound)

			return
		}

		_, _ = respWriter.Write([]byte("same host"))
	}))
	defer srv.Close()

	parsed, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}

	policy := load.RestrictedNetworkPolicy()
	policy.AllowedHosts = []string{parsed.Hostname()}

	loader, err := load.NewLoaderWithNetworkPolicy(settings.LoadGlobal{}, policy)
	if err != nil {
		t.Fatal(err)
	}

	res, err := loader.Load(t.Context(), srv.URL+"/redirect", defaultLP())
	if err != nil {
		t.Fatalf("trailing-dot redirect must be same-host: %v", err)
	}

	if string(res.Body) != "same host" {
		t.Fatalf("body = %q, want %q", res.Body, "same host")
	}
}

// TestMalformedFileURLFallback proves a malformed file:// URL is refused
// instead of being treated as a plain local path.
func TestMalformedFileURLFallback(t *testing.T) {
	t.Parallel()

	pageLoad := defaultLP()
	pageLoad.BlockLocalFileAccess = false
	loader := mustLoader(t)
	loader.EnableLocalFileAccess = true

	_, err := loader.Load(t.Context(), "file://foo bar/baz", pageLoad)
	if err == nil {
		t.Fatal("malformed file URL must be refused")
	}

	if !strings.Contains(err.Error(), "blocked") {
		t.Fatalf("error = %v, want a blocked-access refusal", err)
	}
}

func TestACLExactFileDoesNotAllowSibling(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	allowed := filepath.Join(dir, "allowed.html")
	sibling := filepath.Join(dir, "secret.html")

	for _, path := range []string{allowed, sibling} {
		if err := os.WriteFile(path, []byte("page"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	loader := mustLoader(t)
	loader.Allow = []string{allowed}

	if _, err := loader.Load(t.Context(), allowed, defaultLP()); err != nil {
		t.Fatalf("explicit allowed file: %v", err)
	}

	if _, err := loader.Load(t.Context(), sibling, defaultLP()); err == nil {
		t.Fatal("allowing one file must not allow siblings")
	}
}

func TestACLFilesystemRootPrefix(t *testing.T) {
	t.Parallel()

	page := filepath.Join(t.TempDir(), "page.html")
	if err := os.WriteFile(page, []byte("page"), 0o600); err != nil {
		t.Fatal(err)
	}

	loader := mustLoader(t)
	loader.Allow = []string{filepath.VolumeName(page) + string(filepath.Separator)}

	if _, err := loader.Load(t.Context(), page, defaultLP()); err != nil {
		t.Fatalf("file beneath an allowed filesystem root: %v", err)
	}
}
