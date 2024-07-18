package message_bus

import (
	"time"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/scys-devs/lib-go/conn"
	"github.com/scys-devs/lib-go/server"
)

//var dao MessageDAO

const ESMessageBus = "message_bus"
const ESMessageBusTest = "message_bus_dev"

type MessageDAO interface {
	Put(m DO) interface{}
	CountInPeriod(periodStart, periodEnd, userId int64, groupId string) (count int)
	CountAll(userId int64, groupId string) (count int)
}

type mysqlDao struct {
}

func (mysqlDao) Put(m DO) interface{} {
	res, _ := conn.GetDB().Exec(`insert into message_bus (user_id, group_id, gmt_create, sent) values (?,?,?,?)`,
		m.UserId, m.GroupID(), time.Now().Unix(), m.Sent)
	id, _ := res.LastInsertId()
	return id
}

func (mysqlDao) CountInPeriod(periodStart, periodEnd, userId int64, groupId string) (count int) {
	_ = conn.GetDB().Get(&count, "SELECT COUNT(*) FROM message_bus WHERE user_id=? AND group_id=? AND sent=1 AND gmt_create BETWEEN ? AND ?",
		userId, groupId, periodStart, periodEnd)
	return
}

func (mysqlDao) CountAll(userId int64, groupId string) (count int) {
	_ = conn.GetDB().Get(&count, "SELECT COUNT(*) FROM message_bus WHERE user_id=? AND group_id=? AND sent=1",
		userId, groupId)
	return
}

type ESDao struct {
	Index string
}

func GetESDao() ESDao {
	if len(conn.ENV) > 0 {
		// 测试环境和线上分开吧
		return ESDao{ESMessageBusTest}
	} else {
		return ESDao{ESMessageBus}
	}
}

func (dao ESDao) Put(m DO) interface{} {
	var b = conn.NewESBody()
	b.Write(nil, m.ToEsDO())

	_ = conn.ESPut(dao.Index, b.Body)
	return ""
}

func (dao ESDao) BatchPut(ll []DO) error {
	var b = conn.NewESBody()
	for _, item := range ll {
		b.Write(nil, item.ToEsDO())
	}
	return conn.ESPut(dao.Index, b.Body)
}

func (dao ESDao) CountInPeriod(periodStart, periodEnd, userId int64, groupId string) (count int) {
	req := gin.H{
		"query": gin.H{
			"bool": gin.H{
				"filter": []gin.H{
					{"term": gin.H{"user_id": userId}},
					{"term": gin.H{"group_id": groupId}},
					{
						"range": gin.H{
							"gmt_create": gin.H{
								"gte": periodStart,
								"lte": periodEnd,
							},
						},
					},
				},
			},
		},
	}
	countRaw, _ := conn.ESCount(dao.Index, req)
	return jsoniter.Get(countRaw, "count").ToInt()
}

func (dao ESDao) CountAll(userId int64, groupId string) (count int) {
	req := gin.H{
		"query": gin.H{
			"bool": gin.H{
				"filter": []gin.H{
					{"term": gin.H{"user_id": userId}},
					{"term": gin.H{"group_id": groupId}},
				},
			},
		},
	}
	countRaw, _ := conn.ESCount(dao.Index, req)
	return jsoniter.Get(countRaw, "count").ToInt()
}

// IsLimit 是否限制
func IsLimit(m DO, dao MessageDAO) bool {
	// 判断限制次数
	if m.PeriodLimit.Limit <= 0 {
		return false
	}

	// 必要参数没有就限制吧
	if m.UserId == 0 || len(m.GroupID()) == 0 {
		return true
	}

	var cnt int
	if m.PeriodLimit.Period == -1 { // 整个周期限制次数
		cnt = dao.CountAll(m.UserId, m.GroupID())
	} else {
		start := (server.Scheduler.Now().Unix()+m.PeriodLimit.Phase)/m.PeriodLimit.Period*m.PeriodLimit.Period - m.PeriodLimit.Phase
		end := start + m.PeriodLimit.Period
		cnt = dao.CountInPeriod(start, end, m.UserId, m.GroupID())
	}

	if cnt >= m.PeriodLimit.Limit {
		return true
	}
	return false
}
