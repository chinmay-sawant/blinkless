package chrome_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	repoRoot            = "../.."
	evidenceKindGoTest  = "go-test"
	evidenceKindBrowser = "browser"
	statusCompleted     = "completed"
)

type caseManifest struct {
	CaseCount int            `json:"caseCount"`
	Cases     []manifestCase `json:"cases"`
}

type manifestCase struct {
	ID          string            `json:"id"`
	Source      string            `json:"source"`
	Fixture     string            `json:"fixture"`
	GoTarget    string            `json:"goTarget"`
	Expected    string            `json:"expected"`
	Status      string            `json:"status"`
	Combination string            `json:"combination"`
	Reason      string            `json:"reason"`
	Evidence    *manifestEvidence `json:"evidence"`
}

type manifestEvidence struct {
	Kind    string `json:"kind"`
	File    string `json:"file"`
	Test    string `json:"test"`
	Case    string `json:"case"`
	Browser string `json:"browser"`
	Verdict string `json:"verdict"`
}

func readManifest(tb testing.TB) caseManifest {
	tb.Helper()

	raw, err := os.ReadFile("manifest.json")
	if err != nil {
		tb.Fatalf("read manifest: %v", err)
	}

	var manifest caseManifest

	if err := json.Unmarshal(raw, &manifest); err != nil {
		tb.Fatalf("decode manifest: %v", err)
	}

	return manifest
}

func TestManifestHasFortyUniqueCases(t *testing.T) {
	t.Parallel()

	manifest := readManifest(t)
	if manifest.CaseCount != 40 {
		t.Fatalf("manifest caseCount = %d, want 40", manifest.CaseCount)
	}

	if len(manifest.Cases) != manifest.CaseCount {
		t.Fatalf("manifest cases = %d, want %d", len(manifest.Cases), manifest.CaseCount)
	}

	ids := make(map[string]struct{}, len(manifest.Cases))
	validTargets := map[string]bool{
		"layout-unit":      true,
		"chrome-reference": true,
		"golden-fixture":   true,
	}
	validStatuses := map[string]bool{
		"scaffold":    true,
		"completed":   true,
		"blocked":     true,
		"unsupported": true,
	}

	for _, item := range manifest.Cases {
		assertManifestCase(t, item, ids, validTargets, validStatuses)
	}
}

// TestManifestGoTestEvidenceResolves proves every go-test evidence pointer names
// a test function that exists in the file it cites.
func TestManifestGoTestEvidenceResolves(t *testing.T) {
	t.Parallel()

	manifest := readManifest(t)

	for _, item := range manifest.Cases {
		if item.Evidence == nil || item.Evidence.Kind != evidenceKindGoTest {
			continue
		}

		assertGoTestEvidence(t, item)
	}
}

// TestManifestBrowserEvidenceResolves proves every browser evidence pointer
// names an evidence file that exists and records the case id and browser
// version. The artifacts live under the ignored temps/ directory, so the test
// skips when that directory is absent.
func TestManifestBrowserEvidenceResolves(t *testing.T) {
	t.Parallel()

	manifest := readManifest(t)

	if _, err := os.Stat(filepath.Join(repoRoot, "temps")); errors.Is(err, os.ErrNotExist) {
		t.Skip("browser evidence under temps/ is not present in this checkout")
	}

	for _, item := range manifest.Cases {
		if item.Evidence == nil || item.Evidence.Kind != evidenceKindBrowser {
			continue
		}

		assertBrowserEvidence(t, item)
	}
}

func TestManifestSourcesExistWhenChromiumCheckoutIsPresent(t *testing.T) {
	t.Parallel()

	manifest := readManifest(t)
	root := filepath.Join(repoRoot, "chromium")

	if _, err := os.Stat(root); errors.Is(err, os.ErrNotExist) {
		t.Skip("local Chromium checkout is not present")
	}

	for _, item := range manifest.Cases {
		source := strings.SplitN(item.Source, "::", 2)[0]
		if _, err := os.Stat(filepath.Join(root, source)); err != nil {
			t.Errorf("case %q source %q: %v", item.ID, source, err)
		}
	}
}

func assertManifestCase(
	t *testing.T,
	item manifestCase,
	ids map[string]struct{},
	validTargets, validStatuses map[string]bool,
) {
	t.Helper()

	assertManifestMetadata(t, item, ids, validTargets, validStatuses)
	assertFixture(t, item)
}

func assertManifestMetadata(
	t *testing.T,
	item manifestCase,
	ids map[string]struct{},
	validTargets, validStatuses map[string]bool,
) {
	t.Helper()

	if item.ID == "" {
		t.Error("case has empty id")
	}

	if _, exists := ids[item.ID]; exists {
		t.Errorf("duplicate case id %q", item.ID)
	}

	ids[item.ID] = struct{}{}

	if !validTargets[item.GoTarget] {
		t.Errorf("case %q has unknown Go target %q", item.ID, item.GoTarget)
	}

	if !validStatuses[item.Status] {
		t.Errorf("case %q has unknown status %q", item.ID, item.Status)
	}

	if item.Source == "" || item.Combination == "" || item.Expected == "" {
		t.Errorf("case %q is missing source, combination, or expected behavior", item.ID)
	}

	assertEvidenceRule(t, item)
}

