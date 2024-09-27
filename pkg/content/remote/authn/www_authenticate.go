package authn

import (
	"bytes"
	"errors"
	"sort"
	"strconv"
	"strings"
	"text/scanner"

	"golang.org/x/exp/maps"
)

var errInvalidWwwAuthenticate = errors.New("invalid www-authenticate")

func ParseWwwAuthenticate(str string) (*WwwAuthenticate, error) {
	authType, paramsStr, ok := strings.Cut(str, " ")
	if !ok {
		return nil, errInvalidWwwAuthenticate
	}

	wwwAuth := &WwwAuthenticate{
		AuthType: authType,
	}

	if paramsStr != "" {
		wwwAuth.Params = map[string]string{}

		s := &scanner.Scanner{}
		s.Init(bytes.NewBufferString(paramsStr))

		s.Whitespace = 1<<'\t' | 1<<'\n' | 1<<'\r'

		kv := [2]string{}
		i := 0

		commit := func() {
			i = 0

			if kv[0] != "" {
				if kv[1] != "" && kv[1][0] == '"' {
					v, err := strconv.Unquote(kv[1])
					if err == nil {
						wwwAuth.Params[kv[0]] = v
						return
					}
				}

				wwwAuth.Params[kv[0]] = kv[1]
			}
		}

		for t := s.Scan(); t != scanner.EOF; t = s.Scan() {
			switch t {
			case ',', ' ':
				commit()
				continue
			case '=':
				i = 1
				continue
			}
			kv[i] = s.TokenText()
		}

		commit()
	}

	return wwwAuth, nil
}

type WwwAuthenticate struct {
	AuthType string
	Params   map[string]string
}

func (v WwwAuthenticate) String() string {
	b := &strings.Builder{}
	b.WriteString(v.AuthType)
	b.WriteString(" ")

	keys := maps.Keys(v.Params)
	sort.Strings(keys)

	for i, k := range keys {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(k)
		b.WriteString("=")
		b.WriteString(strconv.Quote(v.Params[k]))
	}

	return b.String()
}
