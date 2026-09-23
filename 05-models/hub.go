package models

import (
	"context"
	"encoding/json"
	"time"

	"github.com/coder/websocket"
)

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
}

type Client struct {
	Hub      *Hub //凡是有状态的对象，字段和参数一律传指针
	Conn     *websocket.Conn
	UID      int64
	Send     chan []byte
	Username string
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

var GlobalHub = NewHub() // 拿到返回的函数值

// 这里使用全局变量封装的好处：
// 全局唯一性(建立client之间的联系)		也利于跨包调用(也基于唯一性)		包级别的初始化(保证在处理任何请求前map和chan已经make完毕(快于main函数初始化),防止nil导致的Panic或死锁,且天生线程安全)

func (h *Hub) Run() { // map 只有 Run 一个 goroutine 能碰
	for {
		select {
		case c := <-h.register:
			h.clients[c] = true
		case c := <-h.unregister: // 退出的客户(c)
			if _, ok := h.clients[c]; ok { // ok为true说明这个客户存在,执行提出逻辑(直接删除key-value)
				delete(h.clients, c)
				close(c.Send)
			}
		case msg := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.Send <- msg: // 该客户的信箱有空位,投递信息
				default:
					delete(h.clients, client)
					close(client.Send)
				}
			}
		}
	}

}
func (h *Hub) Register(c *Client) {
	h.register <- c
}

func (h *Hub) UnRegister(c *Client) {
	// if !h.clients[c] {
	// 	return
	// }
	h.unregister <- c
}

// 广播
func (h *Hub) Broadcast(msg []byte) {
	h.broadcast <- msg
}

// 写泵
func (c *Client) WritePump() { // 绑定Client结构体
	ctx := context.Background() //gin的context在HTTP Handle后就会被回收,但WritePump要活到连接断开,因此要自己造ctx
	pingticker := time.NewTicker(30 * time.Second)
	defer func() {
		pingticker.Stop()                                   //停止计时,释放资源
		_ = c.Conn.Close(websocket.StatusNormalClosure, "") // 兜底关闭底层连接
	}()

	for {
		select {
		case msg, ok := <-c.Send:
			if !ok {
				return
			}
			pctx, cancel := context.WithTimeout(ctx, 15*time.Second)
			err := c.Conn.Write(pctx, websocket.MessageText, msg)
			cancel()
			if err != nil {
				return
			}
		case <-pingticker.C:
			pctx, cancel := context.WithTimeout(ctx, 15*time.Second) // 一次等15S  若15s内客户端还没返回pong  则取消本次操作                                                 //
			err := c.Conn.Ping(pctx)
			cancel() // 15s 已到 释放资源
			if err != nil {
				return
			}
		}
	}
}

// 读泵
func (c *Client) ReadPump() {
	ctx := context.Background()

	defer func() {
		c.Hub.UnRegister(c)                                 // 踢掉客户(已经关闭连接的客户)
		_ = c.Conn.Close(websocket.StatusNormalClosure, "") // 兜底关闭底层连接
	}()

	for {
		_, msg, err := c.Conn.Read(ctx)
		if err != nil {
			return
		}
		// 将byte[]与id,name,time存到结构体中
		var rep Message
		if err := json.Unmarshal(msg, &rep); err != nil {
			continue
		} // 这里用户收到的其余三个参数为0,"","",且用户可以直接在msg中伪造name,id,sendtime,例如 wsA.send('{"uid":1,"username":"admin","msg":"我是管理员"}')
		// 但下面三行可以直接赋值覆盖掉	(以后再改进)
		rep.UID = c.UID
		rep.Username = c.Username
		rep.SendTime = time.Now().Unix()

		// 将结构体转为byte[],为了后续广播
		msg, err = json.Marshal(&rep)
		if err != nil {
			continue
		}
		c.Hub.Broadcast(msg)
	}
}
