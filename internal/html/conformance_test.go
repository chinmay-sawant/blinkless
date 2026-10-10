//nolint:all // conformance harness: corpus format parsing and diff plumbing, not product code
package html

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// TestHTMLConformance runs the pinned html5lib-tests corpus vendored under
// testdata/html-conformance/. The manifest is the fixed contract described in
// the Phase 2 checklist (HTML-BASE-02):
//
//	{"upstream":{"repo":...,"revision":...,"license":...,"licenseFile":...},
//	 "scripting":false,
//	 "categories":[{"id":"tokenizer","format":"html5lib-tokenizer-json",
//	                "files":["tokenizer/*.test"]},
//	               {"id":"tree-construction","format":"html5lib-tree-dat",
//	                "files":["tree-construction/*.dat"]}],
//	 "unsupported":[...]}
//
// Comparison rules:
//   - Tokenizer cases compare the full token stream (coalesced Character
//     runs): token kind, tag names, attribute maps, self-closing flag,
//     DOCTYPE name/public/system/force-quirks, and exact text data.
//   - Tree cases compare node kinds, parent/child order, namespaces, exact
//     text and comment data, attribute name/value pairs (with foreign
//     attribute namespaces), and the doctype node.
//   - Fields the engine cannot represent are never silently skipped. A case
//     whose only remaining difference is a missing engine field is counted
//     unsupported with a reason (processing instructions, document fragments,
//     tokenizer initial states other than Data state, XML-violation
//     coercions, lone surrogates that Go strings cannot carry). A case that
//     also differs in representable behavior is counted failed.
//   - Template contents are compared structurally: the engine keeps them in
//     Node.Contents and the runner emits them as a "content" node.
//   - #document-fragment records are unsupported until context-aware fragment
//     parsing lands; they are counted separately with the reason
//     "document-fragment".
//   - Parser mismatches are baseline evidence, not test failures. The test
//     fails only on harness problems (missing manifest fields, unreadable
//     corpus files, malformed records). Set HTML_CONFORMANCE_STRICT=1 to
//     fail on any failed case.
//
// Evidence: the full baseline (counts, non-passed case ids, reasons) is
// written to temps/html-conformance/engine-baseline.json under the repo root,
// and capped detailed diffs to temps/html-conformance/failures.txt. Detail
// depth is controlled by HTML_CONFORMANCE_DETAIL (default 10).
//
// When the corpus directory or its manifest is not present yet, the test
// skips with an explicit zero-count message instead of failing the package.
func TestHTMLConformance(t *testing.T) {
	t.Parallel()

	dir, found := findConformanceDir()
	if !found {
		t.Skipf("html conformance corpus absent: testdata/%s not found; counts: total=0 passed=0 failed=0 skipped=0 unsupported=0 (reason: corpus not vendored yet)", conformanceDirName)
	}

	manifestPath := filepath.Join(dir, "manifest.json")
	if _, err := os.Stat(manifestPath); err != nil {
		t.Skipf("html conformance manifest absent: %s; counts: total=0 passed=0 failed=0 skipped=0 unsupported=0 (reason: manifest not vendored yet)", manifestPath)
	}

	manifest := loadConformanceManifest(t, dir, manifestPath)
	report := runConformanceCorpus(t, dir, manifest)
	baselinePath := writeConformanceBaseline(t, report)

	logConformanceReport(t, report, baselinePath)

	if os.Getenv(conformanceStrictEnv) == "1" {
		if failed := countStatus(report, statusFailed); failed > 0 {
			t.Errorf("html conformance strict mode: %d failed cases; baseline: %s", failed, baselinePath)
		}
	}
}

