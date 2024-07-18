package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"strings"

	"github.com/RaveNoX/go-jsonmerge"
	"github.com/awa/go-iap/appstore"
	"github.com/awa/go-iap/playstore"
	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"

	"github.com/scys-devs/lib-go"
	"github.com/scys-devs/lib-go/conn"
	"github.com/scys-devs/lib-go/server"
	"github.com/scys-devs/lib-go/server/service/nacos"
)

var HermitCustomFunction func(c *gin.Context, hermit map[string]*HermitDO) []byte
var HermitKey = map[string]HermitKeyDO{
	"ios": {Hermit: "hermit", HermitComputed: "hermit_computed"},
}

type HermitKeyDO struct {
	Hermit         string
	HermitComputed string
}

// hermit 由基础配置+计算配置得出
type HermitDO struct {
	Hermit         json.RawMessage
	HermitComputed json.RawMessage
}

var hermit = make(map[string]*HermitDO)

func loadLanguage() {
	for _, lang := range Language {
		_ = nacos.ParseConfigFromNacos("language_"+lang, true, func(content string) error {
			var tmp = make(map[string]template.HTML)
			_ = jsoniter.UnmarshalFromString(content, &tmp)
			LanguageData[lang] = tmp
			return nil
		})
	}
}

// GetLangCode 获取语言代码
func GetLangCode(raw string) string {
	locate := strings.Split(raw, "_")
	if locate[0] == "zh" {
		if locate[1] == "TW" || (len(locate) > 2 && locate[2] == "TW") {
			return "zh_TW"
		} else {
			return "zh_CN"
		}
	} else if locate[0] == "pt" {
		return "pt_BR"
	} else if locate[0] == "de" {
		return "de_DE"
	} else {
		// 标准语言包文件名过长 只截取国家部分代码
		return locate[0]
	}
}

// T 获取翻译文本
func T(c *gin.Context) map[string]template.HTML {
	if len(LanguageData) == 0 {
		loadLanguage()
	}

	// 处理一下语言
	locale := GetLangCode(c.Request.Header.Get("Accept-Language"))
	if lang, ok := LanguageData[locale]; ok {
		return lang
	} else {
		return LanguageData["en"]
	}
}

type Controller struct{}

func (c Controller) Register(e *gin.RouterGroup) {
	e.POST("/login", c.login)
	e.POST("/purchase/apple/notice", c.purchaseNoticeApple)
	e.POST("/purchase/google/notice", c.purchaseNoticeGoogle)

	g := e.Group("", UserAuth())
	g.GET("/", c.hermit)

	ug := g.Group("/user")
	ug.POST("/notice/token", c.noticeToken)

	pg := g.Group("/purchase")
	pg.POST("/validate", c.purchaseValidate)
}

func (Controller) hermit(c *gin.Context) {
	defer func() { // 兼容下如果不要nacos的应用
		if r := recover(); r != nil {
			server.SendOK(c, gin.H{
				"env": conn.ENV,
			})
		}
	}()

	var uc = GetUserContext(c)
	var configComputed = nacos.NewConfigComputed(c)
	for _, item := range HermitRule {
		configComputed.Set(item)
	}
	if _, ok := HermitKey[uc.Platform]; !ok {
		server.SendErr(c, fmt.Errorf("not found hermit for %v", uc.Platform))
		return
	}

	// 加载缓存
	if hermit[uc.Platform] == nil {
		hermit[uc.Platform] = &HermitDO{}
		if err := nacos.ParseConfigFromNacos(HermitKey[uc.Platform].Hermit, true, func(content string) error {
			hermit[uc.Platform].Hermit = json.RawMessage(content)
			return nil
		}); err != nil {
			server.CtrlLogger.Errorw("get hermit", "err", err)
		}

		// 然后加载计算型配置
		_ = nacos.ParseConfigFromNacos(HermitKey[uc.Platform].HermitComputed, true, func(content string) error {
			hermit[uc.Platform].HermitComputed = json.RawMessage(content)
			return nil
		})
	}

	patch, _ := configComputed.ConfigComputeValue(hermit[uc.Platform].HermitComputed, true)
	patchBytes, _ := jsoniter.Marshal(patch)
	out, _, _ := jsonmerge.MergeBytes(hermit[uc.Platform].Hermit, patchBytes)

	// 用于加载特殊逻辑，也是通过json来merge
	if HermitCustomFunction != nil {
		if tmp := HermitCustomFunction(c, hermit); tmp != nil {
			out, _, _ = jsonmerge.MergeBytes(out, tmp)
		}
	}

	if len(Language) > 0 {
		outTemplate := template.Must(template.New("parse").Parse(string(out)))
		var b = new(bytes.Buffer)
		_ = outTemplate.Execute(b, T(c))
		out = b.Bytes()
	}

	server.SendOK(c, gin.H{
		"env":    conn.ENV,
		"hermit": json.RawMessage(out),
	})
}

