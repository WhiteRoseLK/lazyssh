// Copyright 2025.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package services

import (
	"strconv"
	"strings"

	"github.com/WhiteRoseLK/neossh/internal/core/domain"
)

type serverFilter struct {
	tagPatterns       []string
	negTagPatterns    []string
	userPatterns      []string
	negUserPatterns   []string
	hostPatterns      []string
	negHostPatterns   []string
	portPatterns      []int
	negPortPatterns   []int
	groupPatterns     []string
	negGroupPatterns  []string
	statusPatterns    []string
	negStatusPatterns []string
	freeTerms         []string
}

func parseSearchQuery(query string) *serverFilter {
	filter := &serverFilter{}
	tokens := tokenizeQuery(query)

	for _, token := range tokens {
		if token == "" {
			continue
		}

		lower := strings.ToLower(token)
		isNeg := strings.HasPrefix(lower, "-")
		content := token
		contentLower := lower
		if isNeg {
			content = token[1:]
			contentLower = lower[1:]
		}

		colonIdx := strings.Index(contentLower, ":")
		if colonIdx <= 0 {
			filter.freeTerms = append(filter.freeTerms, token)
			continue
		}

		key := contentLower[:colonIdx]
		val := content[colonIdx+1:]
		valLower := strings.ToLower(val)

		if val == "" {
			filter.freeTerms = append(filter.freeTerms, token)
			continue
		}

		switch key {
		case "tag", "tags":
			if isNeg {
				filter.negTagPatterns = append(filter.negTagPatterns, valLower)
			} else {
				filter.tagPatterns = append(filter.tagPatterns, valLower)
			}
		case "user":
			if isNeg {
				filter.negUserPatterns = append(filter.negUserPatterns, valLower)
			} else {
				filter.userPatterns = append(filter.userPatterns, valLower)
			}
		case "host":
			if isNeg {
				filter.negHostPatterns = append(filter.negHostPatterns, valLower)
			} else {
				filter.hostPatterns = append(filter.hostPatterns, valLower)
			}
		case "port":
			if p, err := strconv.Atoi(val); err == nil && p > 0 {
				if isNeg {
					filter.negPortPatterns = append(filter.negPortPatterns, p)
				} else {
					filter.portPatterns = append(filter.portPatterns, p)
				}
			} else {
				filter.freeTerms = append(filter.freeTerms, token)
			}
		case "status":
			normalized := normalizeStatus(valLower)
			if normalized != "" {
				if isNeg {
					filter.negStatusPatterns = append(filter.negStatusPatterns, normalized)
				} else {
					filter.statusPatterns = append(filter.statusPatterns, normalized)
				}
			} else {
				filter.freeTerms = append(filter.freeTerms, token)
			}
		case "group":
			if isNeg {
				filter.negGroupPatterns = append(filter.negGroupPatterns, valLower)
			} else {
				filter.groupPatterns = append(filter.groupPatterns, valLower)
			}
		default:
			filter.freeTerms = append(filter.freeTerms, token)
		}
	}

	return filter
}

func normalizeStatus(val string) string {
	switch val {
	case "up", "online", "ok", "success":
		return "up"
	case "down", "offline", "fail", "failed":
		return "down"
	case "unknown", "checking", "pending":
		return "unknown"
	default:
		return ""
	}
}

func tokenizeQuery(raw string) []string {
	var tokens []string
	var current strings.Builder
	inQuote := false
	var quoteChar rune

	for _, r := range raw {
		switch {
		case !inQuote && (r == '"' || r == '\''):
			inQuote = true
			quoteChar = r
		case inQuote && r == quoteChar:
			inQuote = false
			quoteChar = 0
		case !inQuote && (r == ' ' || r == '\t'):
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

func (f *serverFilter) matchesNegative(srv domain.Server) bool {
	for _, negTag := range f.negTagPatterns {
		for _, tag := range srv.Tags {
			if strings.Contains(strings.ToLower(tag), negTag) {
				return false
			}
		}
	}

	for _, negUser := range f.negUserPatterns {
		if strings.Contains(strings.ToLower(srv.User), negUser) {
			return false
		}
	}

	for _, negHost := range f.negHostPatterns {
		if strings.Contains(strings.ToLower(srv.Host), negHost) {
			return false
		}
	}

	srvPort := srv.Port
	if srvPort == 0 {
		srvPort = 22
	}
	for _, negPort := range f.negPortPatterns {
		if srvPort == negPort {
			return false
		}
	}

	for _, negGroup := range f.negGroupPatterns {
		if strings.Contains(strings.ToLower(srv.Group), negGroup) {
			return false
		}
	}

	srvStatus := strings.ToLower(srv.PingStatus)
	if srvStatus != "up" && srvStatus != "down" {
		srvStatus = "unknown"
	}
	for _, negStatus := range f.negStatusPatterns {
		if srvStatus == negStatus {
			return false
		}
	}

	return true
}

func (f *serverFilter) matchesPositive(srv domain.Server) bool {
	srvPort := srv.Port
	if srvPort == 0 {
		srvPort = 22
	}
	srvStatus := strings.ToLower(srv.PingStatus)
	if srvStatus != "up" && srvStatus != "down" {
		srvStatus = "unknown"
	}

	for _, reqTag := range f.tagPatterns {
		matched := false
		for _, tag := range srv.Tags {
			if strings.Contains(strings.ToLower(tag), reqTag) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	for _, reqUser := range f.userPatterns {
		if !strings.Contains(strings.ToLower(srv.User), reqUser) {
			return false
		}
	}

	for _, reqHost := range f.hostPatterns {
		if !strings.Contains(strings.ToLower(srv.Host), reqHost) {
			return false
		}
	}

	for _, reqPort := range f.portPatterns {
		if srvPort != reqPort {
			return false
		}
	}

	for _, reqGroup := range f.groupPatterns {
		if !strings.Contains(strings.ToLower(srv.Group), reqGroup) {
			return false
		}
	}

	for _, reqStatus := range f.statusPatterns {
		if srvStatus != reqStatus {
			return false
		}
	}

	return true
}

func (f *serverFilter) Matches(srv domain.Server) bool {
	if !f.matchesNegative(srv) {
		return false
	}
	if !f.matchesPositive(srv) {
		return false
	}

	for _, term := range f.freeTerms {
		if computeServerScoreForTerm(srv, term) == 0 {
			return false
		}
	}

	return true
}

func (f *serverFilter) Score(srv domain.Server) int {
	if !f.Matches(srv) {
		return 0
	}

	if len(f.freeTerms) == 0 {
		return 100
	}

	total := 0
	for _, term := range f.freeTerms {
		sc := computeServerScoreForTerm(srv, term)
		if sc == 0 {
			return 0
		}
		total += sc
	}

	return total
}

func computeServerScoreForTerm(srv domain.Server, term string) int {
	best := 0
	fields := []string{
		srv.Alias,
		srv.Host,
		srv.User,
	}
	if len(srv.Aliases) > 0 {
		fields = append(fields, strings.Join(srv.Aliases, " "))
		for _, a := range srv.Aliases {
			if a != "" {
				fields = append(fields, a)
			}
		}
	}
	if len(srv.Tags) > 0 {
		fields = append(fields, strings.Join(srv.Tags, " "))
	}
	if srv.Group != "" {
		fields = append(fields, srv.Group)
	}

	for _, f := range fields {
		if f == "" {
			continue
		}
		if sc := fuzzyScore(term, f); sc > best {
			best = sc
		}
	}
	return best
}
