package authn

import (
	"regexp"
	"testing"

	. "github.com/octohelm/x/testing/v2"
)

func TestParseWwwAuthenticate(t *testing.T) {
	t.Run("验证 WwwAuthenticate 序列化", func(t *testing.T) {
		a := &WwwAuthenticate{
			AuthType: "Bearer",
			Params: map[string]string{
				"realm":   "http://localhost/token",
				"service": "test",
			},
		}

		Then(t, "序列化结果应该正确",
			Expect(a.String(), Equal(`Bearer realm="http://localhost/token", service="test"`)),
		)
	})

	t.Run("解析带引号的参数", func(t *testing.T) {
		expected := &WwwAuthenticate{
			AuthType: "Bearer",
			Params: map[string]string{
				"realm":   "http://localhost/token",
				"service": "test",
			},
		}

		parsed := MustValue(t, func() (*WwwAuthenticate, error) {
			return ParseWwwAuthenticate(`Bearer realm="http://localhost/token" service=test`)
		})

		Then(t, "解析结果应该匹配",
			Expect(parsed, Equal(expected)),
		)
	})

	t.Run("解析不带引号的参数", func(t *testing.T) {
		input := `Basic realm=test`
		expected := &WwwAuthenticate{
			AuthType: "Basic",
			Params: map[string]string{
				"realm": "test",
			},
		}

		Then(t, "应该成功解析",
			ExpectMustValue(func() (*WwwAuthenticate, error) {
				return ParseWwwAuthenticate(input)
			}, Equal(expected)),
		)
	})

	t.Run("解析多个参数", func(t *testing.T) {
		input := `Digest realm="test@host.com", qop="auth,auth-int", nonce="dcd98b7102dd2f0e8b11d0f600bfb0c093", opaque="5ccc069c403ebaf9f0171e9517f40e41"`
		expected := &WwwAuthenticate{
			AuthType: "Digest",
			Params: map[string]string{
				"realm":  "test@host.com",
				"qop":    "auth,auth-int",
				"nonce":  "dcd98b7102dd2f0e8b11d0f600bfb0c093",
				"opaque": "5ccc069c403ebaf9f0171e9517f40e41",
			},
		}

		Then(t, "应该正确解析复杂参数",
			ExpectMustValue(func() (*WwwAuthenticate, error) {
				return ParseWwwAuthenticate(input)
			}, Equal(expected)),
		)
	})

	t.Run("错误处理", func(t *testing.T) {
		t.Run("无效格式应该返回错误", func(t *testing.T) {
			Then(t, "空字符串应该返回错误",
				ExpectDo(
					func() error {
						_, err := ParseWwwAuthenticate("")
						return err
					},
					ErrorMatch(regexp.MustCompile("invalid www-authenticate")),
				),
			)

			Then(t, "缺少 AuthType 应该返回错误",
				ExpectDo(
					func() error {
						_, err := ParseWwwAuthenticate(`realm="test"`)
						return err
					},
					ErrorMatch(regexp.MustCompile("invalid www-authenticate")),
				),
			)
		})
	})
}
