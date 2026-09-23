package models

type Message struct {
	UID      int64  `json:"uid"`
	Username string `json:"username"`  // JWT claims 里有,不用查库
	Content  string `json:"msg"`       // 用户发来的正文
	SendTime int64  `json:"send_time"` // time.Now().Unix()
}
