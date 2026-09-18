package snowflake

import (
	"time"

	"github.com/bwmarrin/snowflake"
)

// 利用雪花算法生成user_id
var node *snowflake.Node

func Init(startTime string, machineID int64) (err error) {

	var st time.Time
	st, err = time.Parse("2006-01-02", startTime)
	if err != nil {
		return
	}

	snowflake.Epoch = st.UnixNano() / 1000000
	node, err = snowflake.NewNode(machineID)
	return
}

func GenID() int64 {
	// Generate a snowflake ID.
	id := node.Generate().Int64()
	return id
}

// func main() {
// 	if err := Init("2026-09-01", 1); err != nil {
// 		fmt.Printf("init failed,err:%v\n", err)
// 		return
// 	}
// 	id := GenID()
// 	fmt.Println(id)
// }
