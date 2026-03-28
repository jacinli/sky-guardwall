package parser

import (
	"strings"
	"time"

	"github.com/jacinli/sky-guardwall/internal/model"
)

// ParseOutput parses the full output of `iptables -S` and returns a slice of rules.
func ParseOutput(output string, syncedAt time.Time) []model.IptablesRule {
	var rules []model.IptablesRule
	for _, line := range strings.Split(output, "\n") {
		if r, ok := parseLine(line, syncedAt); ok {
			rules = append(rules, r)
		}
	}
	return rules
}

func parseLine(line string, syncedAt time.Time) (model.IptablesRule, bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return model.IptablesRule{}, false
	}

	rule := model.IptablesRule{RawLine: line, SyncedAt: syncedAt}
	tokens := strings.Fields(line)
	if len(tokens) < 2 {
		return model.IptablesRule{}, false
	}

	switch tokens[0] {
	case "-P":
		if len(tokens) >= 3 {
			rule.LineType = "policy"
			rule.Chain = tokens[1]
			rule.Target = tokens[2]
			return rule, true
		}
	case "-N":
		rule.LineType = "chain"
		rule.Chain = tokens[1]
		return rule, true
	case "-A":
		if len(tokens) < 3 {
			return model.IptablesRule{}, false
		}
		rule.LineType = "rule"
		rule.Chain = tokens[1]
		parseRuleFlags(tokens[2:], &rule)
		return rule, true
	}

	return model.IptablesRule{}, false
}

func parseRuleFlags(tokens []string, rule *model.IptablesRule) {
	var extras []string
	i := 0
	for i < len(tokens) {
		tok := tokens[i]
		next := func() string {
			i++
			if i < len(tokens) {
				return tokens[i]
			}
			return ""
		}

		switch tok {
		case "-j", "--jump":
			rule.Target = next()
		case "-p", "--protocol":
			rule.Protocol = next()
		case "-s", "--source", "--src":
			rule.Source = next()
		case "-d", "--destination", "--dst":
			rule.Dest = next()
		case "--dport", "--destination-port":
			rule.DstPort = next()
		case "--sport", "--source-port":
			rule.SrcPort = next()
		case "-i", "--in-interface":
			rule.InIface = next()
		case "-o", "--out-interface":
			rule.OutIface = next()
		case "!":
			// negation — keep with next token as extra
			if i+1 < len(tokens) {
				i++
				extras = append(extras, "!", tokens[i])
			}
		default:
			extras = append(extras, tok)
		}
		i++
	}

	if len(extras) > 0 {
		rule.Extra = strings.Join(extras, " ")
	}
}
