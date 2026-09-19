package logic

import (
	"math"
	"strconv"
	"time"
	"web_app/03-dao/mysql"
	"web_app/03-dao/redis"
	models "web_app/05-models"
	pkg "web_app/07-pkg"
	"web_app/07-pkg/snowflake"

	"errors"

	"go.uber.org/zap"
)

func CreatePost(p *models.Post) error {
	// 先校验communityid
	// 用户传入的community_id 必须真正存在
	count, err := mysql.CheckCommunityID(p.CommunityID)
	if err != nil {
		zap.L().Error("sql link err", zap.Error(err))
		return pkg.WrapBizError(pkg.CodeServerBusy, err)
	}
	if !count {
		return pkg.NewBizError(pkg.CodeCommunityNotExist)
	}
	// 生成postid  (对应表中的id)
	p.PostID = snowflake.GenID()
	if err := mysql.CreatePost(p); err != nil {
		zap.L().Error("CreatePost logic failed", zap.Error(err))
		return pkg.WrapBizError(pkg.CodeServerBusy, err)
	}
	// 记录发帖时间到redis中
	if err := redis.CreatePostRecord(p.PostID, p.CreateTime); err != nil {
		zap.L().Error("redis.CreatePostRecord failed", zap.Error(err))
		// redis缓存错误后的数据可重建,这里不能return err 来打断整体流程,打个日志即可
	}
	return nil
}

// GetPost  查询该用户post,并根据author_id 查出 username
func GetPost(pid int64) (*models.ApiPostDetail, error) {
	data, err := mysql.GetPost(pid)
	if err != nil {
		if errors.Is(err, mysql.ErrPostNotExist) {
			return nil, pkg.NewBizError(pkg.CodePostNotExist) // 业务结果:翻译成码,不打日志,防止污染
		} else {
			return nil, pkg.WrapBizError(pkg.CodeServerBusy, err) // 系统故障:包装后上抛,保留错误链
		}
	}
	// 根据author_id  查询author_name
	author_id := data.AuthorID
	user, err := mysql.GetUserByID(author_id)
	if err != nil {
		if errors.Is(err, mysql.ErrUserNotExist) {
			return nil, pkg.NewBizError(pkg.CodeUserNotExist) // 业务结果:翻译成码,不打日志
		} else {
			zap.L().Error("get userbyid failed", zap.Error(err))
			return nil, pkg.WrapBizError(pkg.CodeServerBusy, err) // 系统故障:包装后上抛,保留错误链
		}
	}

	community, err := mysql.GetCommunity(data.CommunityID)
	if err != nil {
		if errors.Is(err, mysql.ErrCommunityNotExist) {
			return nil, pkg.NewBizError(pkg.CodeCommunityNotExist)
		} else {
			zap.L().Error("get community failed", zap.Error(err))
			return nil, pkg.WrapBizError(pkg.CodeServerBusy, err)
		}

	}
	// 组装接口返回数据
	apiPostDetail := &models.ApiPostDetail{
		AuthorName:    user.Username,
		Post:          data,
		CommunityName: strconv.FormatInt(int64(community.CommunityID), 10), // CommunityDetail 里有同名字段
	}
	return apiPostDetail, nil
}

// 分页查询
func GetPostList(p *models.ParamPostList) ([]*models.ApiPostDetail, error) {
	posts, err := mysql.GetPostList(p)
	if err != nil {
		zap.L().Error("get postlist failed", zap.Error(err))
		return nil, pkg.WrapBizError(pkg.CodeServerBusy, err)
	}
	if len(posts) == 0 { // 没有找到post
		return make([]*models.ApiPostDetail, 0), nil
	}
	//
	Info, err := AssemblePostDetails(posts)
	if err != nil {
		zap.L().Error("combine pid ang cid failed", zap.Error(err))
		return nil, pkg.WrapBizError(pkg.CodeServerBusy, err)

	}
	return Info, nil

}

// 按照热度分页查询
// 计算hn分数
// 公式为: score = (votes + 1) / (age_hours + 2) ^ gravity
const hnGravity = 1.5

func RecomputeHNScores() error {
	//获取时间和票数
	times, err := redis.GetAllPostTimes()
	if err != nil {
		zap.L().Error("redis getallposttime failed", zap.Error(err))
		return err
	}
	votes, err := redis.GetAllPostVoteNum()
	if err != nil {
		zap.L().Error("redis getallpostvotenum failed", zap.Error(err))
		return err
	}
	// 逐帖按照公式计算hot分数
	now := time.Now()
	scores := make(map[int64]float64, len(times))
	for pid, CreatTime := range times {
		vote, ok := votes[pid]
		if !ok { // 票数记录缺失:按0算,但要被看见
			zap.L().Warn("vote record missing, treat as 0", zap.Int64("post_id", pid))
		}
		age := now.Sub(CreatTime).Hours() // 返回过去了多久(小时为单位)
		if age < 0 {                      // 时钟回拨等异常,按0岁算,防止负age导致分数爆炸
			age = 0
		}
		scores[pid] = float64(vote+1) / math.Pow(age+2, hnGravity)
	}
	// 将算出的HNscore 写入redis
	if err := redis.SetPostHNScores(scores); err != nil {
		zap.L().Error("redis setposthnscores failed", zap.Error(err))
		return err
	}
	zap.L().Info("recompute hn scores done", zap.Int("count", len(scores)))
	return nil
}

