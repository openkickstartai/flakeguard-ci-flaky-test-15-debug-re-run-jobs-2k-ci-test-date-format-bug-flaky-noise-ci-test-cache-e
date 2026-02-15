package main

import (
	"os"
	"path/filepath"
	"testing"
)

const xmlRun1 = `<?xml version="1.0"?>
<testsuite name="s" tests="3">
  <testcase classname="auth" name="test_login" time="0.5"/>
  <testcase classname="auth" name="test_token" time="1.2">
    <failure message="timeout waiting">connection timed out</failure>
  </testcase>
  <testcase classname="core" name="test_calc" time="0.1"/>
</testsuite>`

const xmlRun2 = `<?xml version="1.0"?>
<testsuite name="s" tests="3">
  <testcase classname="auth" name="test_login" time="0.4"/>
  <testcase classname="auth" name="test_token" time="0.8"/>
  <testcase classname="core" name="test_calc" time="0.12">
    <failure message="race detected">concurrent map writes</failure>
  </testcase>
</testsuite>`

const xmlRun3 = `<?xml version="1.0"?>
<testsuite name="s" tests="3">
  <testcase classname="auth" name="test_login" time="0.5"/>
  <testcase classname="auth" name="test_token" time="1.0">
    <failure message="deadline exceeded">timeout</failure>
  </testcase>
  <testcase classname="core" name="test_calc" time="0.11"/>
</testsuite>`

func setupRuns(t *testing.T) [][]TestCase {
	t.Helper()
	dir := t.TempDir()
	var runs [][]TestCase
	for i, data := range []string{xmlRun1, xmlRun2, xmlRun3} {
		p := filepath.Join(dir, fmt.Sprintf("run%d.xml", i))
		os.WriteFile(p, []byte(data), 0644)
		cases, err := ParseJUnitXML(p)
		if err != nil {
			t.Fatalf("parse run%d: %v", i, err)
		}
		runs = append(runs, cases)
	}
	return runs
}

func TestParseJUnitXML(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "r.xml")
	os.WriteFile(p, []byte(xmlRun1), 0644)
	cases, err := ParseJUnitXML(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 3 {
		t.Fatalf("want 3 cases, got %d", len(cases))
	}
	if cases[0].Name != "test_login" {
		t.Errorf("want test_login, got %s", cases[0].Name)
	}
	if cases[1].Failure == nil {
		t.Error("test_token should have failure in run1")
	}
	if cases[2].Failure != nil {
		t.Error("test_calc should pass in run1")
	}
}

func TestDetectFlakyTests(t *testing.T) {
	runs := setupRuns(t)
	records := Aggregate(runs)
	if len(records) != 3 {
		t.Fatalf("want 3 records, got %d", len(records))
	}
	flaky := Detect(records, 0.008, 0.5)
	if len(flaky) < 2 {
		t.Fatalf("want >= 2 flaky tests, got %d", len(flaky))
	}
	found := map[string]bool{}
	for _, f := range flaky {
		found[f.Name] = true
		if f.PassRate <= 0 || f.PassRate >= 1 {
			t.Errorf("%s: pass rate %.3f out of (0,1)", f.Name, f.PassRate)
		}
		if f.CICostUSD < 0 {
			t.Errorf("%s: negative cost", f.Name)
		}
		if f.Suggestion == "" {
			t.Errorf("%s: empty suggestion", f.Name)
		}
	}
	if !found["test_token"] {
		t.Error("test_token should be flaky (1 pass / 2 fail)")
	}
	if !found["test_calc"] {
		t.Error("test_calc should be flaky (2 pass / 1 fail)")
	}
}

func TestRootCauseClassification(t *testing.T) {
	cases := []struct {
		errs   []string
		want   string
	}{
		{[]string{"timeout waiting", "deadline exceeded"}, "timing-dependency"},
		{[]string{"race detected", "concurrent map writes"}, "race-condition"},
		{[]string{"connection refused"}, "network"},
		{[]string{"timezone UTC mismatch"}, "timezone"},
		{[]string{"float precision loss"}, "floating-point"},
		{[]string{"random unknown error"}, "unknown"},
	}
	for _, tc := range cases {
		cause, fix := Classify(tc.errs)
		if cause != tc.want {
			t.Errorf("Classify(%v) = %q, want %q", tc.errs, cause, tc.want)
		}
		if fix == "" {
			t.Errorf("Classify(%v): empty fix", tc.errs)
		}
	}
}

func TestBetaFlakyMath(t *testing.T) {
	if p := BetaFlaky(10, 0); p != 0 {
		t.Errorf("all-pass: want 0, got %f", p)
	}
	if p := BetaFlaky(0, 10); p != 0 {
		t.Errorf("all-fail: want 0, got %f", p)
	}
	if p := BetaFlaky(1, 0); p != 0 {
		t.Errorf("single-pass: want 0, got %f", p)
	}
	p := BetaFlaky(5, 5)
	if p < 0.9 {
		t.Errorf("50/50 should be highly flaky, got %f", p)
	}
	p2 := BetaFlaky(1, 2)
	if p2 < 0.5 {
		t.Errorf("1pass/2fail should be flaky, got %f", p2)
	}
}
