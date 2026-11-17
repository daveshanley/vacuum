// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// Includes adaptations of @openlint/ruleset-openapi 0.1.0.
// Copyright 2018 Stoplight, Inc. and 2026 OpenLint.
// See rulesets/packaged/OPENLINT-OPENAPI-LICENSE.
// SPDX-License-Identifier: MIT AND Apache-2.0

package openapi

import (
	"errors"
	"fmt"
	"math"
	"net/netip"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"
	"golang.org/x/net/idna"
)

var packagedServerVariable = regexp.MustCompile(`\{(.+?)\}`)

// The substitution check walks every combination of enum values. A handful of
// large enums would otherwise multiply into millions of URL parses, so only the
// first combinations, in enumeration order, are checked.
const maxServerVariableSubstitutions = 4096

func validPackagedServer(server packagedOperand) bool {
	if !server.field("url").isText() {
		return false
	}
	variables := server.field("variables")
	if variables.node == nil {
		return true
	}
	if !variables.is(yaml.MappingNode) {
		return false
	}
	for _, variable := range variables.fields() {
		if !variable.is(yaml.MappingNode) {
			return false
		}
		for key, field := range variable.fields() {
			switch key {
			case "enum":
				if !field.is(yaml.SequenceNode) {
					return false
				}
				for item := range field.items() {
					if !item.isText() {
						return false
					}
				}
			case "default", "description", "examples":
				if !field.isText() {
					return false
				}
			default:
				if !strings.HasPrefix(key, "x-") {
					return false
				}
			}
		}
	}
	return true
}

type urlVariable struct {
	placeholder string
	values      []string
}

func checkPackagedServerVariables(server packagedOperand, ctx model.RuleFunctionContext, results *packagedDiagnostics) {
	if !validPackagedServer(server) {
		results.add(server, "Invalid Server Object")
		return
	}
	template := server.field("url").text()
	variables := server.field("variables")
	found := make(map[string]bool)
	var missing []string
	for _, match := range packagedServerVariable.FindAllStringSubmatch(template, -1) {
		name := match[1]
		found[name] = true
		if variables.field(name).node == nil {
			missing = append(missing, name)
		}
	}
	var substitutions []urlVariable
	options := ctx.GetOptionsStringMap()
	for name, v := range variables.fields() {
		if !found[name] {
			results.add(v, fmt.Sprintf("Server's \"variables\" object has unused defined %q url variable.", name))
			continue
		}
		values := urlVariable{placeholder: "{" + name + "}"}
		def, enum := v.field("default"), v.field("enum")
		if enum.node != nil {
			for item := range enum.items() {
				values.values = append(values.values, encodeServerVariable(item.text()))
			}
			if def.node != nil && !enum.containsText(def.text()) {
				results.add(def, fmt.Sprintf("Server Variable %q has a default not listed in the enum.", name))
			}
		} else if def.node != nil {
			values.values = []string{encodeServerVariable(def.text())}
		}
		if def.node == nil && options["requireDefault"] == "true" {
			results.add(v, fmt.Sprintf("Server Variable %q has a missing default.", name))
		}
		substitutions = append(substitutions, values)
	}
	at := server.fieldOrSelf("variables")
	if len(missing) > 0 {
		results.add(at, "Not all server's variables are described with \"variables\" object. Missed: "+strings.Join(missing, ", ")+".")
	}
	if options["checkSubstitutions"] != "true" || len(substitutions) == 0 {
		return
	}
	for _, v := range substitutions {
		if len(v.values) == 0 {
			return
		}
	}
	counts := make([]int, len(substitutions))
	var invalid []string
	for checked := 0; ; checked++ {
		if checked == maxServerVariableSubstitutions {
			results.add(at, fmt.Sprintf("Server URL validation is incomplete after %d substitutions.", checked))
			break
		}
		if ctx.ExecutionContext != nil && ctx.ExecutionContext.Err() != nil {
			return
		}
		value := template
		for i, v := range substitutions {
			value = strings.ReplaceAll(value, v.placeholder, v.values[counts[i]])
		}
		if !validPackagedServerURL(value) {
			invalid = append(invalid, value)
			if len(invalid) == 5 {
				results.add(at, "At least 5 substitutions of server variables resulted in invalid URLs: "+strings.Join(invalid, ", ")+" and more")
				return
			}
		}
		i := len(counts) - 1
		for ; i >= 0; i-- {
			counts[i]++
			if counts[i] < len(substitutions[i].values) {
				break
			}
			counts[i] = 0
		}
		if i < 0 {
			break
		}
	}
	if len(invalid) > 0 {
		results.add(at, "A few substitutions of server variables resulted in invalid URLs: "+strings.Join(invalid, ", "))
	}
}

