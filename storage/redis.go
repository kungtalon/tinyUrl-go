package storage

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"go-tiny-url/common"
	"hash"
	"io"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/mattheath/base62"
)

const (
	// URL_ID_KEY is global counter
	URL_ID_KEY = "go_tiny_url:next.url.id"
	// ShortLinkKey mapping the short link to the url
	SHORT_LINK_KEY = "go_tiny_url:short_link:%v:url"
	// URLHashKey mapping the hash of the url to the short link
	URL_HASH_KEY = "go_tiny_url:url_hash:%v:url"
	// ShortLinkDetailKey mapping the shortLink to the detail of the url
	SHORT_LINK_DETAIL_KEY = "go_tiny_url:shortlink:%v:detail"
)

// RedisCli contains a redis Client
type RedisCli struct {
	Cli *redis.Client
	ctx context.Context
}

// UrlDetail contains the detail of the shortLink
type UrlDetail struct {
	URL string `json:"url"`
	CreatedAt string `json:"created_at"`
	ExpirationInMinutes time.Duration `json:"expiration_in_minutes"`
}

func toSha1(s string) hash.Hash {
	enc := sha1.New()
	io.WriteString(enc, s)
	return enc
}

func getShortLinkKey(key interface{}) string {
	return fmt.Sprintf(SHORT_LINK_KEY, key)
}

func getUrlHashKey(key interface{}) string {
	return fmt.Sprintf(URL_HASH_KEY, key)
}

func getShortLinkDetailKey(key interface{}) string {
	return fmt.Sprintf(SHORT_LINK_DETAIL_KEY, key)
}


// NewRedisCli create a redis Client
func NewRedisCli(ctx context.Context, addr string, passwd string, db int) *RedisCli {
	c := redis.NewClient(&redis.Options{
		Addr: addr,
		Password: passwd,
		DB: db,
	})

	if _, err := c.Ping(ctx).Result(); err != nil {
		panic(err)
	}

	return &RedisCli{Cli: c, ctx: ctx}
}

// Shorten converts url to shortLink
func (r *RedisCli) Shorten(url string, exp int64) (l string, err error) {
	// convert url to sha1 hash
	h := toSha1(url)

	// fetch it if the url is cached
	l, err = r.Cli.Get(r.ctx, getUrlHashKey(h)).Result()
	if err == redis.Nil {
		// not found, do nothing
	} else {
		if l == "{}" {
			// expiration, do nothing
		} else {
			return 
		}
	}

	// increase the global counter
	err = r.Cli.Incr(r.ctx, URL_ID_KEY).Err()
	if err != nil {
		return 
	}

	// encode global counter to base62
	// TODO: this is not thread safe
	id, err := r.Cli.Get(r.ctx, URL_ID_KEY).Int64()
	if err != nil {
		return 
	}

	// encode the url to base62 (base64 is bad for it has "+" "/")
	eid := base62.EncodeInt64(id)

	err = r.Cli.Set(r.ctx, getShortLinkKey(eid), url,
					 time.Minute * time.Duration(exp)).Err()
	if err != nil {
		return 
	}

	// store the url against the hash of it
	err = r.Cli.Set(r.ctx, getUrlHashKey(h), eid,
					time.Minute * time.Duration(exp)).Err()
	if err != nil {
		return 
	}

	detail, err := json.Marshal(
		&UrlDetail {
			URL: url,
			CreatedAt: time.Now().String(),
			ExpirationInMinutes: time.Duration(exp),
		})
	if err != nil {
		return 
	}

	// store the url detail against the encoded id
	err = r.Cli.Set(r.ctx, getShortLinkDetailKey(eid), detail, 
						time.Minute * time.Duration(exp)).Err()
	if err != nil {
		return 
	}

	return eid, nil
}

// ShortLinkInfo returens the detail of the shortlink
func (r *RedisCli) ShortLinkInfo(eid string) (d interface{}, err error) {
	d, err = r.Cli.Get(r.ctx, getShortLinkDetailKey(eid)).Result()
	if err == redis.Nil {
		return "", common.StatusError{404, errors.New("Unknown short URL")}
	} 

	return 
}

// Unshorten convert the shortLink to url
func (r *RedisCli) Unshorten(eid string) (url string, err error) {
	url, err = r.Cli.Get(r.ctx, getShortLinkKey(eid)).Result()
	if err == redis.Nil {
		return "", common.StatusError{404, errors.New("shortlink not found")}
	} 

	return 
}