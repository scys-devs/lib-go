package dash

import (
	"fmt"
	"strings"

	"github.com/scys-devs/lib-go"
)

type TablePagination struct {
	// 翻页
	Page     int    `form:"page" json:"page,omitempty"`
	PerPage  int    `form:"perPage" json:"perPage,omitempty"`
	OrderBy  string `form:"orderBy" json:"orderBy,omitempty"`
	OrderDir string `form:"orderDir" json:"orderDir,omitempty"`
	// 搜索
	Fields string            `from:"fields" json:"fields,omitempty"` // 自定义显示字段 为空走cookie
	Form   map[string]string `form:"form" json:"form,omitempty"`     // 基础的搜索条件
	Query  map[string]string `form:"query" json:"query,omitempty"`   // 复杂的搜索条件
	Cond   Condition         `form:"cond" json:"cond,omitempty"`     // 高级搜索
}

func (do *TablePagination) ToWhere(handler func(key string) (string, []interface{}), orderBy ...string) (string, []interface{}) {
	var whereArr []string
	var args []interface{}

	// 支持根据指定顺序读取参数
	for key := range do.Form {
		if do.Form[key] == "" || do.Form[key] == "0" {
			continue
		}
		if lib.Index(orderBy, key) >= 0 {
			continue
		}
		orderBy = append(orderBy, key)
	}

	for _, key := range orderBy {
		if do.Form[key] == "" || do.Form[key] == "0" {
			continue
		}

		q, a := handler(key)
		if len(q) > 0 {
			whereArr = append(whereArr, q)
			args = append(args, a...)
		}
	}

	if len(whereArr) == 0 {
		return "", nil
	}

	return " where " + strings.Join(whereArr, " and "), args
}

func (do *TablePagination) ToLimit() string {
	if do.PerPage <= 0 { // 全量获取
		return ""
	}
	if do.Page == 0 { // 设置默认值
		do.Page = 1
	}
	return fmt.Sprintf(` limit %v offset %v`, do.PerPage, (do.Page-1)*do.PerPage)
}

func (do *TablePagination) ToCndPage() int {
	if do.Page == 0 {
		return 0
	} else {
		return do.Page - 1
	}
}

func (do *TablePagination) ToExport() {
	do.PerPage = -1
}

func (do *TablePagination) GetPage(total int, get func(index int)) {
	start := (do.Page - 1) * do.PerPage
	for i := 0; i < do.PerPage; i++ {
		index := start + i
		if index >= total {
			break
		}
		get(index)
	}
}

func NewPagination(form map[string]string) *TablePagination {
	return &TablePagination{PerPage: -1, Form: form}
}

// TableDTO 表格输出对象
// https://baidu.gitee.io/amis/zh-CN/components/crud#%E6%95%B0%E6%8D%AE%E6%BA%90%E6%8E%A5%E5%8F%A3%E6%95%B0%E6%8D%AE%E7%BB%93%E6%9E%84%E8%A6%81%E6%B1%82
type TableDTO struct {
	Items   interface{} `json:"items"`
	Total   int         `json:"total,omitempty"`
	HasNext bool        `json:"hasNext,omitempty"`
	Extra   interface{} `json:"extra,omitempty"` // 用于配合一些扩展数据，类似${extra.user[user_id].xq_name}的用法
}

type Option struct {
	Label    string   `json:"label"`
	Value    string   `json:"value"`
	Name     string   `json:"name,omitempty"`
	Children []Option `json:"children,omitempty"`
}

type ConditionLeft struct {
	Type  string `json:"type,omitempty"`
	Field string `json:"field,omitempty"`
}

// Condition 组合条件
type Condition struct {
	Conjunction string        `json:"conjunction,omitempty"` // and or
	Children    []Condition   `json:"children,omitempty"`
	Left        ConditionLeft `json:"left,omitempty"`
	Op          string        `json:"op,omitempty"`
	Right       interface{}   `json:"right,omitempty"` // 需要兼容数字还是字符
}
