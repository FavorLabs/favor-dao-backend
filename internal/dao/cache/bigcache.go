package cache

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"strings"
	"time"

	"favor-dao-backend/internal/core"
	"favor-dao-backend/internal/model"
	"favor-dao-backend/internal/model/rest"
	"github.com/Masterminds/semver/v3"
	"github.com/allegro/bigcache/v3"
	"github.com/sirupsen/logrus"
)

var (
	_ core.CacheIndexService = (*bigCacheIndexServant)(nil)
	_ core.VersionInfo       = (*bigCacheIndexServant)(nil)
)

type postsEntry struct {
	key    string
	tweets *rest.IndexTweetsResp
}

type bigCacheIndexServant struct {
	ips core.IndexPostsService

	indexActionCh      chan *core.IndexAction
	cachePostsCh       chan *postsEntry
	cache              *bigcache.BigCache
	lastCacheResetTime time.Time
	preventDuration    time.Duration
}

func (s *bigCacheIndexServant) IndexPosts(user *model.User, offset int, limit int) (*rest.IndexTweetsResp, error) {
	key := s.keyFrom(user, offset, limit)
	posts, err := s.getPosts(key)
	if err == nil {
		logrus.Debugf("bigCacheIndexServant.IndexPosts get index posts from cache by key: %s", key)
		return posts, nil
	}

	if posts, err = s.ips.IndexPosts(user, offset, limit); err != nil {
		return nil, err
	}
	logrus.Debugf("bigCacheIndexServant.IndexPosts get index posts from database by key: %s", key)
	s.cachePosts(key, posts)
	return posts, nil
}

func (s *bigCacheIndexServant) getPosts(key string) (*rest.IndexTweetsResp, error) {
	data, err := s.cache.Get(key)
	if err != nil {
		logrus.Debugf("bigCacheIndexServant.getPosts get posts by key: %s from cache err: %v", key, err)
		return nil, err
	}
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	var resp rest.IndexTweetsResp
	if err := dec.Decode(&resp); err != nil {
		logrus.Debugf("bigCacheIndexServant.getPosts get posts from cache in decode err: %v", err)
		return nil, err
	}
	return &resp, nil
}

func (s *bigCacheIndexServant) cachePosts(key string, tweets *rest.IndexTweetsResp) {
	entry := &postsEntry{key: key, tweets: tweets}
	select {
	case s.cachePostsCh <- entry:
		logrus.Debugf("bigCacheIndexServant.cachePosts cachePosts by chan of key: %s", key)
	default:
		go func(ch chan<- *postsEntry, entry *postsEntry) {
			logrus.Debugf("bigCacheIndexServant.cachePosts cachePosts indexAction by goroutine of key: %s", key)
			ch <- entry
		}(s.cachePostsCh, entry)
	}
}

func (s *bigCacheIndexServant) setPosts(entry *postsEntry) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(entry.tweets); err != nil {
		logrus.Debugf("bigCacheIndexServant.setPosts setPosts encode post entry err: %v", err)
		return
	}
	if err := s.cache.Set(entry.key, buf.Bytes()); err != nil {
		logrus.Debugf("bigCacheIndexServant.setPosts setPosts set cache err: %v", err)
	}
	logrus.Debugf("bigCacheIndexServant.setPosts setPosts set cache by key: %s", entry.key)
}

func (s *bigCacheIndexServant) keyFrom(user *model.User, offset int, limit int) string {
	var userId = ""
	if user != nil {
		userId = user.Address
	}
	return fmt.Sprintf("index:%s:%d:%d", userId, offset, limit)
}

func (s *bigCacheIndexServant) SendAction(act core.IdxAct, post *model.Post) {
	action := core.NewIndexAction(act, post)
	select {
	case s.indexActionCh <- action:
		logrus.Debugf("bigCacheIndexServant.SendAction send indexAction by chan: %s", act)
	default:
		go func(ch chan<- *core.IndexAction, act *core.IndexAction) {
			logrus.Debugf("bigCacheIndexServant.SendAction send indexAction by goroutine: %s", action.Act)
			ch <- act
		}(s.indexActionCh, action)
	}
}

func (s *bigCacheIndexServant) startIndexPosts() {
	for {
		select {
		case entry := <-s.cachePostsCh:
			s.setPosts(entry)
		case action := <-s.indexActionCh:
			s.handleIndexAction(action)
		}
	}
}

func (s *bigCacheIndexServant) handleIndexAction(action *core.IndexAction) {
	act, post := action.Act, action.Post

	switch act {
	case core.IdxActCreatePost, core.IdxActDeletePost:
		if post.Visibility == model.PostVisitPrivate {
			s.deleteCacheByAddress(post.Address)
			return
		}
	}

	// Clear all caches if within s.preventDuration time, otherwise clear only your own cache
	// TODO: Optimization is needed to clear only the affected cache, subsequent refinement
	if time.Since(s.lastCacheResetTime) > s.preventDuration {
		s.cache.Reset()
		s.lastCacheResetTime = time.Now()
		logrus.Debugf("bigCacheIndexServant.handleIndexAction reset cache by %s", action.Act)
	} else {
		s.deleteCacheByAddress(post.Address)
	}
}

func (s *bigCacheIndexServant) deleteCacheByAddress(address string) {
	var keys []string

	for it := s.cache.Iterator(); it.SetNext(); {
		entry, err := it.Value()
		if err != nil {
			logrus.Debugf("bigCacheIndexServant.deleteCacheByUserId usrId: %s err:%s", address, err)
			return
		}
		key := entry.Key()
		keyParts := strings.Split(key, ":")
		if len(keyParts) > 2 && keyParts[0] == "index" && keyParts[1] == address {
			keys = append(keys, key)
		}
	}

	for _, k := range keys {
		s.cache.Delete(k)
	}
	s.lastCacheResetTime = time.Now()
	logrus.Debugf("bigCacheIndexServant.deleteCacheByUserId userId:%s", address)
}

func (s *bigCacheIndexServant) Name() string {
	return "BigCacheIndex"
}

func (s *bigCacheIndexServant) Version() *semver.Version {
	return semver.MustParse("v0.2.0")
}
