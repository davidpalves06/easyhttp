package easyhttp

import (
	"errors"
	"fmt"
	"maps"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Cookie struct {
	Name     string
	Value    string
	Expires  time.Time
	MaxAge   int
	Domain   string
	Path     string
	Secure   bool
	HTTPOnly bool
	SameSite SameSite

	creation time.Time
}

func (c *Cookie) String() string {
	cookieStringBuilder := strings.Builder{}
	cookieStringBuilder.WriteString(fmt.Sprintf("%s=%s", c.Name, c.Value))
	if !c.Expires.IsZero() {
		cookieStringBuilder.WriteString(fmt.Sprintf("; Expires=%s", c.Expires.Format(time.RFC1123)))
	}
	if c.MaxAge != 0 {
		cookieStringBuilder.WriteString(fmt.Sprintf("; Max-Age=%d", c.MaxAge))
	}
	if c.Domain != "" {
		cookieStringBuilder.WriteString(fmt.Sprintf("; Domain=%s", c.Domain))
	}
	if c.Path != "" {
		cookieStringBuilder.WriteString(fmt.Sprintf("; Path=%s", c.Path))
	}
	if c.Secure {
		cookieStringBuilder.WriteString("; Secure")
	}
	if c.HTTPOnly {
		cookieStringBuilder.WriteString("; HttpOnly")
	}
	if c.SameSite == SAME_SITE_LAX {
		cookieStringBuilder.WriteString("; SameSite=Lax")
	}
	if c.SameSite == SAME_SITE_STRICT {
		cookieStringBuilder.WriteString("; SameSite=Strict")
	}
	if c.SameSite == SAME_SITE_NONE {
		cookieStringBuilder.WriteString("; SameSite=None")
	}
	return cookieStringBuilder.String()
}

type SameSite int

const (
	SAME_SITE_DEFAULT = iota
	SAME_SITE_LAX
	SAME_SITE_STRICT
	SAME_SITE_NONE
)

type CookieStorage struct {
	cookieMap map[string]map[string]*Cookie
}

func newCookieStorage() *CookieStorage {
	return &CookieStorage{
		cookieMap: make(map[string]map[string]*Cookie),
	}
}

func (cs *CookieStorage) SetCookies(url *url.URL, cookies []*Cookie) {
	for _, c := range cookies {
		var host string = url.Hostname()
		if c.Domain != "" {
			host = c.Domain
		}
		if _, ok := cs.cookieMap[host]; !ok {
			cs.cookieMap[host] = make(map[string]*Cookie)
		}
		if !c.Expires.IsZero() && c.Expires.Before(time.Now().UTC()) {
			delete(cs.cookieMap[host], c.Name)
		} else {
			cs.cookieMap[host][c.Name] = c
		}
	}
}

func (cs *CookieStorage) Cookies(url *url.URL) []*Cookie {
	var host = url.Hostname()
	var cookies []*Cookie

	if _, ok := cs.cookieMap[host]; ok {
		cookieIterator := maps.Values(cs.cookieMap[host])
		for cookie := range cookieIterator {
			if !cookie.Expires.IsZero() && cookie.Expires.Before(time.Now().UTC()) {
				delete(cs.cookieMap[host], cookie.Name)
				continue
			}
			if cookie.Secure && url.Scheme != "https" {
				continue
			}
			if matchesPath(url.Path, cookie.Path) {
				cookies = append(cookies, cookie)
			}
		}
	}
	return cookies
}

func matchesPath(requestPath, cookiePath string) bool {
	if cookiePath == "" {
		return true
	}
	if strings.HasPrefix(requestPath, cookiePath) {
		return true
	}
	return false
}

func parseSetCookieLine(cookieLine string) (*Cookie, error) {
	splittedCookie := strings.Split(strings.TrimSpace(cookieLine), ";")
	if len(splittedCookie) < 1 {
		return nil, errors.New("bad cookie line")
	}
	var cookie = &Cookie{
		creation: time.Now(),
		SameSite: SAME_SITE_DEFAULT,
	}
	valuePair := strings.Split(splittedCookie[0], "=")
	if len(valuePair) < 2 {
		return nil, errors.New("bad name value pair")
	}
	cookie.Name = valuePair[0]
	cookie.Value = strings.Join(valuePair[1:], "=")

	for _, attribute := range splittedCookie[1:] {
		splittedAttribute := strings.Split(attribute, "=")
		switch strings.TrimSpace(strings.ToLower(splittedAttribute[0])) {
		case "expires":
			if !cookie.Expires.IsZero() {
				continue
			}
			if len(splittedAttribute) < 2 {
				return nil, errors.New("bad attribute")
			}
			expireTime, err := time.Parse(time.RFC1123, splittedAttribute[1])
			if err != nil {
				return nil, errors.New("bad expire value")
			}
			cookie.Expires = expireTime.UTC()
		case "max-age":
			if len(splittedAttribute) < 2 {
				return nil, errors.New("bad attribute")
			}
			maxAge, err := strconv.ParseInt(splittedAttribute[1], 10, 64)
			if err != nil {
				return nil, errors.New("bad max age value")
			}
			cookie.MaxAge = int(maxAge)
			cookie.Expires = cookie.creation.Add(time.Duration(maxAge) * time.Second).UTC()
		case "secure":
			cookie.Secure = true
		case "httponly":
			cookie.HTTPOnly = true
		case "domain":
			if len(splittedAttribute) < 2 {
				return nil, errors.New("bad attribute")
			}
			cookie.Domain = splittedAttribute[1]
		case "path":
			if len(splittedAttribute) < 2 {
				return nil, errors.New("bad attribute")
			}
			cookie.Path = splittedAttribute[1]
		case "samesite":
			if len(splittedAttribute) < 2 {
				return nil, errors.New("bad attribute")
			}
			switch strings.ToLower(splittedAttribute[1]) {
			case "lax":
				cookie.SameSite = SAME_SITE_LAX
			case "strict":
				cookie.SameSite = SAME_SITE_STRICT
			case "none":
				cookie.SameSite = SAME_SITE_NONE
				cookie.Secure = true
			default:
				return nil, errors.New("bad samesite value")
			}
		default:
			return nil, errors.New("unknown attribute")
		}
	}
	return cookie, nil
}
