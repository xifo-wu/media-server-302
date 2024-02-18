package alist

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"strconv"

	"github.com/spf13/viper"
)

func Sign(data string, expire int64) string {
	h := hmac.New(sha256.New, []byte(viper.GetString("alist.token")))

	expireTimeStamp := strconv.FormatInt(expire, 10)
	_, err := io.WriteString(h, data+":"+expireTimeStamp)
	if err != nil {
		return ""
	}

	return base64.URLEncoding.EncodeToString(h.Sum(nil)) + ":" + expireTimeStamp
}
