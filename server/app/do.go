package app

import (
	"encoding/base64"
	"encoding/json"
	"html/template"
	"strings"
	"time"

	"github.com/scys-devs/lib-go/server/service/nacos"

	"github.com/awa/go-iap/appstore"
	"github.com/awa/go-iap/playstore"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/scys-devs/lib-go"
	"google.golang.org/api/androidpublisher/v3"
)

const keyUserContext = "userContext"

var JwtSecret = []byte("p6Ui5WLsJfDk9hoA2cNAyglM5llFSqtz")
var ApplePassword = ""
var GooglePassword []byte
var Language []string
var LanguageData = make(map[string]map[string]template.HTML)
var HermitRule []nacos.ConfigComputedRule

var purchaseLogger = lib.GetLogger("purchase-raw")

type UserDO struct {
	Id           int64  `db:"id"`
	Uuid         string `db:"uuid"`
	Device       string `db:"device"`
	DeviceSystem string `db:"device_system"`
	IpAddr       string `db:"ip_addr"`
	GmtCreate    int64  `db:"gmt_create"`
	// 付费套餐
	SubsExpiresAt int64  `db:"subs_expires_at"`
	SubsPkgId     string `db:"subs_pkg_id"`
	// 其他属性
	FcmToken       string `db:"fcm_token"`
	Lang           string `db:"lang"`
	TimezoneOffset int    `db:"timezone_offset"`
}

func (do UserDO) ToContext() *UserContext {
	return &UserContext{Id: do.Id, SubsExpiresAt: do.SubsExpiresAt, SubsPkgId: do.SubsPkgId}
}

// UserContext 引入JWT只是为了写入一些基础信息，不做判断
// 后续使用session来维护
type UserContext struct {
	jwt.StandardClaims
	// 用户信息
	Id            int64  `json:"id"`
	SubsExpiresAt int64  `json:"subs_expires_at"`
	SubsPkgId     string `json:"subs_pkg_id"`
	// app信息
	BundleId string `json:"-"`
	Version  string `json:"-"`
	Platform string `json:"-"`
}

// IsExpire 会员是否有效
func (uc *UserContext) IsExpire() bool {
	return uc.SubsExpiresAt > 0 && uc.SubsExpiresAt < time.Now().Unix()
}

func (uc *UserContext) IsAndroid() bool {
	return uc.Platform == "Android"
}

func (uc *UserContext) UpdateToken(c *gin.Context) string {
	token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, uc).SignedString(JwtSecret)
	c.Header("set-token", token)
	return token
}

type PurchaseDO struct {
	Id        int64  `db:"id"`
	UserId    int64  `db:"user_id"`
	PkgId     string `db:"pkg_id"`
	TxnId     string `db:"txn_id"`
	GmtCreate int64  `db:"gmt_create"`
	GmtExpire int64  `db:"gmt_expire"`
	GmtRefund int64  `db:"gmt_refund"`
	Env       string `db:"env"`
	Platform  int8   `db:"platform"`
}

func (do *PurchaseDO) FromInApp(item appstore.InApp) {
	do.PkgId = item.ProductID
	do.TxnId = item.TransactionID
	do.GmtCreate = lib.StrToInt64(item.PurchaseDateMS) / 1000
	do.GmtExpire = lib.StrToInt64(item.ExpiresDateMS) / 1000
	do.GmtRefund = lib.StrToInt64(item.CancellationDateMS) / 1000
	do.Env = "Production"
	if do.GmtExpire-do.GmtCreate < 86400 {
		do.Env = "Sandbox"
	}
}

func (do *PurchaseDO) FromAndroidPurchase(item *androidpublisher.SubscriptionPurchase) {
	do.TxnId = item.OrderId
	do.GmtCreate = item.StartTimeMillis / 1000
	do.GmtExpire = item.ExpiryTimeMillis / 1000
	do.GmtRefund = item.UserCancellationTimeMillis / 1000
	do.Env = "Production"
	if do.GmtExpire-do.GmtCreate < 86400 {
		do.Env = "Sandbox"
	}
}

type PurchaseSubsDO struct {
	Id         int64  `db:"id"`
	OriginalId string `db:"original_id"`
	PkgId      string `db:"pkg_id"`
	Periods    int    `db:"periods"`
	GmtCreate  int64  `db:"gmt_create"`
	GmtLatest  int64  `db:"gmt_latest"`
	GmtCancel  int64  `db:"gmt_cancel"`
	Platform   int8   `db:"platform"`
}

// google pay回调格式
type GooglePayCallBack struct {
	Message      *GooglePayCallBackData `json:"message"`
	Subscription string                 `json:"subscription"`
}

// google pay回调中 message格式
type GooglePayCallBackData struct {
	Data         string `json:"data"`
	MessageId    string `json:"messageId"`
	Message_Id   string `json:"message_id"`
	PublishTime  string `json:"publishTime"`
	Publish_Time string `json:"publish_time"`
}

// google pay 验单接口参数
type GooglePayCheckPost struct {
	Package       string `json:"packageName"`
	OrderId       string `json:"orderId"`
	PurchaseToken string `json:"purchaseToken"`
	ProductID     string `json:"productId"`
}

// google pay 回调base64字符串解码格式
type GooglePayBaseData struct {
	Version                  string                              `json:"version"`
	PackageName              string                              `json:"packageName"`
	EventTimeMillis          string                              `json:"eventTimeMillis"`
	SubscriptionNotification *playstore.SubscriptionNotification `json:"subscriptionNotification"`
}

// 获取google pay回调中的data
func GetBase64Data(data *GooglePayCallBack) (result *GooglePayBaseData, err error) {
	// data.Message.Data base64字符串 base64解码
	base64Str, err := base64.StdEncoding.DecodeString(data.Message.Data)
	if err != nil {
		return nil, err
	}
	// data.Message.Data []byte 转结构体
	err = json.Unmarshal(base64Str, &result)
	if err != nil {
		return nil, err
	}
	return
}

// 处理谷歌订单id 返回基础订单号和订阅次数
func GetGoogleOrderTimes(orderId string) (baseOrderId string, times int) {
	arr := strings.Split(orderId, "..")
	baseOrderId = arr[0]
	times = 1
	if len(arr) == 2 {
		times = lib.StrToInt(arr[1]) + 2
	}
	return

}
