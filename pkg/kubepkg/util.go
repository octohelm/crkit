package kubepkg

import (
	"github.com/gobwas/glob"
	"github.com/pkg/errors"
	"strings"
)

func Compile(patterns []string) (glob.Glob, error) {
	rr := rules{}

	for _, p := range patterns {
		if strings.HasPrefix(p, "!") {
			g, err := glob.Compile(p[1:])
			if err != nil {
				return nil, errors.Wrapf(err, "compile failed %s", p)
			}
			rr = append(rr, rule{
				glob: g,
				omit: true,
			})
			continue
		}
		g, err := glob.Compile(p)
		if err != nil {
			return nil, errors.Wrapf(err, "compile failed %s", p)
		}
		rr = append(rr, rule{glob: g})
	}

	return rr, nil
}

type rules []rule

func (rr rules) Match(s string) bool {
	for _, x := range rr {
		if x.Match(s) {
			return true
		}
	}
	return false
}

type rule struct {
	glob glob.Glob
	omit bool
}

func (r rule) Match(s string) bool {
	if r.omit {
		return !r.glob.Match(s)
	}
	return r.glob.Match(s)
}
