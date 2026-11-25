package cli

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

// ExtractIDs decodes a list-style API response and returns, for each item, the
// value of the first field in `fields` that is present and a string. It locates
// the item slice whether resp is a top-level array or an object wrapping the
// slice under any key (items, listData, volumeTypes, ...). Returns nil when no
// slice/fields are found.
func ExtractIDs(resp interface{}, fields ...string) []string {
	items := findSlice(resp)
	if items == nil {
		return nil
	}
	var out []string
	seen := map[string]bool{}
	for _, it := range items {
		m, ok := it.(map[string]interface{})
		if !ok {
			continue
		}
		for _, f := range fields {
			if v, ok := m[f].(string); ok && v != "" {
				if !seen[v] {
					seen[v] = true
					out = append(out, v)
				}
				break
			}
		}
	}
	return out
}

// knownListKeys are the wrapper keys VKS/vserver use for their list payloads,
// checked in order before falling back to a generic scan. Checking by name first
// avoids non-deterministic results when a response has more than one array field.
var knownListKeys = []string{"items", "listData", "data", "volumeTypes"}

func findSlice(resp interface{}) []interface{} {
	switch v := resp.(type) {
	case []interface{}:
		return v
	case map[string]interface{}:
		for _, key := range knownListKeys {
			if s, ok := v[key].([]interface{}); ok {
				return s
			}
		}
		for _, val := range v {
			if s, ok := val.([]interface{}); ok {
				return s
			}
		}
	}
	return nil
}

// ExtractCompletions pulls "id\tname" completion pairs from a list response.
// listKeys are the envelope keys to try in order (e.g. "listData", "data");
// idKey is the completion value and nameKey the tab description. Items missing
// idKey are skipped; a missing nameKey yields a bare id.
func ExtractCompletions(result interface{}, listKeys []string, idKey, nameKey string) []string {
	var items []interface{}
	switch v := result.(type) {
	case []interface{}:
		items = v
	case map[string]interface{}:
		for _, key := range listKeys {
			if d, ok := v[key].([]interface{}); ok {
				items = d
				break
			}
		}
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		obj, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		id, _ := obj[idKey].(string)
		if id == "" {
			continue
		}
		if name, _ := obj[nameKey].(string); name != "" {
			out = append(out, id+"\t"+name)
		} else {
			out = append(out, id)
		}
	}
	return out
}

// CompFunc is a cobra flag/positional value completion function.
type CompFunc = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective)

// completionTimeout bounds API-backed completion so a slow backend never hangs
// the shell. Overridable in tests.
var completionTimeout = 2 * time.Second

func filterPrefix(values []string, toComplete string) []string {
	if toComplete == "" {
		return values
	}
	var out []string
	for _, v := range values {
		if strings.HasPrefix(v, toComplete) {
			out = append(out, v)
		}
	}
	return out
}

// FlagValues completes from a fixed list (enum), prefix-filtered.
func FlagValues(values ...string) CompFunc {
	return func(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return filterPrefix(values, toComplete), cobra.ShellCompDirectiveNoFileComp
	}
}

// FlagValuesFrom completes from a list computed at completion time (e.g. config).
func FlagValuesFrom(fn func() []string) CompFunc {
	return func(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return filterPrefix(fn(), toComplete), cobra.ShellCompDirectiveNoFileComp
	}
}

// FlagFromAPI completes from a backend list with a bounded timeout; on any error
// or timeout it yields no suggestions (never breaks the shell). Prefix-filtered.
func FlagFromAPI(fetch func(ctx context.Context, cmd *cobra.Command) ([]string, error)) CompFunc {
	return func(cmd *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		ctx, cancel := context.WithTimeout(context.Background(), completionTimeout)
		defer cancel()
		type result struct {
			vals []string
			err  error
		}
		ch := make(chan result, 1)
		go func() {
			v, e := fetch(ctx, cmd)
			ch <- result{v, e}
		}()
		select {
		case <-ctx.Done():
			return nil, cobra.ShellCompDirectiveNoFileComp
		case r := <-ch:
			if r.err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return filterPrefix(r.vals, toComplete), cobra.ShellCompDirectiveNoFileComp
		}
	}
}

var (
	resourceMu         sync.RWMutex
	resourceCompleters = map[string]CompFunc{}
)

// RegisterResourceCompleter registers a completer under a stable key (e.g.
// "vserver:subnet"). Providers call this from init(). Concurrency-safe.
func RegisterResourceCompleter(key string, fn CompFunc) {
	resourceMu.Lock()
	defer resourceMu.Unlock()
	resourceCompleters[key] = fn
}

// ResourceCompletion returns a completer that dispatches to the provider
// registered under key at completion time. Unregistered key -> safe no-op.
func ResourceCompletion(key string) CompFunc {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		resourceMu.RLock()
		fn := resourceCompleters[key]
		resourceMu.RUnlock()
		if fn == nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return fn(cmd, args, toComplete)
	}
}
