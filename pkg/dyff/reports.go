package dyff

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gonvenience/ytbx"
)

func (r Report) filter(hasPath func(*ytbx.Path) bool) (result Report) {
	result = Report{
		From: r.From,
		To:   r.To,
	}

	for _, diff := range r.Diffs {
		if hasPath(diff.Path) {
			result.Diffs = append(result.Diffs, diff)
		}
	}

	return result
}

// Filter accepts YAML paths as input and returns a new report with differences for those paths only
func (r Report) Filter(paths ...string) (result Report) {
	if len(paths) == 0 {
		return r
	}

	return r.filter(func(filterPath *ytbx.Path) bool {
		for _, pathString := range paths {
			path, err := ytbx.ParsePathStringUnsafe(pathString)
			if err == nil && filterPath != nil && path.String() == filterPath.String() {
				return true
			}
		}

		return false
	})
}

func isGoPath(p string) bool { return strings.HasPrefix(p, "/") }

// TODO(Thearas): Prefix search.
// Exclude accepts YAML paths as input and returns a new report with differences without those paths
func (r Report) Exclude(paths ...string) (result Report, err error) {
	if len(paths) == 0 {
		return r, nil
	}

	ps := make([]string, len(paths))
	for i, pathString := range paths {
		if !isGoPath(pathString) {
			return r, fmt.Errorf("exclude path should be a Go Patch, but got %s", pathString)
		}

		path, err := ytbx.ParsePathStringUnsafe(pathString)
		if err != nil {
			return r, err
		}
		ps[i] = path.String()
	}

	return r.filter(func(filterPath *ytbx.Path) bool {
		for _, pathString := range ps {
			if err == nil && filterPath != nil && strings.HasPrefix(filterPath.String(), pathString) {
				return false
			}
		}

		return true
	}), nil
}

// FilterRegexp accepts regular expressions as input and returns a new report with differences for matching those patterns
func (r Report) FilterRegexp(pattern ...string) (result Report) {
	if len(pattern) == 0 {
		return r
	}

	regexps := make([]*regexp.Regexp, len(pattern))
	for i := range pattern {
		regexps[i] = regexp.MustCompile(pattern[i])
	}

	return r.filter(func(filterPath *ytbx.Path) bool {
		for _, regexp := range regexps {
			if filterPath != nil && regexp.MatchString(filterPath.String()) {
				return true
			}
		}
		return false
	})
}

// ExcludeRegexp accepts regular expressions as input and returns a new report with differences for not matching those patterns
func (r Report) ExcludeRegexp(pattern ...string) (result Report) {
	if len(pattern) == 0 {
		return r
	}

	regexps := make([]*regexp.Regexp, len(pattern))
	for i := range pattern {
		regexps[i] = regexp.MustCompile(pattern[i])
	}

	return r.filter(func(filterPath *ytbx.Path) bool {
		for _, regexp := range regexps {
			if filterPath != nil && regexp.MatchString(filterPath.String()) {
				return false
			}
		}
		return true
	})
}
