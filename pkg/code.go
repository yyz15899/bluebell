package pkg

type ResCode int64 // 定义状态码类型

const (
	CodeSuccess ResCode = 1000 + iota // 这里的iota在数组每一层累计加1  该层为 1000
	// 后续默认全为ResCode 类型
	CodeInvalidParam      // 1001
	CodeUserExist         //1002
	CodeUserNotExist      //1003
	CodeInvalidPassword   //1004
	CodeServerBusy        //1005
	CodeInvalidToken      // 1006
	CodeNeedLogin         // 1007
	CodeCommunityNotExist // 1008
)

// 封装错误码
var codeMsgMap = map[ResCode]string{ // map[KeyType]ValueType	map[key类型]参数类型
	CodeSuccess:           "success",
	CodeInvalidParam:      "请求参数错误",
	CodeUserExist:         "用户名已存在",
	CodeUserNotExist:      "用户名不存在",
	CodeInvalidPassword:   "用户名或密码错误",
	CodeServerBusy:        "服务繁忙",
	CodeInvalidToken:      "无效的token",
	CodeNeedLogin:         "请先登录",
	CodeCommunityNotExist: "未找到相关社区",
}

func (c ResCode) Msg() string {
	msg, ok := codeMsgMap[c]
	if !ok {
		msg = codeMsgMap[CodeServerBusy]
	}
	return msg
}
