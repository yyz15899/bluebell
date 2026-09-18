package jwt

import (
	"errors"
	"time" // 注意导入 time 包

	"github.com/golang-jwt/jwt/v5"
)

// 定义密钥（实际项目中建议从配置文件获取）
var mySecret = []byte("夏天夏天")

const TokenExpireDuration = time.Hour * 2 // token 有效期, 登录存 redis 也用这个

type MyClaims struct { // 定义 token 想要存储(认证)的信息
	UserId               int64  `json:"userid"`
	UserName             string `json:"username"`
	jwt.RegisteredClaims        // 官方标准字段（如过期时间、签发人等）
}

// GenToken 生成 JWT Token
func GenToken(userid int64, username string) (string, error) {
	// 实例化 Claims（建议显式指定所有字段名）
	claims := MyClaims{
		UserId:   userid,
		UserName: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 2)), // 注意 Add 的 A 大写
			Issuer:    "my-app",                                          // 签发人
			IssuedAt:  jwt.NewNumericDate(time.Now()),                    // 签发时间
		},
	}
	// 使用指定签名方法创建获取 token 对象
	// 使用 HS256 加密claims 数据
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// 使用密钥签名并获得完整的编码后的字符串 Token
	return token.SignedString(mySecret)
}

// ParseToken 解析jwt Token
func ParseToken(tokenString string) (*MyClaims, error) {
	// 解析 Token
	token, err := jwt.ParseWithClaims(tokenString, &MyClaims{}, func(token *jwt.Token) (interface{}, error) {
		return mySecret, nil
	})
	if err != nil {
		return nil, err
	}

	// 校验 Claims 并返回数据(既校验了数据格式（类型断言），也校验了 Token 的合法性(valid)与有效状态(时间是否生效/过期与是否被篡改过)。)
	if claims, ok := token.Claims.(*MyClaims); ok && token.Valid { // 解析发生时,库把你传进去的那个指针对象"就地填满"了数据，然后同一个对象被塞进 token.Claims。
		// token.Claims 实际类型是一个接口(为了能容纳任意自定义 claims 结构体),运行时的真实值就是你的 *MyClaims。类型断言就是告诉编译器："别把它当通用接口了，它实际上是我的 *MyClaims，让我取里面的字段。"
		return claims, nil // 数据正常
	}
	// ok 确保了：数据成功解构到了你的 MyClaims 结构体中（格式正确）。token.Valid 确保了：这个 Token 没有被伪造，且没有过期（状态合法）。

	return nil, errors.New("invalid token")
}