const (
	conformanceDirName   = "html-conformance"
	conformanceStrictEnv = "HTML_CONFORMANCE_STRICT"
	conformanceDetailEnv = "HTML_CONFORMANCE_DETAIL"

	conformanceDefaultDetail = 10
	conformanceMaxDiffLines  = 200

	statusPassed      = "passed"
	statusFailed      = "failed"
	statusSkipped     = "skipped"
	statusUnsupported = "unsupported"

	formatTokenizerJSON = "html5lib-tokenizer-json"
	formatTreeDAT       = "html5lib-tree-dat"

	dataStateName = "Data state"

	reasonProcessingInstr = "processing-instruction-unsupported"
	reasonInitialState    = "initial-state:"
	reasonLoneSurrogate   = "lone-surrogate-input"
	reasonEngineError     = "engine-error: "
	reasonScriptingOnOnly = "scripting-on-only"
	reasonXMLViolation    = "xml-violation-coercion"
	reasonManifestEntry   = "manifest-unsupported: "

	kindDocument = "document"
	kindElement  = "element"
	kindText     = "text"
	kindComment  = "comment"
	kindDoctype  = "doctype"
	kindContent  = "content"
	kindPI       = "pi"

	tokenCharacter = "Character"
	tokenStartTag  = "StartTag"
	tokenEndTag    = "EndTag"
	tokenComment   = "Comment"
	tokenDoctype   = "DOCTYPE"

	dumpIndentStep   = 2
	dumpCommentOpen  = "<!-- "
	dumpCommentClose = " -->"

	cfAttrPairSize = 2
)

// cfAttr is one attribute in the neutral comparison form. ns is the dump
// namespace designator (xlink/xml/xmlns) and stays empty for engine values,
// which cannot carry namespaces.
type cfAttr struct {
	ns    string
	name  string
	value string
}

type cfDoctype struct {
	name        string
	public      string
	publicSet   bool
	system      string
	systemSet   bool
	forceQuirks bool
}

// cfNode is the neutral tree form shared by parsed .dat dumps and engine
// trees. doctypeOK is false when the engine doctype text could not be parsed
// into name/public/system fields.
type cfNode struct {
	kind      string
	ns        string
	name      string
	data      string
	doctype   cfDoctype
	doctypeOK bool
	attrs     []cfAttr
	children  []*cfNode
}

type cfToken struct {
	kind        string
	name        string
	data        string
	attrs       []cfAttr
	selfClosing bool
	doctype     cfDoctype
}

type conformanceUpstream struct {
	Repo        string `json:"repo"`
	Revision    string `json:"revision"`
	License     string `json:"license"`
	LicenseFile string `json:"licenseFile"`
}

type conformanceCategory struct {
	ID     string   `json:"id"`
	Format string   `json:"format"`
	Files  []string `json:"files"`
}

type conformanceManifest struct {
	Upstream    conformanceUpstream   `json:"upstream"`
	Scripting   bool                  `json:"scripting"`
	Categories  []conformanceCategory `json:"categories"`
	Unsupported []string              `json:"unsupported"`
}

type conformanceCase struct {
	ID       string
	Category string
	Status   string
	Reason   string
	Detail   string
}

type conformanceStats struct {
	total       int
	passed      int
	failed      int
	skipped     int
	unsupported int
	reasons     map[string]int
}

type conformanceReport struct {
	dir        string
	upstream   conformanceUpstream
	scripting  bool
	order      []string
	categories map[string]*conformanceStats
	cases      []conformanceCase
	notes      []string
}

func newConformanceReport(dir string, manifest conformanceManifest) *conformanceReport {
	return &conformanceReport{
		dir:        dir,
		upstream:   manifest.Upstream,
		scripting:  manifest.Scripting,
		categories: map[string]*conformanceStats{},
	}
}

func (r *conformanceReport) add(c conformanceCase) {
	stats := r.categories[c.Category]
	if stats == nil {
		stats = &conformanceStats{reasons: map[string]int{}}
		r.categories[c.Category] = stats
		r.order = append(r.order, c.Category)
	}

	stats.total++

	switch c.Status {
	case statusPassed:
		stats.passed++
	case statusFailed:
		stats.failed++
	case statusSkipped:
		stats.skipped++
	case statusUnsupported:
		stats.unsupported++
	}

	key := c.Status
	if c.Reason != "" {
		key += ": " + c.Reason
	}

	stats.reasons[key]++

	r.cases = append(r.cases, c)
}

func (r *conformanceReport) addCase(category, id, status, reason, detail string) {
	r.add(conformanceCase{ID: id, Category: category, Status: status, Reason: reason, Detail: detail})
}

// findConformanceDir walks up from the test working directory looking for
// testdata/html-conformance. The corpus is vendored at the repo root.
func findConformanceDir() (string, bool) {
	dir, err := os.Getwd()
	if err != nil {
		return "", false
	}

	for {
		candidate := filepath.Join(dir, "testdata", conformanceDirName)
		if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
			return candidate, true
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}

		dir = parent
	}
}

