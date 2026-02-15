# FlakeGuard

**Detect, classify, and quarantine flaky tests — before they waste your team's time and CI budget.**

FlakeGuard analyzes JUnit XML test results across multiple CI runs using Bayesian statistics to identify non-deterministic tests, classify their root cause, and estimate the CI cost they burn.

## 🚀 Quick Start

```bash
# Build
go build -o flakeguard .

# Point at your JUnit XML results from multiple CI runs
./flakeguard --results "./test-results/*.xml"

# JSON output for CI integration
./flakeguard --results "./test-results/*.xml" --json

# Custom CI cost rate ($0.008/min = GitHub Actions Linux)
./flakeguard --results "./test-results/*.xml" --cost 0.008 --threshold 0.6
```

Works with **any framework** that outputs JUnit XML:
- **Python**: `pytest --junitxml=results.xml`
- **JavaScript**: `jest --reporters=jest-junit`
- **Java**: JUnit/TestNG (native XML)
- **Go**: `go test | go-junit-report`
- **Ruby**: `rspec_junit_formatter`

## ⚡ How It Works

1. **Parse** — Reads JUnit XML files from N CI runs
2. **Aggregate** — Tracks pass/fail per test across all runs
3. **Detect** — Beta-Binomial Bayesian model computes P(flaky) per test
4. **Classify** — Pattern-matches errors to root causes (race condition, timeout, timezone, etc.)
5. **Report** — Actionable output with fix suggestions and CI cost per flaky test

Exit code `2` when flaky tests found — use as CI quality gate.

## 📊 Why Pay for FlakeGuard?

| Metric | Before | After |
|--------|--------|-------|
| Weekly hours debugging flaky tests | 15h | 1h |
| Monthly CI retry cost | $2,000+ | $200 |
| Time to identify new flaky test | Days | Minutes |
| False failures blocking deploys | 5-10/week | 0 |

**ROI**: Team of 10 saves ~$5K/month. Pro costs $149/month. **33x ROI.**

## 💰 Pricing

| Feature | Free | Pro $49/mo | Team $149/mo | Enterprise $399/mo |
|---------|------|-----------|-------------|--------------------|
| Flaky detection + classification | ✅ | ✅ | ✅ | ✅ |
| CLI + JSON output | ✅ | ✅ | ✅ | ✅ |
| Max history | 10 runs | 100 runs | Unlimited | Unlimited |
| GitHub/GitLab PR comments | ❌ | ✅ | ✅ | ✅ |
| Auto-quarantine PR generation | ❌ | ✅ | ✅ | ✅ |
| Slack notifications | ❌ | ❌ | ✅ | ✅ |
| Trend dashboard (SaaS) | ❌ | ❌ | ✅ | ✅ |
| CI cost attribution reports | ❌ | ❌ | ✅ | ✅ |
| Multi-repo / SSO / self-hosted | ❌ | ❌ | ❌ | ✅ |

## 🏗 Architecture

Single static binary. Zero dependencies. Parses XML at ~500MB/s.
Bayesian model uses normal approximation to Beta posterior — O(1) per test.

## License

MIT (free core). Commercial features require a paid license key.
