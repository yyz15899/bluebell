package controller

import (
	logic "web_app/02-logic"
	models "web_app/05-models"
	pkg "web_app/07-pkg"

	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
)

func GetChat(c *gin.Context) { // c内部自带w,r
	jwt := c.Query("token")
	claims, err := logic.VerifyToken(jwt) // 判断token是否合法
	if err != nil {
		pkg.ResponseError(c, pkg.CodeInvalidToken)
		return
	}
	// 获取uid
	uid := claims.UserId
	// -----------------------------------------------------------------------------
	// 建立连接
	// 升级为websocket
	con, err := websocket.Accept(c.Writer, c.Request, nil)
	if err != nil {
		pkg.ResponseError(c, pkg.CodeUpgradefailed)
		return
	}
	defer con.CloseNow()
	username := claims.UserName
	// 填充client数据
	client := &models.Client{
		Conn:     con, // Accept 返回的那条连接,代表这个用户
		UID:      uid, // 刚从token里解析出来的用户身份
		Username: username,
		Send:     make(chan []byte, 256), // 新建一个缓冲256的信道
		Hub:      models.GlobalHub,
	}
	// 注册client
	models.GlobalHub.Register(client)
	// 将name,id,time 写入messages结构体
	// 写泵
	go client.WritePump()
	// 读泵
	client.ReadPump()
}
