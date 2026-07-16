package report

import (
	"fmt"
	"strings"
)

// Endpoint is a shields.io endpoint-badge document. SchemaVersion is shields'
// own contract (always 1), unrelated to the report Document's SchemaVersion.
// bigo writes this JSON; a repo commits or publishes it and points shields.io
// at it as a static endpoint (https://shields.io/badges/endpoint-badge).
type Endpoint struct {
	SchemaVersion int    `json:"schemaVersion"`
	Label         string `json:"label"`
	Message       string `json:"message"`
	Color         string `json:"color"`
}

// Badge projects a report Document into the budget badge — the honest
// anti-rating (spec §6). It claims only that the functions this module chose to
// budget are verified within those budgets; unverifiable and invalid budgets
// are counted in the message, never hidden. Every declared budget counts once:
// a function with both a //bigo:max and a //bigo:space contributes two. Verdicts
// are read verbatim off the document — the badge reinterprets nothing.
func Badge(doc Document) Endpoint {
	var within, exceeds, unverifiable, invalid int
	tally := func(v string) {
		switch v {
		case "within":
			within++
		case "exceeds":
			exceeds++
		case "invalid":
			invalid++
		default: // "unverifiable" and any verdict the badge does not model → not verified
			unverifiable++
		}
	}
	for _, f := range doc.Functions {
		if f.Budget != nil {
			tally(f.Budget.Verdict)
		}
		if f.Space != nil && f.Space.Budget != nil {
			tally(f.Space.Budget.Verdict)
		}
	}
	total := within + exceeds + unverifiable + invalid

	ep := Endpoint{SchemaVersion: 1, Label: "bigo"}
	if total == 0 {
		ep.Message = "no budgets"
		ep.Color = "lightgrey"
		return ep
	}
	noun := "budgets"
	if total == 1 {
		noun = "budget"
	}
	if exceeds == 0 && invalid == 0 && unverifiable == 0 {
		ep.Message = fmt.Sprintf("%d %s · all within", total, noun)
		ep.Color = "brightgreen"
		return ep
	}
	// Non-within states, listed and colored in severity order:
	// exceeds (proven violation) > invalid (broken annotation) > unverifiable.
	var parts []string
	if exceeds > 0 {
		parts = append(parts, fmt.Sprintf("%d exceeds", exceeds))
	}
	if invalid > 0 {
		parts = append(parts, fmt.Sprintf("%d invalid", invalid))
	}
	if unverifiable > 0 {
		parts = append(parts, fmt.Sprintf("%d unverifiable", unverifiable))
	}
	ep.Message = fmt.Sprintf("%d %s · %s", total, noun, strings.Join(parts, ", "))
	switch {
	case exceeds > 0:
		ep.Color = "red"
	case invalid > 0:
		ep.Color = "orange"
	default:
		ep.Color = "yellow"
	}
	return ep
}