// OpenLint calls encodeURI on each substitution before checking the absolute URL.
func encodeServerVariable(s string) string {
	const hex = "0123456789ABCDEF"
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || strings.ContainsRune(";,/?:@&=+$-_.!~*'()#", rune(c)) {
			b.WriteByte(c)
			continue
		}
		b.WriteByte('%')
		b.WriteByte(hex[c>>4])
		b.WriteByte(hex[c&15])
	}
	return b.String()
}

// OpenLint checks each substituted URL with JavaScript's new URL(), which
// follows the WHATWG URL Standard. Only validity matters here, so this applies
// the standard's scheme, host and port rules without building a parsed URL.
// TestPackagedServerURLMatchesWHATWG pins the expected results to Node.
func validPackagedServerURL(raw string) bool {
	raw = strings.Trim(raw, "\x00\x01\x02\x03\x04\x05\x06\x07\x08\t\n\v\f\r\x0e\x0f\x10\x11\x12\x13\x14\x15\x16\x17\x18\x19\x1a\x1b\x1c\x1d\x1e\x1f ")
	raw = strings.NewReplacer("\t", "", "\n", "", "\r", "").Replace(raw)
	colon := strings.IndexByte(raw, ':')
	if colon < 1 || !validURLScheme(raw[:colon]) {
		return false
	}
	scheme := strings.ToLower(raw[:colon])
	switch scheme {
	case "http", "https", "ws", "wss", "ftp":
	case "file":
		// A file host may be empty or a Windows drive letter, and never has a port.
		rest := strings.ReplaceAll(raw[colon+1:], "\\", "/")
		if !strings.HasPrefix(rest, "//") {
			return true
		}
		host := rest[2:]
		if end := strings.IndexAny(host, "/?#"); end >= 0 {
			host = host[:end]
		}
		if host == "" || len(host) == 2 && validURLScheme(host[:1]) && (host[1] == ':' || host[1] == '|') {
			return true
		}
		return validSpecialURLHost(host)
	default:
		// Non-special URLs accept an opaque path, so only an authority
		// introduced by "//" is checked.
		rest := raw[colon+1:]
		if !strings.HasPrefix(rest, "//") {
			return true
		}
		authority := rest[2:]
		if end := strings.IndexAny(authority, "/?#"); end >= 0 {
			authority = authority[:end]
		}
		host, port, ok := splitURLAuthority(authority)
		if !ok || !validURLPort(port) {
			return false
		}
		// An empty host is valid only without credentials or a port.
		if host == "" {
			return authority == ""
		}
		return validOpaqueURLHost(host)
	}
	// Special URLs treat backslashes as slashes and ignore any number of
	// slashes before the authority, so "https:example.com" has a host.
	rest := strings.TrimLeft(strings.ReplaceAll(raw[colon+1:], "\\", "/"), "/")
	authority := rest
	if end := strings.IndexAny(authority, "/?#"); end >= 0 {
		authority = authority[:end]
	}
	host, port, ok := splitURLAuthority(authority)
	if !ok || host == "" || !validURLPort(port) {
		return false
	}
	return validSpecialURLHost(host)
}

func validURLScheme(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		alpha := c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
		if !alpha && (i == 0 || !(c >= '0' && c <= '9' || c == '+' || c == '-' || c == '.')) {
			return false
		}
	}
	return s != ""
}

// splitURLAuthority drops userinfo and separates the host from the port.
func splitURLAuthority(authority string) (host, port string, ok bool) {
	if at := strings.LastIndexByte(authority, '@'); at >= 0 {
		authority = authority[at+1:]
	}
	if strings.HasPrefix(authority, "[") {
		end := strings.IndexByte(authority, ']')
		if end < 0 {
			return "", "", false
		}
		host, rest := authority[:end+1], authority[end+1:]
		if rest == "" {
			return host, "", true
		}
		if !strings.HasPrefix(rest, ":") {
			return "", "", false
		}
		return host, rest[1:], true
	}
	if colon := strings.IndexByte(authority, ':'); colon >= 0 {
		return authority[:colon], authority[colon+1:], true
	}
	return authority, "", true
}

