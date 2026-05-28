package privacy

import "regexp"

var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(authorization\s*[:=]\s*bearer\s+)[^\s"']+`),
	regexp.MustCompile(`(?i)((api[_-]?key|token|password|secret)\s*[:=]\s*)[^\s"',]+`),
	regexp.MustCompile(`sk-[A-Za-z0-9_\-]{12,}`),
	regexp.MustCompile(`gh[pousr]_[A-Za-z0-9_]{12,}`),
	regexp.MustCompile(`prtbl_[A-Za-z0-9_\-]{12,}`),
}

func Sanitize(input string) string {
	out := input
	for _, pattern := range secretPatterns {
		out = pattern.ReplaceAllStringFunc(out, func(match string) string {
			submatches := pattern.FindStringSubmatch(match)
			if len(submatches) > 1 && submatches[1] != "" {
				return submatches[1] + "[REDACTED]"
			}
			return "[REDACTED]"
		})
	}
	return out
}