func loadConformanceManifest(t *testing.T, dir, path string) conformanceManifest {
	t.Helper()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("conformance harness: read manifest %s: %v", path, err)
	}

	var manifest conformanceManifest

	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("conformance harness: parse manifest %s: %v", path, err)
	}

	if manifest.Upstream.Repo == "" || manifest.Upstream.Revision == "" || manifest.Upstream.License == "" {
		t.Fatalf("conformance harness: %s: upstream repo, revision and license must all be recorded", path)
	}

	if manifest.Scripting {
		t.Fatalf("conformance harness: %s: manifest requests scripting=true; this runner implements the pinned scripting=false mode only", path)
	}

	if len(manifest.Categories) == 0 {
		t.Fatalf("conformance harness: %s: manifest has no categories", path)
	}

	if manifest.Upstream.LicenseFile != "" {
		licensePath := filepath.Join(dir, filepath.FromSlash(manifest.Upstream.LicenseFile))
		if _, statErr := os.Stat(licensePath); statErr != nil {
			t.Errorf("conformance harness: declared license file missing: %s", licensePath)
		}
	}

	return manifest
}

type conformanceCategoryPlan struct {
	category    conformanceCategory
	files       []string
	wholeReason string
	fileReasons map[string]string
}

func runConformanceCorpus(t *testing.T, dir string, manifest conformanceManifest) *conformanceReport {
	t.Helper()

	report := newConformanceReport(dir, manifest)
	plans, notes := planConformanceCategories(t, dir, manifest)
	report.notes = append(report.notes, notes...)

	for _, plan := range plans {
		switch plan.category.Format {
		case formatTokenizerJSON:
			runTokenizerCategory(t, plan, report)
		case formatTreeDAT:
			runTreeCategory(t, plan, report, manifest.Scripting)
		default:
			for _, file := range plan.files {
				report.addCase(plan.category.ID, plan.category.ID+"/"+file, statusUnsupported, "unknown-format:"+plan.category.Format, "")
			}

			t.Errorf("conformance harness: category %q has unknown format %q", plan.category.ID, plan.category.Format)
		}
	}

	return report
}

// planConformanceCategories expands the manifest file globs once and resolves
// the manifest "unsupported" entries: an entry matching a category id marks
// the whole category, an entry matching a file marks that file. Entries that
// match nothing are surfaced as notes instead of being ignored silently.
func planConformanceCategories(t *testing.T, dir string, manifest conformanceManifest) ([]conformanceCategoryPlan, []string) {
	t.Helper()

	plans := make([]conformanceCategoryPlan, 0, len(manifest.Categories))

	for _, cat := range manifest.Categories {
		plans = append(plans, conformanceCategoryPlan{
			category:    cat,
			files:       expandCategoryFiles(t, dir, cat),
			fileReasons: map[string]string{},
		})
	}

	var notes []string

	for _, entry := range manifest.Unsupported {
		matched := false

		for i := range plans {
			if entry == plans[i].category.ID {
				plans[i].wholeReason = reasonManifestEntry + entry
				matched = true

				continue
			}

			for _, file := range plans[i].files {
				glob, globErr := filepath.Match(entry, file)
				if (globErr == nil && glob) || entry == file {
					plans[i].fileReasons[file] = reasonManifestEntry + entry
					matched = true
				}
			}
		}

		if !matched {
			notes = append(notes, fmt.Sprintf("manifest unsupported entry %q matched no category id and no vendored file", entry))
		}
	}

	return plans, notes
}

func expandCategoryFiles(t *testing.T, dir string, cat conformanceCategory) []string {
	t.Helper()

	var files []string

	for _, pattern := range cat.Files {
		matches, err := filepath.Glob(filepath.Join(dir, filepath.FromSlash(pattern)))
		if err != nil {
			t.Fatalf("conformance harness: category %q pattern %q: %v", cat.ID, pattern, err)
		}

		if len(matches) == 0 {
			t.Errorf("conformance harness: category %q pattern %q matched no vendored files", cat.ID, pattern)
		}

		files = append(files, matches...)
	}

	sort.Strings(files)

	return dedupeStrings(files)
}

func dedupeStrings(values []string) []string {
	out := values[:0]

	for i, value := range values {
		if i == 0 || value != values[i-1] {
			out = append(out, value)
		}
	}

	return out
}

func relativeSlash(dir, path string) string {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return filepath.ToSlash(path)
	}

	return filepath.ToSlash(rel)
}

