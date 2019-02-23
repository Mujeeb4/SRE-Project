// Copyright 2019 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package lastcommit

import (
	"fmt"
	"time"

	"code.gitea.io/git"
	"code.gitea.io/gitea/modules/cache"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
)

var (
	// Cache defines globally last commit cache object
	Cache git.LastCommitCache
)

// NewContext init
func NewContext() error {
	var err error
	switch setting.CacheService.LastCommit.Type {
	case "default":
		if cache.Cache != nil {
			Cache = &lastCommitCache{
				mc:      cache.Cache,
				timeout: int64(setting.CacheService.TTL / time.Second),
			}
		} else {
			log.Warn("Last Commit Cache Enabled but Cache Service not Configed Well")
			return nil
		}
	case "memory":
		Cache = &MemoryCache{}
	case "boltdb":
		Cache, err = NewBoltDBCache(setting.CacheService.LastCommit.ConnStr)
	case "redis":
		addrs, pass, dbIdx, err := parseConnStr(setting.CacheService.LastCommit.ConnStr)
		if err != nil {
			return err
		}

		Cache, err = NewRedisCache(addrs, pass, dbIdx)
	case "":
		return nil
	default:
		return fmt.Errorf("Unsupported last commit type: %s", setting.CacheService.LastCommit.Type)
	}
	if err == nil {
		log.Info("Last Commit Cache %s Enabled", setting.CacheService.LastCommit.Type)
	}
	return err
}
