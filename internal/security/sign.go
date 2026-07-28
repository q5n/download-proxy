package security


import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)



func Sign(
	url string,
	expire int64,
	secret string,
) string {


	data:=fmt.Sprintf(
		"url=%s&expire=%d",
		url,
		expire,
	)


	h:=hmac.New(
		sha256.New,
		[]byte(secret),
	)


	h.Write(
		[]byte(data),
	)


	return hex.EncodeToString(
		h.Sum(nil),
	)
}



func Verify(
	url string,
	expire int64,
	sign string,
	secret string,
	maxExpire int64,
) bool {


	now:=time.Now().Unix()


	// 已过期
	if expire < now {
		return false
	}


	// 最大有效时间限制
	if expire-now > maxExpire {
		return false
	}


	expected:=Sign(
		url,
		expire,
		secret,
	)


	return hmac.Equal(
		[]byte(expected),
		[]byte(sign),
	)
}