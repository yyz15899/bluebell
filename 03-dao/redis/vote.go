package redis

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// 使用pipline将两步绑定执行
func DoVote(postid int64, userid int64, direction int64, delta int64) error {
	ctx := context.Background()
	pipeline := RDB.TxPipeline()
	value := redis.Z{
		Score:  float64(direction),
		Member: strconv.FormatInt(userid, 10),
	}
	pipeline.ZAdd(ctx, GetKeyForPostVote(postid), value)
	pipeline.ZIncrBy(ctx, GetKeyForVoteNum(), float64(delta), strconv.FormatInt(postid, 10))

	_, err := pipeline.Exec(ctx)
	if err != nil {
		return err
	}
	return nil
}

// Check 返回该用户对该帖子的投票方向(like):1赞 -1踩 0没投过
func CheckLike(postid int64, userid int64) (int64, error) {
	ctx := context.Background()
	like, err := RDB.ZScore(ctx, GetKeyForPostVote(postid), strconv.FormatInt(userid, 10)).Result() // 查看当前userid的用户的分数(票数)
	// like存的是0 1 -1 表示投票方向
	if errors.Is(err, redis.Nil) { // 没投过(like=0)
		return 0, nil
	}
	if err != nil { // 本体错误
		return 0, err
	}
	return int64(like), nil
}

// post的7天窗口期
// 记录post创建时间
func SetPostTime(postid int64, createTime time.Time) error {
	ctx := context.Background()
	value := redis.Z{
		Score:  float64(createTime.Unix()),
		Member: strconv.FormatInt(postid, 10),
	}
	_, err := RDB.ZAdd(ctx, GetKeyForPostTime(), value).Result() // 存入创建时间
	if err != nil {
		return err
	}
	return nil
}

// 查看创建post的时间戳
func GetPostTime(postid int64) (time.Time, error) {
	ctx := context.Background()
	score, err := RDB.ZScore(ctx, GetKeyForPostTime(), strconv.FormatInt(postid, 10)).Result()
	if errors.Is(err, redis.Nil) {
		// return time.Time{}, nil // ← 帖子没有时间记录时,返回"零值时间(0001-01-01 00:00:00)"和"无错误",
		// 但实际上这种情况应该算作异常,否则当新post发布时,若Redis恰好抖动了一下导致数据没写入,但logic层依旧接收到超长时间,但其实post是正常的
		// 考虑将nil也当成异常上抛
		return time.Time{}, ErrPostTimeNotFound // 或专门的"无投票资格"错误码
	}
	if err != nil { // 本体错误
		return time.Time{}, err
	}
	return time.Unix(int64(score), 0), nil
}

// 同步post得票数据到mysql中
func GetAllPostVoteNum() (map[int64]int64, error) {
	ctx := context.Background()
	res, err := RDB.ZRangeWithScores(ctx, GetKeyForVoteNum(), 0, -1).Result()
	if err != nil { // 查询不到时会返回[],不用判断redis.Nil
		return nil, err
	}
	votes := make(map[int64]int64, len(res)) // 创建一个字典,并提前开好空间
	for _, z := range res {                  // 遍历postid - vote  zset结构,将member写入字典的key
		postID, err := strconv.ParseInt(z.Member.(string), 10, 64)
		if err != nil {
			continue
		}
		votes[postID] = int64(z.Score) // 给每个key(postid)赋值value(vote)
	}
	return votes, nil
}

// dao/redis: 发帖时初始化,两条一起(防止key: post:vote_num 中没有值)
func CreatePostRecord(postid int64, createTime time.Time) error {
	ctx := context.Background()
	pid := strconv.FormatInt(postid, 10)
	pipe := RDB.TxPipeline()
	pipe.ZAdd(ctx, GetKeyForPostTime(), redis.Z{Score: float64(createTime.Unix()), Member: pid})
	pipe.ZAdd(ctx, GetKeyForVoteNum(), redis.Z{Score: 0, Member: pid}) // 票数从0起
	_, err := pipe.Exec(ctx)
	return err
}

// 热榜
// 返回所有post时间
func GetAllPostTimes() (map[int64]time.Time, error) {
	ctx := context.Background()
	score, err := RDB.ZRangeWithScores(ctx, GetKeyForPostTime(), 0, -1).Result() // 按照创建时间排序
	if err != nil {                                                              // 查询不到时会返回[],不用判断redis.Nil
		return nil, err
	}
	pid_time := make(map[int64]time.Time)
	for _, t := range score {
		postID, err := strconv.ParseInt(t.Member.(string), 10, 64)
		if err != nil {
			continue
		}
		sec := int64(t.Score)
		nsec := int64((t.Score - float64(sec)) * 1e9)
		pid_time[postID] = time.Unix(sec, nsec)
	}
	return pid_time, nil
}

// 存入所有帖子的HN得分
func SetPostHNScores(scores map[int64]float64) error {
	ctx := context.Background()
	zs := make([]redis.Z, 0, len(scores))
	for pid, score := range scores {
		zs = append(zs, redis.Z{Score: score, Member: strconv.FormatInt(pid, 10)})
	}
	_, err := RDB.ZAdd(ctx, GetKeyForPostScore(), zs...).Result()
	if err != nil {
		return err
	}
	return nil

}

// 基于Hacker News(简称HN) 算法:score = (votes - 1) / (age_hours + 2) ^ gravity    # gravity 通常取 1.5
func SetPostIDbyHN(postid int64, hotscore float64) error {
	ctx := context.Background()
	value := redis.Z{
		Member: strconv.FormatInt(postid, 10),
		Score:  float64(hotscore),
	}
	_, err := RDB.ZAdd(ctx, GetKeyForPostScore(), value).Result()
	// 这里选择zadd直接覆盖原score  防止失真(天文数字)
	if err != nil {
		return err
	}
	return nil
}

// 获取redis排行榜数据(记录分页)
func GetPostIDByHot(page, size int64) ([]int64, error) { // 只需要返回作为结果的pid,以后需要将分数展示时,考虑在返回[]int64 记录score
	ctx := context.Background()
	start, stop := (page-1)*size, page*size-1
	res, err := RDB.ZRevRangeWithScores(ctx, GetKeyForPostScore(), start, stop).Result() // 采用倒序
	if err != nil {
		return nil, err // 最后一页之后为[]是正常现象
	}
	slic_pid := make([]int64, 0, len(res))
	for _, p := range res {
		pid, err := strconv.ParseInt(p.Member.(string), 10, 64)
		if err != nil {
			continue // 脏member跳过,和GetAllPostTimes保持一致
		}
		slic_pid = append(slic_pid, pid)

	}
	return slic_pid, nil
}

// 写一个txpipeline来删除key
func DeletePostRecord(postid int64) error {
	ctx := context.Background()
	pid := strconv.FormatInt(postid, 10)
	pipe := RDB.TxPipeline()
	pipe.ZRem(ctx, GetKeyForPostTime(), pid)
	pipe.ZRem(ctx, GetKeyForVoteNum(), pid)
	pipe.Del(ctx, GetKeyForPostVote(postid))
	_, err := pipe.Exec(ctx)
	return err
}