func (Controller) login(c *gin.Context) {
	b, _ := c.GetRawData()
	args := new(struct {
		Uuid     string `json:"uuid" form:"uuid"`
		Device   string `json:"device" form:"device"`
		System   string `json:"system" form:"system"`
		Timezone int    `json:"timezone" form:"timezone"`
	})
	_ = json.Unmarshal(b, args)

	// 兼容scanner数据格式
	if len(args.Uuid) == 0 {
		scannerArgs := new(struct {
			Raw string `json:"raw" form:"raw"`
		})
		_ = json.Unmarshal(b, scannerArgs)
		_ = json.Unmarshal(lib.StrToBytes(scannerArgs.Raw), args)
	}

	var u UserDO
	if args.Uuid == "0000-0000-0000-0000" { // 测试用户
		u = UserDO{Id: 1, SubsExpiresAt: 253392422400}
	} else {
		id := Dao.Login(UserDO{
			Uuid:           args.Uuid,
			Device:         args.Device,
			DeviceSystem:   args.System,
			IpAddr:         c.ClientIP(),
			Lang:           c.Request.Header.Get("Accept-Language"),
			TimezoneOffset: args.Timezone,
		})
		u = Dao.Get(id)
	}
	// 更新签名
	token := u.ToContext().UpdateToken(c)

	server.SendOK(c, gin.H{
		"token": token,
	})
}

func (Controller) noticeToken(c *gin.Context) {
	args := new(struct {
		FcmToken string `json:"fcm_token"`
		Timezone int    `json:"timezone"`
	})
	c.Bind(args)

	uc := GetUserContext(c)
	u := Dao.Get(uc.Id)
	u.FcmToken = args.FcmToken
	u.TimezoneOffset = args.Timezone
	u.IpAddr = c.ClientIP()
	u.Lang = c.Request.Header.Get("Accept-Language")
	Dao.Login(u)
	// 更新用户token
	token := u.ToContext().UpdateToken(c)
	server.SendOK(c, gin.H{
		"token": token,
	})
}

func (Controller) purchaseNoticeApple(c *gin.Context) {
	args := new(appstore.SubscriptionNotification)
	c.BindJSON(args)

	purchaseLogger.Infow("purchase apple notice", "raw", args)
	server.SendOK(c, nil) // 由于是返回给苹果的，所以先直接写结果了

	resp := &appstore.IAPResponse{}
	if err := appstore.New().Verify(context.Background(), appstore.IAPRequest{
		ReceiptData: args.UnifiedReceipt.LatestReceipt,
		Password:    ApplePassword,
	}, resp); err != nil {
		server.CtrlLogger.Errorw("purchase notice verify", "err", err, "raw", args)
		return
	}

	latest := resp.LatestReceiptInfo[0]
	// 给用户更新上时间
	info := PurchaseDO{}
	info.FromInApp(latest)
	txn := Dao.GetPurchase(latest.OriginalTransactionID)

	// 如果是付费失败，只增加有效期
	if args.NotificationType == appstore.NotificationTypeDidFailToRenew {
		info.GmtExpire += 7 * 86400
	}

	if err := Dao.UpdateUserSubs(txn.UserId, info); err != nil {
		server.CtrlLogger.Errorw("purchase notice update user subs", "err", err, "userId", txn.UserId)
	}

	// 更新subs
	var gmtCancel int64 = 0
	if len(args.CancellationDateMS) > 0 {
		gmtCancel = lib.StrToInt64(args.CancellationDateMS) / 1000
	} else if len(args.AutoRenewStatusChangeDateMS) > 0 {
		gmtCancel = lib.StrToInt64(args.AutoRenewStatusChangeDateMS) / 1000
	}
	Dao.PutPurchaseSubs(PurchaseSubsDO{
		OriginalId: latest.OriginalTransactionID,
		PkgId:      latest.ProductID,
		Periods:    len(resp.LatestReceiptInfo),
		GmtCreate:  lib.StrToInt64(latest.OriginalPurchaseDateMS) / 1000,
		GmtLatest:  lib.StrToInt64(latest.PurchaseDateMS) / 1000,
		GmtCancel:  gmtCancel,
		Platform:   1,
	})
}

