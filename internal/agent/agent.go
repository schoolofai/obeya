package agent

import (
	"fmt"
	"sort"
	"strings"
)

// AgentContext carries metadata needed by agent setup.
type AgentContext struct {
	Root       string // project root directory
	BoardName  string // board name for summary output
	SkipPlugin bool   // --skip-plugin flag
	Shared     bool   // true when --shared + --agent are used together
}

// AgentSetup defines the interface for agent-specific initialization.
type AgentSetup interface {
	Name() string
	Setup(ctx AgentContext) error
}

var registry = map[string]AgentSetup{}

func register(a AgentSetup) {
	registry[a.Name()] = a
}

// Get returns the AgentSetup for the given name.
func Get(name string) (AgentSetup, error) {
	a, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unsupported agent %q. Supported: %s", name, strings.Join(SupportedNames(), ", "))
	}
	return a, nil
}

// SupportedNames returns sorted list of registered agent names.
func SupportedNames() []string {
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
