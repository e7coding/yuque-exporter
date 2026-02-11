package book

import (
	"github.com/go-rod/rod/lib/proto"
	"net/http"
	"strings"
)

// HttpCookiesToProto 将标准库 Cookie 转为 proto.NetworkCookie
func HttpCookiesToProto(cookies []*http.Cookie) []*proto.NetworkCookieParam {
	var res []*proto.NetworkCookieParam
	for _, c := range cookies {
		res = append(res, &proto.NetworkCookieParam{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Secure:   c.Secure,
			HTTPOnly: c.HttpOnly,
			SameSite: convertSameSite(c.SameSite),
			Expires:  proto.TimeSinceEpoch(c.Expires.Unix()),
		})
	}
	return res
}

// CookieMapToHeader 将 map 转换为 Cookie Header 字符串
func CookieMapToHeader(cookies []*proto.NetworkCookie) string {
	if len(cookies) == 0 {
		return ""
	}
	var parts []string
	for _, cookie := range cookies {
		v := strings.TrimSpace(cookie.Value)
		if v == "" {
			continue
		}

		// 基础安全处理
		v = strings.ReplaceAll(v, ";", "%3B")
		v = strings.ReplaceAll(v, "\n", "")
		v = strings.ReplaceAll(v, "\r", "")

		parts = append(parts, cookie.Name+"="+v)
	}
	return strings.Join(parts, "; ")
}

// convertSameSite 转换 SameSite 策略
func convertSameSite(samesite http.SameSite) proto.NetworkCookieSameSite {
	switch samesite {
	case http.SameSiteDefaultMode:
		return proto.NetworkCookieSameSiteNone
	case http.SameSiteStrictMode:
		return proto.NetworkCookieSameSiteStrict
	case http.SameSiteLaxMode:
		return proto.NetworkCookieSameSiteLax
	default:
		return proto.NetworkCookieSameSiteNone
	}
}
