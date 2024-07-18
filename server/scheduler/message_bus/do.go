package message_bus

import "time"

type PeriodLimit struct {
	Phase  int64 `json:"phase,omitempty"`  // 偏移量，默认东8区
	Period int64 `json:"period,omitempty"` // 周期
	Limit  int   `json:"limit,omitempty"`  // 限制次数
}

// seconds秒内限制limit次，seconds=-1:整个周期限制limit次
func NewPeriodLimit(period int64, limit int) (limiter PeriodLimit) {
	limiter.Phase = 3600 * 8
	limiter.Period = period
	limiter.Limit = limit
	return
}

func NewLimit(limit int) PeriodLimit {
	return NewPeriodLimit(-1, limit)
}

type DO struct {
	UserId         int64             `db:"user_id" json:"user_id,omitempty"`
	XQGroupNumber  int64             `json:"xq_group_number,omitempty"` // 当做event日志的时提交
	Group          string            `db:"group" json:"group,omitempty"`
	GroupKey       string            `json:"group_key,omitempty"`
	Sent           int               `db:"sent" json:"sent,omitempty"`
	SentErr        string            `json:"sent_err,omitempty"`        // 发送失败的原因
	Data           map[string]string `json:"data,omitempty"`            // 附带的数据
	AppID          string            `json:"app_id,omitempty"`          // 需要发送的app_id
	PeriodLimit    PeriodLimit       `json:"period_limit"`              // 时间限制
	WhiteListLimit bool              `json:"whitelist_limit,omitempty"` // 仅限白名单
	// 临时字段
	CanSent bool `json:"-"`
	// 实际没作用
	GmtCreate int64 `db:"gmt_create" json:"gmt_create,omitempty"`
}

func (do DO) GroupID() string {
	if len(do.GroupKey) > 0 {
		return do.Group + do.GroupKey
	}
	return do.Group
}

func (do DO) ToEsDO() EsDO {
	return EsDO{
		UserId:        do.UserId,
		XQGroupNumber: do.XQGroupNumber,
		GroupID:       do.GroupID(),
		Sent:          do.Sent,
		GmtCreate:     time.Now().Unix(),
	}
}

type EsDO struct {
	UserId        int64  `json:"user_id,omitempty"`
	XQGroupNumber int64  `json:"xq_group_number,omitempty"`
	GroupID       string `json:"group_id"`
	Sent          int    `json:"sent"`
	GmtCreate     int64  `json:"gmt_create"`
}