func cutWord(s string) (string, string) {
	idx := strings.IndexAny(s, " \t\n\r\f")
	if idx < 0 {
		return s, ""
	}

	return s[:idx], s[idx:]
}

func cutQuoted(s string) (string, string, bool) {
	if s == "" || (s[0] != '"' && s[0] != '\'') {
		return "", s, false
	}

	quote := s[0]

	end := strings.IndexByte(s[1:], quote)
	if end < 0 {
		return s[1:], "", false
	}

	return s[1 : end+1], s[end+2:], true
}

// --- dump rendering and diffing ---

func renderDoctype(doctype cfDoctype) string {
	var b strings.Builder

	b.WriteString("<!DOCTYPE " + doctype.name)

	if doctype.publicSet || doctype.systemSet {
		b.WriteString(` "` + escapeDumpText(doctype.public) + `" "` + escapeDumpText(doctype.system) + `"`)
	}

	b.WriteString(">")

	return b.String()
}

func quoteDump(s string) string {
	return `"` + escapeDumpText(s) + `"`
}

func escapeDumpText(s string) string {
	s = strings.ReplaceAll(s, "\r", `\r`)
	s = strings.ReplaceAll(s, "\n", `\n`)

	return s
}

func mismatchDetail(mismatch *cfMismatch) string {
	return "at " + mismatch.path + "\n" + diffLines(mismatch.want, mismatch.got)
}

func diffLines(want, got []string) string {
	truncated := false

	if len(want) > conformanceMaxDiffLines {
		want = want[:conformanceMaxDiffLines]
		truncated = true
	}

	if len(got) > conformanceMaxDiffLines {
		got = got[:conformanceMaxDiffLines]
		truncated = true
	}

	table := make([][]int, len(want)+1)
	for i := range table {
		table[i] = make([]int, len(got)+1)
	}

	for i := len(want) - 1; i >= 0; i-- {
		for j := len(got) - 1; j >= 0; j-- {
			switch {
			case want[i] == got[j]:
				table[i][j] = table[i+1][j+1] + 1
			case table[i+1][j] >= table[i][j+1]:
				table[i][j] = table[i+1][j]
			default:
				table[i][j] = table[i][j+1]
			}
		}
	}

	var b strings.Builder

	i, j := 0, 0

	for i < len(want) && j < len(got) {
		switch {
		case want[i] == got[j]:
			fmt.Fprintf(&b, "  %s\n", want[i])
			i++
			j++
		case table[i+1][j] >= table[i][j+1]:
			fmt.Fprintf(&b, "- %s\n", want[i])
			i++
		default:
			fmt.Fprintf(&b, "+ %s\n", got[j])
			j++
		}
	}

	for ; i < len(want); i++ {
		fmt.Fprintf(&b, "- %s\n", want[i])
	}

	for ; j < len(got); j++ {
		fmt.Fprintf(&b, "+ %s\n", got[j])
	}

	if truncated {
		b.WriteString("... (diff truncated)\n")
	}

	return b.String()
}

// --- reporting and baseline evidence ---

func logConformanceReport(t *testing.T, report *conformanceReport, baselinePath string) {
	t.Helper()

	t.Logf("html conformance corpus: %s", report.dir)
	t.Logf("upstream: %s @ %s (license %s, file %s, scripting=%t)",
		report.upstream.Repo, report.upstream.Revision, report.upstream.License, report.upstream.LicenseFile, report.scripting)

	for _, category := range report.order {
		stats := report.categories[category]
		t.Logf("%s: total=%d passed=%d failed=%d skipped=%d unsupported=%d",
			category, stats.total, stats.passed, stats.failed, stats.skipped, stats.unsupported)

		for _, key := range sortedReasonKeys(stats.reasons) {
			t.Logf("  %s: %d", key, stats.reasons[key])
		}
	}

	for _, note := range report.notes {
		t.Logf("manifest note: %s", note)
	}

	if baselinePath != "" {
		t.Logf("baseline: %s", baselinePath)
	}

	detail := conformanceDetailLimit()
	shown := 0

	for _, c := range report.cases {
		if shown >= detail {
			break
		}

		if c.Status == statusFailed && c.Detail != "" {
			t.Logf("case %s failed (%s):\n%s", c.ID, c.Reason, c.Detail)
			shown++
		}
	}
}

