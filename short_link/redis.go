package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/astaxie/goredis"
	"io"
	"time"
)

const (
	UrlIdKey           = "next.url.id"
	ShortLinkKey       = "shortLink:%s:url"
	UrlHashKey         = "urlHash:%s:url"
	ShortLinkDetailKey = "shortLink:%s:detail"
)

type RedisCli struct {
	Cli *goredis.Client
}

func GetMD5(lurl string) string {
	h := md5.New()
	salt1 := "salt4shorturl"
	io.WriteString(h, lurl+salt1)
	urlmd5 := fmt.Sprintf("%x", h.Sum(nil))
	return urlmd5
}

func getRange(start, end rune) (ran []rune) {
	for i := start; i <= end; i++ {
		ran = append(ran, i)
	}
	return ran
}

func merge(a, b []rune) []rune {
	c := make([]rune, len(a)+len(b))
	copy(c, a)
	copy(c[len(a):], b)
	return c
}

func Generate(num int64) (tiny string) {
	fmt.Println(num)
	num += 100000000
	alpha := merge(getRange(48, 57), getRange(65, 90))
	alpha = merge(alpha, getRange(97, 122))
	if num < 62 {
		tiny = string(alpha[num])
		return tiny
	} else {
		var runes []rune
		runes = append(runes, alpha[num%62])
		num = num / 62
		for num >= 1 {
			if num < 62 {
				runes = append(runes, alpha[num-1])
			} else {
				runes = append(runes, alpha[num%62])
			}
			num = num / 62

		}
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		tiny = string(runes)
		return tiny
	}
	return tiny
}
func (r *RedisCli) Shorten(url string, exp int64) (string, error) {
	urlHash := GetMD5(url)
	d, err := r.Cli.Get(fmt.Sprintf(UrlHashKey, urlHash))
	//有可能不存在
	if err != nil {
		//return "", err
	} else {
		if string(d) == "{}" {
			//expiration,nothing to do, generate new
		} else {
			return string(d), nil
		}
	}
	id, err := r.Cli.Incr(UrlIdKey)
	if err != nil {
		return "", err
	}
	//id, err := r.Cli.Get(UrlIdKey).Int64()
	//if err != nil {
	//	return "", err
	//}
	//shortLink := base62.EncodeInt64(id)
	shortLink := Generate(id)
	err = r.Cli.Setex(fmt.Sprintf(ShortLinkKey, shortLink), exp, []byte(url))
	//err = r.Cli.Set(fmt.Sprintf(ShortLinkKey, shortLink), url,
	//	time.Minute*time.Duration(exp)).Err()
	if err != nil {
		return "", err
	}

	err = r.Cli.Setex(fmt.Sprintf(UrlHashKey, urlHash), exp, []byte(shortLink))
	if err != nil {
		return "", nil
	}
	detail, err := json.Marshal(&UrlDetail{
		Url:                 url,
		CreateAt:            time.Now().String(),
		ExpirationInMinutes: time.Duration(exp),
	})
	if err != nil {
		return "", err
	}
	err = r.Cli.Setex(fmt.Sprintf(ShortLinkDetailKey, shortLink), exp, detail)
	if err != nil {
		return "", err
	}
	return shortLink, nil
}

//返回一个32位md5加密后的字符串
func GetMD5Encode(data string) string {
	h := md5.New()
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

//返回一个16位md5加密后的字符串
func Get16MD5Encode(data string) string {
	return GetMD5Encode(data)[8:24]
}

func toHash(url string) string {
	return Get16MD5Encode(url)
}

func (r *RedisCli) ShortLinkInfo(shortLink string) (interface{}, error) {
	detail, _ := r.Cli.Get(fmt.Sprintf(ShortLinkDetailKey, shortLink))
	if detail == nil {
		return "", nil
	} else {
		var return_data = &UrlDetail{}
		err := json.Unmarshal(detail, return_data)
		if err != nil {
			return "", err
		}
		if err != nil {
			return "", nil
		}
		return return_data, nil
	}
}

func (r *RedisCli) UnShorten(shortLink string) (string, error) {
	url, _ := r.Cli.Get(fmt.Sprintf(ShortLinkKey, shortLink))
	if url == nil {
		return "", nil
	} else {
		return string(url), nil
	}
}

type UrlDetail struct {
	Url                 string        `json:"url"`
	CreateAt            string        `json:"create_at"`
	ExpirationInMinutes time.Duration `json:"expiration_in_minutes"`
}

func NewRedisCli(addr string, pwd string, db int) *RedisCli {
	var client goredis.Client
	client.Addr = addr
	client.Password = pwd
	client.Db = db
	return &RedisCli{Cli: &client}
}