func GetPostListByHot(page, size int64) ([]*models.ApiPostDetail, error) {
	// 查出redis中的score,返回有序的pid
	rank_id, err := redis.GetPostIDByHot(page, size)
	if err != nil {
		zap.L().Error("get postidbyhot failed", zap.Error(err))
		return nil, pkg.WrapBizError(pkg.CodeServerBusy, err)
	}
	// 空列表早退(正常现象)
	if len(rank_id) == 0 {
		return make([]*models.ApiPostDetail, 0), nil // 空页,正常返回
	}
	// 将pid存入mysql
	postlist, err := mysql.GetPostListByIDs(rank_id)
	// 拿出来的postlist中的pid已经不是按照先前有序排列的了,需要恢复顺序
	if err != nil {
		return nil, pkg.WrapBizError(pkg.CodeServerBusy, err)
	}
	// 将pid 与 post存入map
	map_pidpost := make(map[int64]*models.Post)
	for _, p := range postlist {
		map_pidpost[p.PostID] = p
	}
	// 按照redis的有序pid遍历
	orderedposts := make([]*models.Post, 0, len(rank_id))
	for _, p := range rank_id {
		post, ok := map_pidpost[p]
		if !ok { // mysql里已不存在(脏member),跳过
			continue
		}
		orderedposts = append(orderedposts, post) // 拿到有序地post模型切片
	}

	//
	details, err := AssemblePostDetails(orderedposts)
	if err != nil {
		zap.L().Error("combine post details failed", zap.Error(err))
		return nil, pkg.WrapBizError(pkg.CodeServerBusy, err)
	}
	return details, nil

}

func AssemblePostDetails(posts []*models.Post) ([]*models.ApiPostDetail, error) {
	// 收集所有authorid 然后批量从数据库里取出username(查user)
	idset := make(map[int64]struct{}) // 证明一个key为int64  value为空结构体的集合
	cidset := make(map[int64]struct{})
	for _, p := range posts { // 遍历posts 每一个元素,并去重(key)添加到struct中
		idset[p.AuthorID] = struct{}{} // 空结构体实例,向字典 idset 中放入一个键（Key,并给它配一个“没有任何实际内容”的值（Value）
		//相当于单独设置key 先将value置空
		cidset[p.CommunityID] = struct{}{} //同理,将communityid存入map
	}
	// 这里community_id 可以是任何值(用户传的),但实际上db中可能没有这个communityid
	//存储authorid到一个切片中
	ids := make([]int64, 0, len(idset)) // 创建长度为0,预分配空间为len(idset)的int64切片
	cids := make([]int64, 0, len(cidset))
	for id := range idset { //单变量遍历 map 时，拿到的是 key（键),不是 value
		ids = append(ids, id) // 将authorid 加到ids中
	}
	for cid := range cidset {
		cids = append(cids, cid)
	}

	//到数据库中根据authorid拿到user
	userlist, err := mysql.GetUserByIDs(ids)
	if err != nil {
		zap.L().Error("get userbyids failed", zap.Error(err))
		return nil, pkg.WrapBizError(pkg.CodeServerBusy, err)
	}
	// 根据community拿到community信息
	communitylist, err := mysql.GetCommunityByid(cids) // 想起187行的注释,实际上post与community(并集)都存在的cid是在sql中识别出的
	if err != nil {
		zap.L().Error("get communitybyid failed", zap.Error(err))
		return nil, pkg.WrapBizError(pkg.CodeServerBusy, err)
	}

	//映射userid---username
	userMap := make(map[int64]string, len(userlist))
	for _, u := range userlist {
		userMap[u.UserID] = u.Username // 将userid与username对应
	}
	// 映射communityid与community
	communityMap := make(map[int64]*models.Community, len(communitylist))
	for i := range communitylist {
		communityMap[int64(communitylist[i].CommunityID)] = &communitylist[i]
	}

	// 组装
	data := make([]*models.ApiPostDetail, 0, len(posts))
	for _, p := range posts {
		communityName := ""
		if c, ok := communityMap[p.CommunityID]; ok && c != nil { //ok 检查的是"map 里有没有这个键"
			communityName = c.CommunityName
		}
		data = append(data, &models.ApiPostDetail{
			AuthorName:    userMap[p.AuthorID],
			Post:          p,
			CommunityName: communityName,
		})
	}
	return data, nil
}

