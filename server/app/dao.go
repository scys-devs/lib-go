package app

import (
	"time"

	"github.com/scys-devs/lib-go/conn"
	"github.com/scys-devs/lib-go/server"
)

var Dao = dao{}

type dao struct {
}

func (dao) Login(do UserDO) (id int64) {
	res, err := conn.GetDB().Exec(`insert into user (uuid, device, device_system, gmt_create, ip_addr, lang, fcm_token, timezone_offset) values (?,?,?,?,?,?,?,?)
		on duplicate key update id=LAST_INSERT_ID(id), device=?, device_system=?, ip_addr=?, lang=?, fcm_token=?, timezone_offset=?`,
		do.Uuid, do.Device, do.DeviceSystem, time.Now().Unix(), do.IpAddr, do.Lang, do.FcmToken, do.TimezoneOffset,
		do.Device, do.DeviceSystem, do.IpAddr, do.Lang, do.FcmToken, do.TimezoneOffset)
	if err != nil {
		server.DaoLogger.Errorw("user login", "err", err, "info", do)
		return
	}
	id, _ = res.LastInsertId()
	return
}

func (dao) Get(id int64) (u UserDO) {
	err := conn.GetDB().Get(&u, `select * from user where id=?`, id)
	if err != nil {
		server.DaoLogger.Errorw("get user", "err", err, "userId", id)
	}
	return
}

func (dao) GetPurchase(txnId string) (item PurchaseDO) {
	err := conn.GetDB().Get(&item, `select * from purchase where txn_id=?`, txnId)
	if err != nil {
		server.DaoLogger.Errorw("purchase not found", "err", err, "txnId", txnId)
	}
	return
}

// 获取google purchase的第一笔信息
func (dao) GetPurchaseHistory(txnId string) (item PurchaseDO, err error) {
	err = conn.GetDB().Get(&item, `select * from purchase where txn_id=?`, txnId)
	if err != nil {
		server.DaoLogger.Errorw("now purchase not found", "err", err, "txnId", txnId)
		return
	}

	// 查询历史订单 (索引考虑优化)
	err = conn.GetDB().Get(&item, `select * from purchase where user_id=? limit 1`, item.UserId)
	if err != nil {
		server.DaoLogger.Errorw("first purchase not found", "err", err, "userId", item.UserId)
		return
	}

	return
}

// 获取purchase_sub信息
func (dao) GetPurchaseSub(originalId string) (item PurchaseSubsDO) {
	err := conn.GetDB().Get(&item, `select * from purchase_subs where original_id=?`, originalId)
	if err != nil {
		server.DaoLogger.Errorw("purchase_sub not found", "err", err, "original_id", originalId)
	}
	return
}

func (dao) UpdateUserSubs(userId int64, info PurchaseDO) (err error) {
	// 同时更新下用户身上信息
	_, err = conn.GetDB().Exec(`update user set subs_expires_at=?, subs_pkg_id=? where id=?`, info.GmtExpire, info.PkgId, userId)
	if err != nil {
		server.DaoLogger.Errorw("update subs", "err", err, "userId", userId)
	}
	return
}

func (d dao) PutPurchase(items []PurchaseDO) (latest PurchaseDO, err error) {
	latest = items[0]

	_, err = conn.GetDB().NamedExec(`insert ignore into purchase (user_id, pkg_id, txn_id, gmt_create, gmt_expire, gmt_refund, env, platform)
    values (:user_id, :pkg_id, :txn_id, :gmt_create, :gmt_expire, :gmt_refund, :env, :platform)`, items)
	if err != nil {
		server.DaoLogger.Errorw("put purchase", "err", err)
		return
	}
	err = d.UpdateUserSubs(items[0].UserId, latest)
	return
}

func (dao) PutPurchaseSubs(item PurchaseSubsDO) {
	_, err := conn.GetDB().Exec(`INSERT INTO purchase_subs (original_id, pkg_id, periods, gmt_create, gmt_latest, gmt_cancel, platform)
		VALUE (?,?,?,?,?,?,?) on duplicate key update periods=?, gmt_latest=?, gmt_cancel=?`,
		item.OriginalId, item.PkgId, item.Periods, item.GmtCreate, item.GmtLatest, item.GmtCancel, item.Platform,
		item.Periods, item.GmtLatest, item.GmtCancel)
	if err != nil {
		server.DaoLogger.Errorw("put purchase subs", "err", err)
	}
}
