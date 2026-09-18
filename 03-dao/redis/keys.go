package redis

import (
	"errors"
	"strconv"
)

const (
	keyPostTimeZSet     = "post:time"     // 发帖时间
	keyPostScoreZSet    = "post:score"    // 票数榜(票数分)
	keyPostVotedZSetPre = "post:voted:"   // ① 投票记录,后拼帖子ID
	keyVoteNumZSet      = "post:vote_num" // ② 总票数,key固定,不用拼
	keyPostHacker       = "post:hotscore" // 热榜接口(算HNScore)
)

var (
	ErrPostTimeNotFound = errors.New("post time record not found in redis")
	ErrPostVoteNotFound = errors.New("post vote record not found in redis")
)

// 获取reids中的完整key		post:voted:postid	记录用户是否投过票
func GetKeyForPostVote(postid int64) string {
	key := keyPostVotedZSetPre + strconv.FormatInt(postid, 10)
	return key
}

// post:vote_num	记录投票数
func GetKeyForVoteNum() string {
	return keyVoteNumZSet
}

// post:time		记录post的创建时间
func GetKeyForPostTime() string {
	return keyPostTimeZSet
}

func GetKeyForPostScore() string {
	return keyPostHacker
}