func PostLike(p *models.PostLikeData, uid int64) error {
	// 确保用户不会取消点赞先前没有点赞的post
	if p.Direction != models.PostLike && p.Direction != models.PostUnLike {
		return pkg.NewBizError(pkg.CodeVoteDirectionInvalid)
	}
	// TODO: 帖子存在 + 7天窗口期检查
	creat_time, err := redis.GetPostTime(p.PostID)
	switch {
	case errors.Is(err, redis.ErrPostTimeNotFound): // dao层的redis抖动也算作了系统错误
		zap.L().Error("redis time abnormal", zap.Error(err), zap.Int64("post_id", p.PostID))
		return pkg.WrapBizError(pkg.CodeServerBusy, err)
	case err != nil: // redis 本体错误
		zap.L().Error("get posttime failed", zap.Error(err), zap.Int64("post_id", p.PostID))
		return pkg.WrapBizError(pkg.CodeServerBusy, err)
	}

	if time.Since(creat_time) > 7*24*time.Hour { // 是否过了7天
		return pkg.NewBizError(pkg.CodeVoteTimeExpired)
	}

	// 查询之前是否点过赞
	direction, err := redis.CheckLike(p.PostID, uid)
	if err != nil { // redis本体err
		zap.L().Error("check like failed", zap.Error(err), zap.Int64("post_id", p.PostID))
		return pkg.WrapBizError(pkg.CodeServerBusy, err)
	}
	if direction == p.Direction {
		if p.Direction == models.PostLike {
			return pkg.NewBizError(pkg.CodeLikeRepeat)
		}
		return pkg.NewBizError(pkg.CodeUnLikeRepeat)
	}

	// ZAdd/ZIncrBy 返回的 err 只可能是 Redis 本身故障
	delta := p.Direction - direction
	if err := redis.DoVote(p.PostID, uid, p.Direction, delta); err != nil {
		zap.L().Error("redis dolike failed", zap.Error(err), zap.Int64("post_id", p.PostID))
		return pkg.WrapBizError(pkg.CodeServerBusy, err)
	}
	return nil

}

// 同步postid - votenum 到mysql
func SyncPostVoteNumSQL() {
	// 获取当前post对应的vote
	map_postvote, err := redis.GetAllPostVoteNum()
	if err != nil {
		zap.L().Error("get postvote failed", zap.Error(err))
		return
	}
	if len(map_postvote) == 0 { // 空数据直接跳过,别空转一个事务
		return
	}
	//
	// 将票数存入mysql(直接以map形式存入)
	if err := mysql.UpdatePostVoteNums(map_postvote); err != nil {
		zap.L().Error("Update postvotenums Transaction failed,waiting rollback", zap.Error(err))
		return
	}
	zap.L().Info("sync post vote nums done", zap.Int("count", len(map_postvote)))
}

// 删除post
func DeletePost(pid int64, uid int64) error {
	// 从数据库中填入数据
	post, err := mysql.GetPost(pid)
	if err != nil {
		if errors.Is(err, mysql.ErrPostNotExist) {
			return pkg.NewBizError(pkg.CodePostNotExist)
		}
		zap.L().Error("get post failed", zap.Error(err), zap.Int64("post_id", pid))
		return pkg.WrapBizError(pkg.CodeServerBusy, err)
	}
	// 校验身份(userid 与 authorid)
	if post.AuthorID != uid {
		zap.L().Warn("你不是这篇帖子作者") // 业务错误,按照规范可以不打日志
		return pkg.NewBizError(pkg.CodePermissionDenied)
	}

	if err := mysql.DeletePost(pid); err != nil {
		if errors.Is(err, mysql.ErrPostNotExist) {
			return pkg.NewBizError(pkg.CodePostNotExist)
		}
		zap.L().Error("delete post failed", zap.Error(err), zap.Int64("post_id", pid))
		return pkg.WrapBizError(pkg.CodeServerBusy, err)
	}
	if err := redis.DeletePostRecord(pid); err != nil {
		zap.L().Error("delete post failed", zap.Error(err), zap.Int64("post_id", pid))
		// Redis清理可容忍:热榜会跳过已删帖,不因缓存失败否定已完成的删除
	}
	return nil
}

func UpdatePost(uid int64, pid int64, p *models.ParamUpdatePost) error {
	// 获取authorid
	post, err := mysql.GetPost(pid)
	if err != nil {
		if errors.Is(err, mysql.ErrPostNotExist) {
			return pkg.NewBizError(pkg.CodePostNotExist)
		}
		zap.L().Error("db link err", zap.Error(err))
		return pkg.WrapBizError(pkg.CodeServerBusy, err)
	}
	// 校验身份(authorid与uid)
	if uid != post.AuthorID {
		zap.L().Warn("update post: not the author",
			zap.Int64("post_id", pid), zap.Int64("uid", uid))
		return pkg.NewBizError(pkg.CodePermissionDenied)
	}
	// 先校验更改后的社区是否存在
	if p.CommunityID != nil {
		ok, err := mysql.CheckCommunityID(*p.CommunityID)
		if err != nil {
			zap.L().Error("check community failed", zap.Error(err))
			return pkg.WrapBizError(pkg.CodeServerBusy, err)
		}
		if !ok {
			return pkg.NewBizError(pkg.CodeCommunityNotExist)
		}
	}

	// 更新数据
	if err := mysql.UpdatePost(pid, p); err != nil {
		zap.L().Error("update post failed", zap.Error(err), zap.Int64("post_id", pid))
		return pkg.WrapBizError(pkg.CodeServerBusy, err)
	}
	return nil

}