func (Controller) purchaseNoticeGoogle(c *gin.Context) {
	// 获取google pay 推送数据
	args := new(GooglePayCallBack)
	c.Bind(args)

	purchaseLogger.Infow("purchase google notice", "raw", args)
	server.SendOK(c, nil)

	// 解析base64字符串数据
	callBack, err := GetBase64Data(args)
	if err != nil {
		server.CtrlLogger.Errorw("verify google base64", "err", err)
		server.SendErr(c, err)
		return
	}

	// 验证google pay 订单信息
	client, err := playstore.New(GooglePassword)
	if err != nil {
		server.CtrlLogger.Errorw("new google client", "err", err)
		server.SendErr(c, err)
		return
	}
	if callBack.SubscriptionNotification == nil {
		info, _ := jsoniter.MarshalToString(callBack)
		fmt.Println("无订阅消息", info)
		return
	}
	resp, err := client.VerifySubscription(context.Background(), callBack.PackageName, callBack.SubscriptionNotification.SubscriptionID, callBack.SubscriptionNotification.PurchaseToken)
	if err != nil {
		server.CtrlLogger.Errorw("verify purchase google", "err", err)
		server.SendErr(c, err)
		return
	}

	// 分割谷歌订单id和订阅次数
	orderId, periods := GetGoogleOrderTimes(resp.OrderId)
	first := Dao.GetPurchase(orderId)
	// 首次订阅 未生成purchase信息时 不处理purchase_sub
	if err != nil {
		server.CtrlLogger.Errorw("first purchase not found", "err", err)
	}

	resp.OrderId = orderId

	txn := PurchaseDO{
		UserId:   first.UserId,
		PkgId:    callBack.SubscriptionNotification.SubscriptionID,
		Platform: 2,
	}
	txn.FromAndroidPurchase(resp)

	// 宽限期 增加3天有效期 （按照谷歌后台订阅设置来设置）
	if callBack.SubscriptionNotification.NotificationType == 6 {
		txn.GmtExpire += 3 * 86400
	}

	// 获取purchase_sub 获取最迟的有效时间 (订阅多个 同时订阅情况处理)
	purchaseSub := Dao.GetPurchaseSub(first.TxnId)
	if purchaseSub.GmtLatest > (resp.ExpiryTimeMillis / 1000) {
		server.CtrlLogger.Infow("purchase ExpiryTimeMillis err", "resp", resp, "periods", periods)
		resp.ExpiryTimeMillis = purchaseSub.GmtLatest
		txn.GmtExpire = purchaseSub.GmtLatest
	}

	// 更新用户信息
	if err = Dao.UpdateUserSubs(txn.UserId, txn); err != nil {
		server.CtrlLogger.Errorw("purchase notice update user subs", "err", err, "userId", txn.UserId)
	}

	var gmtCancel int64 = 0

	if resp.AutoResumeTimeMillis > 0 {
		// 用户请求暂停订阅
		gmtCancel = resp.AutoResumeTimeMillis / 1000
	} else if resp.UserCancellationTimeMillis > 0 {
		// 用户自动订阅
		gmtCancel = resp.UserCancellationTimeMillis / 1000
	}

	// 更新subs
	Dao.PutPurchaseSubs(PurchaseSubsDO{
		OriginalId: orderId,
		PkgId:      callBack.SubscriptionNotification.SubscriptionID,
		Periods:    periods,
		GmtCreate:  resp.StartTimeMillis / 1000,
		GmtLatest:  resp.StartTimeMillis / 1000,
		GmtCancel:  gmtCancel,
		Platform:   2,
	})
}

func (Controller) purchaseValidate(c *gin.Context) {
	args := new(struct {
		Receipt  string `json:"receipt" form:"receipt"`
		PlatForm string `json:"platform" form:"platform"`
	})
	c.Bind(args)

	uc := GetUserContext(c)
	var pur = make([]PurchaseDO, 0, 4)

	if args.PlatForm == "android" {
		var receipt GooglePayCheckPost
		_ = json.Unmarshal(lib.StrToBytes(args.Receipt), &receipt)

		client, err := playstore.New(GooglePassword)
		resp, err := client.VerifySubscription(context.Background(), receipt.Package, receipt.ProductID, receipt.PurchaseToken)
		if err != nil {
			server.CtrlLogger.Errorw("verify purchase google", "userId", uc.Id, "err", err)
			server.SendErr(c, err)
			return
		}
		// 由于安卓校验出来是单次的，所以我转换一下
		tmp := PurchaseDO{
			UserId:   uc.Id,
			PkgId:    receipt.ProductID,
			Platform: 2,
		}
		tmp.FromAndroidPurchase(resp)
		pur = append(pur, tmp)
	} else {
		resp := &appstore.IAPResponse{}
		if err := appstore.New().Verify(context.Background(), appstore.IAPRequest{
			ReceiptData: args.Receipt,
			Password:    ApplePassword,
		}, resp); err != nil {
			server.CtrlLogger.Errorw("verify purchase apple", "userId", uc.Id, "err", err)
			server.SendErr(c, err)
			return
		}
		for _, item := range resp.LatestReceiptInfo {
			tmp := PurchaseDO{
				UserId:   uc.Id,
				Platform: 1,
			}
			tmp.FromInApp(item)
			pur = append(pur, tmp)
		}
	}

	latest, err := Dao.PutPurchase(pur)
	if err != nil {
		server.SendErr(c, err)
		return
	}

	token := Dao.Get(uc.Id).ToContext().UpdateToken(c)
	server.SendOK(c, gin.H{
		"token":           token,
		"subs_expires_at": latest.GmtExpire,
		"subs_pkg_id":     latest.PkgId,
	})
}