// assertEvidenceRule enforces the manifest contract: a completed case must
// carry evidence, every other status must carry a reason, and a browser
// evidence verdict of fail cannot be marked completed.
func assertEvidenceRule(t *testing.T, item manifestCase) {
	t.Helper()

	if item.Status == statusCompleted {
		if item.Evidence == nil {
			t.Errorf("case %q is completed without evidence", item.ID)
		}
	} else if strings.TrimSpace(item.Reason) == "" {
		t.Errorf("case %q has status %q without a reason", item.ID, item.Status)
	}

	if item.Evidence == nil {
		return
	}

	assertEvidenceMetadata(t, item)
}

func assertEvidenceMetadata(t *testing.T, item manifestCase) {
	t.Helper()

	evidence := item.Evidence

	if !safeRepoPath(evidence.File) {
		t.Errorf("case %q has unsafe evidence file %q", item.ID, evidence.File)
	}

	switch evidence.Kind {
	case evidenceKindGoTest:
		if evidence.Test == "" {
			t.Errorf("case %q go-test evidence names no test", item.ID)
		}
	case evidenceKindBrowser:
		assertBrowserEvidenceMetadata(t, item)
	default:
		t.Errorf("case %q has unknown evidence kind %q", item.ID, evidence.Kind)
	}
}

func assertBrowserEvidenceMetadata(t *testing.T, item manifestCase) {
	t.Helper()

	evidence := item.Evidence

	if evidence.Case == "" || evidence.Browser == "" {
		t.Errorf("case %q browser evidence is missing the case id or browser version", item.ID)
	}

	if evidence.Verdict != "pass" && evidence.Verdict != "fail" {
		t.Errorf("case %q browser evidence has unknown verdict %q", item.ID, evidence.Verdict)
	}

	if evidence.Verdict == "fail" && item.Status == statusCompleted {
		t.Errorf("case %q is completed but its browser evidence verdict is fail", item.ID)
	}
}

func assertGoTestEvidence(t *testing.T, item manifestCase) {
	t.Helper()

	evidence := item.Evidence

	if !strings.HasSuffix(evidence.File, "_test.go") {
		t.Errorf("case %q go-test evidence file %q is not a Go test file", item.ID, evidence.File)

		return
	}

	source, err := os.ReadFile(filepath.Join(repoRoot, evidence.File))
	if err != nil {
		t.Errorf("case %q read go-test evidence %q: %v", item.ID, evidence.File, err)

		return
	}

	if !strings.Contains(string(source), "func "+evidence.Test+"(") {
		t.Errorf("case %q names test %q, which %s does not define", item.ID, evidence.Test, evidence.File)
	}
}

func assertBrowserEvidence(t *testing.T, item manifestCase) {
	t.Helper()

	evidence := item.Evidence

	source, err := os.ReadFile(filepath.Join(repoRoot, evidence.File))
	if err != nil {
		t.Errorf("case %q read browser evidence %q: %v", item.ID, evidence.File, err)

		return
	}

	text := string(source)
	if !strings.Contains(text, evidence.Case) {
		t.Errorf("case %q browser evidence %q does not contain case %q", item.ID, evidence.File, evidence.Case)
	}

	if !strings.Contains(text, evidence.Browser) {
		t.Errorf("case %q browser evidence %q does not record browser %q", item.ID, evidence.File, evidence.Browser)
	}
}

func safeRepoPath(path string) bool {
	if path == "" || filepath.IsAbs(path) || filepath.Clean(path) != path {
		return false
	}

	return !strings.HasPrefix(path, "..")
}

func assertFixture(t *testing.T, item manifestCase) {
	t.Helper()

	if filepath.Clean(item.Fixture) != item.Fixture || !strings.HasPrefix(item.Fixture, "cases/") {
		t.Errorf("case %q has unsafe fixture path %q", item.ID, item.Fixture)

		return
	}

	fixture, err := os.ReadFile(item.Fixture)
	if err != nil {
		t.Errorf("case %q read fixture %q: %v", item.ID, item.Fixture, err)

		return
	}

	fixtureText := string(fixture)
	if !strings.HasPrefix(fixtureText, "<!doctype html>") {
		t.Errorf("case %q fixture does not start with a doctype", item.ID)
	}

	if !strings.Contains(fixtureText, "Port status: "+item.Status) {
		t.Errorf("case %q fixture lacks %q marker", item.ID, "Port status: "+item.Status)
	}
}
