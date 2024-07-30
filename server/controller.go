package server

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/foolin/goview/supports/ginview"
	"github.com/gin-gonic/gin"
	"github.com/scys-devs/lib-go/conn"
	"github.com/xuri/excelize/v2"
)

// 支持外部自定义参数
var (
	SendKeyCode    = "code"
	SendKeyMessage = "message"
	SendKeyData    = "data"
	ContextHTML    func(ctx *gin.Context) gin.H

	KeyUser = "user"
)

type Controller interface {
	Register(e *gin.RouterGroup)
}

type ExcelExportData struct {
	Name string
	Data [][]string
}

func GetBody(c *gin.Context) string {
	bodyBytes, _ := ioutil.ReadAll(c.Request.Body)
	c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes)) // 防止后续还需要bind等行为无法获取数据
	return string(bodyBytes)
}

func FormFile(c *gin.Context, name string) ([]byte, string, error) {
	f, err := c.FormFile(name)
	if err != nil {
		return nil, "", err
	}
	src, err := f.Open()
	if err != nil {
		return nil, "", err
	}
	dst := bytes.NewBufferString("")
	_, err = io.Copy(dst, src)

	src.Close()

	return dst.Bytes(), f.Filename, err
}

func Send(c *gin.Context, code int, message string, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		SendKeyCode:    code,
		SendKeyMessage: message,
		SendKeyData:    data,
	})
}

func SendOK(c *gin.Context, data interface{}) {
	message, _ := c.Value(SendKeyMessage).(string)
	Send(c, 0, message, data)
}

func SendErr(c *gin.Context, err error) {
	m, ok := c.Value(SendKeyMessage).(string)
	AccessLogger.Errorw("handler request", "err", err, "uri", c.Request.RequestURI, "message", m, "user", c.Value(KeyUser))
	if !ok { // 尝试处理下常见错误
		if err == nil {
		} else if strings.Contains(err.Error(), "Duplicate entry") {
			m = "数据重复"
		} else if strings.HasPrefix(err.Error(), "err:") {
			m = err.Error()
		} else {
			m = "发生未知错误"
		}
	}

	Send(c, -1, m, nil)
}

func SendHTML(c *gin.Context, name string, data interface{}) {
	var h gin.H
	if ContextHTML == nil {
		h = gin.H{}
	} else {
		h = ContextHTML(c)
	}
	h["data"] = data
	ginview.HTML(c, http.StatusOK, name, h)
}

func SendExcel(c *gin.Context, name string, line [][]string) {
	SendExcelMulti(c, name, ExcelExportData{Name: "Sheet1", Data: line})
}

func SendExcelMulti(c *gin.Context, name string, extends ...ExcelExportData) string {
	var f = excelize.NewFile()

	for _, extend := range extends {
		f.NewSheet(extend.Name)
		for i, _ := range extend.Data {
			idx, _ := excelize.CoordinatesToCellName(1, i+1)
			_ = f.SetSheetRow(extend.Name, idx, &extend.Data[i])
		}
	}
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "+", "_")
	exportName := fmt.Sprintf("%v_%v.xlsx", name, time.Now().Unix())
	exportBytes, _ := f.WriteToBuffer() // 检测发现，导出花费的时间主要在这，估计因为导出需要zip
	return SendByte(c, exportName, exportBytes.Bytes())
}

func SendByte(c *gin.Context, name string, b []byte) string {
	_exportName := url.QueryEscape(name)
	attachment := fmt.Sprintf(`attachment; filename*=UTF-8''%v; filename=%v`, _exportName, _exportName)
	_b := bytes.NewReader(b)
	if conn.ENV == "local" || conn.ENV == "local-docker" {
		nameSlice := strings.Split(name, ".")
		mineType := mime.TypeByExtension(nameSlice[len(nameSlice)-1:][0])
		c.Header("Content-Type", mineType)
		c.Header("Content-Disposition", attachment)
		http.ServeContent(c.Writer, c.Request, name, time.Now(), _b)
		return ""
	} else {
		exportPath := fmt.Sprintf("upload/tmp/%v", name)
		_ = conn.GetOSS().PutObject(exportPath, _b, oss.ContentDisposition(attachment))
		//c.Redirect(http.StatusFound, conn.HostOSS+exportPath)
		SendOK(c, gin.H{
			"download": conn.HostOSS + exportPath,
		})
		return conn.HostOSS + exportPath
	}
}

func Redirect(c *gin.Context, name, href string) {
	// 处理下链接，添加一个时间参数
	location, _ := url.Parse(href)
	query := location.Query()
	query.Set("_", fmt.Sprint(time.Now().Unix()))
	location.RawQuery = query.Encode()
	c.Redirect(http.StatusFound, location.String())
}

// 使用JS的方式进行页面跳转
func Redirect2(c *gin.Context, name, href string) {
	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	c.Writer.WriteString(fmt.Sprintf(`<html lang="zh">
<title>%v</title>
<meta charset="UTF-8"/>
<meta name="viewport" content="width=device-width, initial-scale=1.0"/>
<body>
<p id='hint' style='display: none'>跳转%v，如果浏览器没有反应<a href='%v'>请按这里</a></p>
<script>
setTimeout(function () {
	document.getElementById('hint').style.display = 'block'
}, 1200)
window.location.href="%v";
</script>
</body>
</html>`, name, name, href, href))
}

func ClearCookie(c *gin.Context, prefix string) {
	for _, cookie := range c.Request.Cookies() {
		if strings.HasPrefix(cookie.Name, prefix) {
			c.SetCookie(cookie.Name, "", 0, "/", "", false, true)
		}
	}
}
