package sprig

import "regexp"

func mustCompile(pattern string) *regexp.Regexp {
	r, err := regexp.Compile(pattern)
	if err != nil {
		panic(errInvalidRegexp)
	}
	return r
}

func regexMatch(regex string, s string) bool {
	match, _ := regexp.MatchString(regex, s)
	return match
}

func mustRegexMatch(regex string, s string) (bool, error) {
	match, err := regexp.MatchString(regex, s)
	if err != nil {
		return false, errInvalidRegexp
	}
	return match, nil
}
func regexFindAll(regex, s string, n int) []string { return mustCompile(regex).FindAllString(s, n) }
func mustRegexFindAll(regex, s string, n int) ([]string, error) {
	r, err := regexp.Compile(regex)
	if err != nil {
		return []string{}, errInvalidRegexp
	}
	return r.FindAllString(s, n), nil
}
func regexFind(regex, s string) string { return mustCompile(regex).FindString(s) }
func mustRegexFind(regex, s string) (string, error) {
	r, err := regexp.Compile(regex)
	if err != nil {
		return "", errInvalidRegexp
	}
	return r.FindString(s), nil
}
func regexReplaceAll(regex, s, repl string) string {
	return mustCompile(regex).ReplaceAllString(s, repl)
}
func mustRegexReplaceAll(regex, s, repl string) (string, error) {
	r, err := regexp.Compile(regex)
	if err != nil {
		return "", errInvalidRegexp
	}
	return r.ReplaceAllString(s, repl), nil
}
func regexReplaceAllLiteral(regex, s, repl string) string {
	return mustCompile(regex).ReplaceAllLiteralString(s, repl)
}
func mustRegexReplaceAllLiteral(regex, s, repl string) (string, error) {
	r, err := regexp.Compile(regex)
	if err != nil {
		return "", errInvalidRegexp
	}
	return r.ReplaceAllLiteralString(s, repl), nil
}
func regexSplit(regex, s string, n int) []string { return mustCompile(regex).Split(s, n) }
func mustRegexSplit(regex, s string, n int) ([]string, error) {
	r, err := regexp.Compile(regex)
	if err != nil {
		return []string{}, errInvalidRegexp
	}
	return r.Split(s, n), nil
}
func regexQuoteMeta(s string) string { return regexp.QuoteMeta(s) }
