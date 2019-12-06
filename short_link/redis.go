package main

import (
	"encoding/json"
	"fmt"
	"github.com/speps/go-hashids"
	"time"
	"github.com/astaxie/goredis"
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

func (r *RedisCli) Shorten(url string, exp int64) (string, error) {
	urlHash := toHash(url)
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
	id,err := r.Cli.Incr(UrlIdKey)
	if err != nil {
		return "", err
	}
	//id, err := r.Cli.Get(UrlIdKey).Int64()
	//if err != nil {
	//	return "", err
	//}
	//shortLink := base62.EncodeInt64(id)
	shortLink := toHash(fmt.Sprint(ShortLinkKey,id))
	err = r.Cli.Setex(fmt.Sprintf(ShortLinkKey, shortLink), exp, []byte(url))
	//err = r.Cli.Set(fmt.Sprintf(ShortLinkKey, shortLink), url,
	//	time.Minute*time.Duration(exp)).Err()
	if err != nil {
		return "", err
	}

	err = r.Cli.Setex(fmt.Sprintf(UrlHashKey, urlHash),exp, []byte(shortLink))
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
	err = r.Cli.Setex(fmt.Sprintf(ShortLinkDetailKey, shortLink),exp, detail)
	if err != nil {
		return "", err
	}
	return shortLink, nil
}

func toHash(url string) string {
	hd := hashids.NewData()
	hd.Salt = url
	hd.MinLength = 0
	h, _ := hashids.NewWithData(hd)
	r, _ := h.Encode([]int{45, 434, 1313, 99})
	return r
}

func (r *RedisCli) ShortLinkInfo(shortLink string) (interface{}, error) {
	detail, _ := r.Cli.Get(fmt.Sprintf(ShortLinkDetailKey, shortLink))
	if detail == nil {
		return "", nil
	} else {
		var return_data = &UrlDetail{}
		err := json.Unmarshal(detail,return_data)
		if err != nil{
			return "", err
		}
		if err != nil{
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
