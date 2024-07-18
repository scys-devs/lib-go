package server

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

// UserContextParser 用于处理用户信息
type UserContextParser struct {
	Key        string // 存储在context中的名字
	CookieName string
	Secret     []byte
	Domain     string // 支持子域名鉴权
}

// 默认7天
func (parser *UserContextParser) Save(c *gin.Context, uc jwt.Claims, keep bool) {
	var maxAge = 7 * 86400
	if keep {
		maxAge = 0
	}
	token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, uc).SignedString(parser.Secret)
	c.SetCookie(parser.CookieName, token, maxAge, "/", parser.Domain, false, false)
}

func (parser *UserContextParser) Parse(c *gin.Context, uc jwt.Claims) (ok bool) {
	cookie, _ := c.Cookie(parser.CookieName)
	if len(cookie) == 0 { // 防止无法写入cookie，从header备用字段再读取
		cookie = c.GetHeader("X-TOKEN")
	}
	if len(cookie) == 0 {
		return
	}
	ok = parser.ParseFromString(cookie, uc)
	c.Set(parser.Key, uc)
	return
}

func (parser *UserContextParser) ParseFromString(cookie string, uc jwt.Claims) (ok bool) {
	if tmp, err := jwt.ParseWithClaims(cookie, uc, func(token *jwt.Token) (interface{}, error) {
		return parser.Secret, nil
	}); err == nil && tmp.Valid {
		ok = true
	}
	return
}
