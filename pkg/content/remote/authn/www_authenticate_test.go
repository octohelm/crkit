package authn

import (
	"testing"

	testingx "github.com/octohelm/x/testing"
)

func TestParseWwwAuthenticate(t *testing.T) {
	a := &WwwAuthenticate{
		AuthType: "Bearer",
		Params: map[string]string{
			"realm":   "http://localhost/token",
			"service": "test",
		},
	}

	testingx.Expect(t, a.String(), testingx.Be(`Bearer realm="http://localhost/token", service="test"`))

	parsed, err := ParseWwwAuthenticate(`Bearer realm="http://localhost/token" service=test`)
	testingx.Expect(t, err, testingx.BeNil[error]())
	testingx.Expect(t, parsed, testingx.Equal(a))
}