func sortedReasonKeys(reasons map[string]int) []string {
	keys := make([]string, 0, len(reasons))
	for key := range reasons {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	return keys
}

func countStatus(report *conformanceReport, status string) int {
	total := 0

	for _, c := range report.cases {
		if c.Status == status {
			total++
		}
	}

	return total
}

type conformanceBaselineCounts struct {
	Total       int            `json:"total"`
	Passed      int            `json:"passed"`
	Failed      int            `json:"failed"`
	Skipped     int            `json:"skipped"`
	Unsupported int            `json:"unsupported"`
	Reasons     map[string]int `json:"reasons,omitempty"`
}

type conformanceBaselineCase struct {
	ID       string `json:"id"`
	Category string `json:"category"`
	Status   string `json:"status"`
	Reason   string `json:"reason,omitempty"`
}

type conformanceBaselineCorpus struct {
	Dir       string `json:"dir"`
	Repo      string `json:"repo"`
	Revision  string `json:"revision"`
	License   string `json:"license"`
	Scripting bool   `json:"scripting"`
}

type conformanceBaseline struct {
	Corpus    conformanceBaselineCorpus            `json:"corpus"`
	Counts    map[string]conformanceBaselineCounts `json:"counts"`
	NonPassed []conformanceBaselineCase            `json:"nonPassed"`
	Notes     []string                             `json:"notes,omitempty"`
}

// writeConformanceBaseline records machine-readable evidence under the
// gitignored temps/html-conformance/ directory at the repo root and returns
// the baseline path ("" when the repo root cannot be located).
func writeConformanceBaseline(t *testing.T, report *conformanceReport) string {
	t.Helper()

	root, ok := findRepoRoot()
	if !ok {
		t.Logf("baseline not written: repo root (go.mod) not found from %s", report.dir)

		return ""
	}

	outDir := filepath.Join(root, "temps", conformanceDirName)
	if err := os.MkdirAll(outDir, 0o750); err != nil {
		t.Logf("baseline not written: %v", err)

		return ""
	}

	baseline := conformanceBaseline{
		Corpus: conformanceBaselineCorpus{
			Dir:       report.dir,
			Repo:      report.upstream.Repo,
			Revision:  report.upstream.Revision,
			License:   report.upstream.License,
			Scripting: report.scripting,
		},
		Counts: map[string]conformanceBaselineCounts{},
		Notes:  report.notes,
	}

	for _, category := range report.order {
		stats := report.categories[category]
		baseline.Counts[category] = conformanceBaselineCounts{
			Total:       stats.total,
			Passed:      stats.passed,
			Failed:      stats.failed,
			Skipped:     stats.skipped,
			Unsupported: stats.unsupported,
			Reasons:     stats.reasons,
		}
	}

	for _, c := range report.cases {
		if c.Status == statusPassed {
			continue
		}

		baseline.NonPassed = append(baseline.NonPassed, conformanceBaselineCase{
			ID:       c.ID,
			Category: c.Category,
			Status:   c.Status,
			Reason:   c.Reason,
		})
	}

	data, err := json.MarshalIndent(baseline, "", "  ")
	if err != nil {
		t.Logf("baseline not written: %v", err)

		return ""
	}

	baselinePath := filepath.Join(outDir, "engine-baseline.json")
	if err := os.WriteFile(baselinePath, append(data, '\n'), 0o600); err != nil {
		t.Logf("baseline not written: %v", err)

		return ""
	}

	writeConformanceFailures(t, outDir, report)

	return baselinePath
}

func writeConformanceFailures(t *testing.T, outDir string, report *conformanceReport) {
	t.Helper()

	limit := conformanceDetailLimit()

	var b strings.Builder

	for _, c := range report.cases {
		if limit <= 0 {
			break
		}

		if c.Status == statusFailed && c.Detail != "" {
			fmt.Fprintf(&b, "case %s (%s) [%s]\n%s\n\n", c.ID, c.Reason, c.Category, c.Detail)
			limit--
		}
	}

	path := filepath.Join(outDir, "failures.txt")
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		t.Logf("failures file not written: %v", err)
	}
}

func findRepoRoot() (string, bool) {
	dir, err := os.Getwd()
	if err != nil {
		return "", false
	}

	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir, true
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}

		dir = parent
	}
}

func conformanceDetailLimit() int {
	value := os.Getenv(conformanceDetailEnv)
	if value == "" {
		return conformanceDefaultDetail
	}

	n, err := strconv.Atoi(value)
	if err != nil || n < 0 {
		return conformanceDefaultDetail
	}

	return n
}
