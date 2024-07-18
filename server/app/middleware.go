package app

import (
	"net/http"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/scys-devs/lib-go/server"
)

func GetUserContext(c *gin.Context) *UserContext {
	if val, ok := c.Get(keyUserContext); ok {
		return val.(*UserContext)
	} else {
		return &UserContext{}
	}
}

func UserAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := parseToken(c); !ok { // 如果校验授权失败，那么我要记录下
			server.AccessLogger.Errorw("parse token failed", "uri", c.Request.URL.String(), "token", c.Request.Header.Get("token"))
			c.AbortWithStatus(http.StatusUnauthorized)
		}
		c.Next()
	}
}

// 对于部分接口，可能接受用户没授权，但是有需要用户信息，所以单独拆出该函数
func parseToken(c *gin.Context) (uc *UserContext, ok bool) {
	uc = new(UserContext)

	if tmp, err := jwt.ParseWithClaims(c.Request.Header.Get("token"), uc, func(token *jwt.Token) (interface{}, error) { // 可以再校验签名算法
		return JwtSecret, nil
	}); err == nil && tmp.Valid && uc.Id > 0 {
		// 如果用户过期了，那么重新拿一下
		if uc.IsExpire() {
			uc = Dao.Get(uc.Id).ToContext()
			uc.UpdateToken(c)
		}

		ok = true
	}

	// 补充一些基础参数
	uc.BundleId = c.Request.Header.Get("bundle-id")
	uc.Version = c.Request.Header.Get("version")
	uc.Platform = c.Request.Header.Get("platform")
	if len(uc.Platform) == 0 {
		uc.Platform = c.DefaultQuery("platform", "ios")
	}

	c.Set(keyUserContext, uc)
	return
}
