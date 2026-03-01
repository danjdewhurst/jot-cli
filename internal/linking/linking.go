package linking

import "regexp"

// refPattern matches @<alnum>{4,} references that are not preceded by another
// alphanumeric character (to avoid matching email addresses).
var refPattern = regexp.MustCompile(`(?:^|[^a-zA-Z0-9])@([a-zA-Z0-9]{4,})`)

// ExtractRefs scans a note body for @<prefix> references and returns
// the unique prefixes found, in order of first appearance.
func ExtractRefs(body string) []string {
	matches := refPattern.FindAllStringSubmatch(body, -1)
	seen := make(map[string]struct{})
	var refs []string
	for _, m := range matches {
		prefix := m[1]
		if _, ok := seen[prefix]; ok {
			continue
		}
		seen[prefix] = struct{}{}
		refs = append(refs, prefix)
	}
	return refs
}