func validURLPort(port string) bool {
	if port == "" {
		return true
	}
	for i := 0; i < len(port); i++ {
		if port[i] < '0' || port[i] > '9' {
			return false
		}
	}
	n, err := strconv.ParseUint(port, 10, 64)
	return err == nil && n <= 65535
}

func validIPv6URLHost(host string) bool {
	addr, err := netip.ParseAddr(host[1 : len(host)-1])
	return err == nil && addr.Is6() && addr.Zone() == ""
}

func validOpaqueURLHost(host string) bool {
	if strings.HasPrefix(host, "[") {
		return strings.HasSuffix(host, "]") && validIPv6URLHost(host)
	}
	for i := 0; i < len(host); i++ {
		if c := host[i]; c != '%' && forbiddenURLHostByte(c) {
			return false
		}
	}
	return true
}

var urlHostProfile = idna.New(
	idna.MapForLookup(),
	idna.Transitional(false),
	idna.CheckHyphens(false),
	idna.CheckJoiners(true),
	idna.BidiRule(),
	idna.StrictDomainName(false),
	idna.VerifyDNSLength(false),
)

func validSpecialURLHost(host string) bool {
	if strings.HasPrefix(host, "[") {
		return strings.HasSuffix(host, "]") && validIPv6URLHost(host)
	}
	decoded, err := url.PathUnescape(host)
	if err != nil {
		return false
	}
	ascii, err := urlHostProfile.ToASCII(decoded)
	if err != nil || ascii == "" {
		return false
	}
	for i := 0; i < len(ascii); i++ {
		if c := ascii[i]; forbiddenURLHostByte(c) || c == '%' || c == 0x7f || c < 0x20 {
			return false
		}
	}
	if !urlHostEndsInNumber(ascii) {
		return true
	}
	return validURLIPv4(ascii)
}

func forbiddenURLHostByte(c byte) bool {
	switch c {
	case 0, '\t', '\n', '\r', ' ', '#', '/', ':', '<', '>', '?', '@', '[', '\\', ']', '^', '|':
		return true
	}
	return false
}

func urlIPv4Labels(host string) []string {
	labels := strings.Split(host, ".")
	if len(labels) > 1 && labels[len(labels)-1] == "" {
		labels = labels[:len(labels)-1]
	}
	return labels
}

// A host whose last label is a number must be a valid IPv4 address.
func urlHostEndsInNumber(host string) bool {
	labels := urlIPv4Labels(host)
	last := labels[len(labels)-1]
	if last == "" {
		return false
	}
	if strings.Trim(last, "0123456789") == "" {
		return true
	}
	_, ok := parseURLIPv4Number(last)
	return ok
}

func parseURLIPv4Number(s string) (uint64, bool) {
	if s == "" {
		return 0, false
	}
	base := 10
	switch {
	case len(s) >= 2 && (s[:2] == "0x" || s[:2] == "0X"):
		s, base = s[2:], 16
		if s == "" {
			return 0, true
		}
	case len(s) >= 2 && s[0] == '0':
		s, base = s[1:], 8
	}
	n, err := strconv.ParseUint(s, base, 64)
	if err != nil {
		// Digits beyond 64 bits still form a number, just an out-of-range one.
		var numErr *strconv.NumError
		if errors.As(err, &numErr) && numErr.Err == strconv.ErrRange {
			return math.MaxUint64, true
		}
		return 0, false
	}
	return n, true
}

func validURLIPv4(host string) bool {
	labels := urlIPv4Labels(host)
	if len(labels) > 4 {
		return false
	}
	for i, label := range labels {
		n, ok := parseURLIPv4Number(label)
		if !ok {
			return false
		}
		if i < len(labels)-1 {
			if n > 255 {
				return false
			}
			continue
		}
		return n < uint64(1)<<(8*(5-len(labels)))
	}
	return false
}
