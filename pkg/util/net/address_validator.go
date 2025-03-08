package net

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const (
	addressSplitChar = ":"
)

// Analyze it yourself here, in order to support the diversity of configurations
func SplitIPPortStr(addr string) (string, int, error) {
	if addr == "" {
		return "", 0, fmt.Errorf("split ip err: param addr must not empty")
	}

	if addr[0] == '[' {
		reg := regexp.MustCompile("[\\[\\]]")
		addr = reg.ReplaceAllString(addr, "")
	}

	i := strings.LastIndex(addr, addressSplitChar)
	if i < 0 {
		return "", 0, fmt.Errorf("address %s: missing port in address", addr)
	}

	host := addr[:i]
	port := addr[i+1:]

	if strings.Contains(host, "%") {
		reg := regexp.MustCompile("\\%[0-9]+")
		host = reg.ReplaceAllString(host, "")
	}

	portInt, err := strconv.Atoi(port)
	if err != nil {
		return "", 0, err
	}
	return host, portInt, nil
}
