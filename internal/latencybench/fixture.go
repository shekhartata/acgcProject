package latencybench

import (
	"encoding/json"
	"fmt"
	"os"
)

// Fixture describes warm-up transcript pairs plus the timed probe question.
type Fixture struct {
	System    string         `json:"system"`
	WarmPairs []WarmPairTurn `json:"warm_pairs"`
	Probe     string         `json:"probe"`
}

type WarmPairTurn struct {
	User      string `json:"user"`
	Assistant string `json:"assistant"`
}

// DefaultFixtureJSON is bundled when --fixture is not set.
var DefaultFixtureJSON = []byte(`{
  "system": "You are a helpful technical assistant. Answer concisely.",
  "warm_pairs": [
    {
      "user": "Briefly summarize what Go modules accomplish.",
      "assistant": "Go modules add versioned dependency management: a go.mod file pins module paths and semver versions while the checksum database helps reproducible builds."
    },
    {
      "user": "When would you still need GOPATH?",
      "assistant": "Legacy workflows and some tooling that predate modules might expect GOPATH, but typical projects now ignore GOPATH aside from caches."
    }
  ],
  "probe": "Give one downside of pinning many replace directives in go.mod."
}`)

func LoadFixture(path string) (*Fixture, error) {
	raw := DefaultFixtureJSON
	if path != "" {
		var err error
		raw, err = os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read fixture: %w", err)
		}
	}
	var f Fixture
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("parse fixture JSON: %w", err)
	}
	if f.Probe == "" {
		return nil, fmt.Errorf("fixture.probe is required")
	}
	return &f, nil
}
