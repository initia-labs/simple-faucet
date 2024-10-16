package main

import (
	"fmt"
	"regexp"
)

func parseRegexp(regexpStr string, target string) (data string, err error) {
	// Capture seqeunce string from json
	r := regexp.MustCompile(regexpStr)
	groups := r.FindStringSubmatch(string(target))

	if len(groups) != 2 {
		return data, fmt.Errorf("failed to parse regexp - expstr: %s, target: %s", regexpStr, target)
	}

	// Convert sequence string to int64
	data = groups[1]
	return
}
