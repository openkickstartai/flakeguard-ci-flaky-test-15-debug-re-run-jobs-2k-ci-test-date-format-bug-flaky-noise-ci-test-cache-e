package main

import (
	"encoding/xml"
	"math"
	"os"
	"strings"
)

type xmlSuites struct {
	XMLName xml.Name   `xml:"testsuites"`
	Suites  []xmlSuite `xml:"testsuite"`
}
type xmlSuite struct {
	XMLName xml.Name   `xml:"testsuite"`
	Cases   []TestCase `xml:"testcase"`
}
type TestCase struct {
	Name    string      `xml:"name,attr"`
	Class   string      `xml:"classname,attr"`
	Time    float64     `xml:"time,attr"`
	Failure *xmlFailure `xml:"failure"`
}
type xmlFailure struct {
	Message string `xml:"message,attr"`
	Body    string `xml:",chardata"`
}
type TestRecord struct {
	Name, Class string
	Pass, Fail  int
	TotalTime   float64
	Errors      []string
}
type FlakyTest struct {
	Name       string  `json:"name"`
	Class      string  `json:"class"`
	PassRate   float64 `json:"pass_rate"`
	FlakyProb  float64 `json:"flaky_probability"`
	RootCause  string  `json:"root_cause"`
	Runs       int     `json:"total_runs"`
	AvgTime    float64 `json:"avg_duration_s"`
	CICostUSD  float64 `json:"ci_cost_usd"`
	Suggestion string  `json:"fix_suggestion"`
}

func ParseJUnitXML(path string) ([]TestCase, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var suites xmlSuites
	if xml.Unmarshal(data, &suites) == nil && len(suites.Suites) > 0 {
		var out []TestCase
		for _, s := range suites.Suites {
			out = append(out, s.Cases...)
		}
		return out, nil
	}
	var suite xmlSuite
	if err := xml.Unmarshal(data, &suite); err != nil {
		return nil, err
	}
	return suite.Cases, nil
}

func Aggregate(runs [][]TestCase) map[string]*TestRecord {
	m := map[string]*TestRecord{}
	for _, cases := range runs {
		for _, tc := range cases {
			key := tc.Class + "::" + tc.Name
			r := m[key]
			if r == nil {
				r = &TestRecord{Name: tc.Name, Class: tc.Class}
				m[key] = r
			}
			if tc.Failure != nil {
				r.Fail++
				r.Errors = append(r.Errors, tc.Failure.Message+" "+tc.Failure.Body)
			} else {
				r.Pass++
			}
			r.TotalTime += tc.Time
		}
	}
	return m
}

func BetaFlaky(pass, fail int) float64 {
	if pass+fail < 2 || fail == 0 || pass == 0 {
		return 0
	}
	a, b := float64(pass+1), float64(fail+1)
	mu := a / (a + b)
	sd := math.Sqrt(a * b / ((a + b) * (a + b) * (a + b + 1)))
	if sd < 1e-10 {
		return 0
	}
	phi := func(x float64) float64 { return 0.5 * (1 + math.Erf(x/math.Sqrt2)) }
	return phi((0.95-mu)/sd) - phi((0.05-mu)/sd)
}

var rootCauses = []struct{ tag, kws, fix string }{
	{"race-condition", "race,concurrent,mutex,lock,deadlock,thread", "Add synchronization; isolate shared state per test"},
	{"timing-dependency", "timeout,deadline,sleep,elapsed,timed out", "Replace sleeps with polling/retry; shorten timeouts"},
	{"timezone", "timezone,utc,datetime,date_format,localtime", "Pin TZ=UTC; use time-freezing libraries"},
	{"network", "connection,network,econnrefused,dns,socket,refused", "Mock external calls; use httptest/wiremock"},
	{"floating-point", "precision,float,decimal,epsilon,almost", "Use approximate assertions with epsilon tolerance"},
	{"test-ordering", "order,sequence,setup,teardown,depends", "Ensure independent setup/teardown per test"},
	{"shared-state", "state,cache,global,singleton,pollution", "Reset shared state in setUp; use fresh fixtures"},
}

func Classify(errors []string) (string, string) {
	text := strings.ToLower(strings.Join(errors, " "))
	for _, c := range rootCauses {
		for _, kw := range strings.Split(c.kws, ",") {
			if strings.Contains(text, kw) {
				return c.tag, c.fix
			}
		}
	}
	return "unknown", "Run tests in random order; check for hidden shared state"
}

func Detect(records map[string]*TestRecord, costPerMin, threshold float64) []FlakyTest {
	var out []FlakyTest
	for _, r := range records {
		prob := BetaFlaky(r.Pass, r.Fail)
		if prob < threshold {
			continue
		}
		n := r.Pass + r.Fail
		cause, fix := Classify(r.Errors)
		avg := r.TotalTime / float64(n)
		cost := float64(r.Fail) * avg / 60.0 * costPerMin
		out = append(out, FlakyTest{
			Name: r.Name, Class: r.Class,
			PassRate:  math.Round(float64(r.Pass)/float64(n)*1000) / 1000,
			FlakyProb: math.Round(prob*1000) / 1000, RootCause: cause,
			Runs: n, AvgTime: math.Round(avg*100) / 100,
			CICostUSD: math.Round(cost*100) / 100, Suggestion: fix,
		})
	}
	return out
}
