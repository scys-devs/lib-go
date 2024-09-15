package server

import (
	"github.com/gin-gonic/gin"
	"regexp"
	"strings"
)

var mobileRe, _ = regexp.Compile("(?i:Mobile|iPhone|Android|Opera Mini|UCWEB)")

type UserAgentDetector int

const (
	UserAgentMobile UserAgentDetector = 1 << iota
	UserAgentPC
	UserAgentWX
	UserAgentXQ
)

func UserAgent(c *gin.Context) string {
	return strings.ToLower(c.GetHeader("User-Agent"))
}

func UserAgentDetect(c *gin.Context, typ UserAgentDetector) bool {
	var curr = UserAgentDetector(c.GetInt("UserAgentDetector"))
	if curr == 0 {
		ua := UserAgent(c)
		if len(mobileRe.FindString(ua)) > 0 {
			curr |= UserAgentMobile
		} else {
			curr |= UserAgentPC
		}
		if strings.Contains(ua, "micromessenger") {
			curr |= UserAgentWX
		}
		if strings.Contains(ua, "xiaomiquan") {
			curr |= UserAgentXQ
		}
		c.Set("UserAgentDetector", curr)
	}
	return curr&typ == typ
}
