package service

import (
	"context"
	"crypto/sha256"
	"dodevops-api/pkg/rediscompat"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"sync"

	"github.com/go-redis/redis/v8"
)

type clusterScanCursor struct {
	Source   uint   `json:"source"`
	Topology string `json:"topology"`
	Shard    int    `json:"shard"`
	Cursor   uint64 `json:"cursor"`
	Match    string `json:"match"`
}

func (s *DatabaseOperationService) RedisKeysCursor(ctx context.Context, id uint, cursor, match string, count int64) ([]RedisKeySummary, any, error) {
	client, _, e := s.redisClient(id)
	if e != nil {
		return nil, nil, e
	}
	defer client.Close()
	cluster, ok := client.(*redis.ClusterClient)
	if !ok {
		n, e := strconv.ParseUint(cursor, 10, 64)
		if e != nil {
			return nil, nil, errors.New("单机游标必须为非负整数")
		}
		items, next, e := s.RedisKeys(ctx, id, n, match, count)
		return items, next, e
	}
	if count < 1 || count > 200 {
		count = 100
	}
	if len(match) > 256 || len(cursor) > 4096 {
		return nil, nil, errors.New("扫描参数超过限制")
	}
	slots, e := rediscompat.Slots(ctx, cluster, "")
	if e != nil {
		return nil, nil, errors.New("无法读取 Cluster 槽位拓扑")
	}
	identities := []string{}
	for _, slot := range slots {
		if len(slot.Nodes) > 0 {
			identities = append(identities, strconv.Itoa(slot.Start)+"-"+strconv.Itoa(slot.End)+"/"+slot.Nodes[0].ID+"/"+slot.Nodes[0].Addr)
		}
	}
	sort.Strings(identities)
	encoded, _ := json.Marshal(identities)
	hash := sha256.Sum256(encoded)
	topology := hex.EncodeToString(hash[:])
	position := clusterScanCursor{Source: id, Topology: topology, Match: match}
	if cursor != "" && cursor != "0" {
		raw, e := base64.RawURLEncoding.DecodeString(cursor)
		if e != nil || json.Unmarshal(raw, &position) != nil || position.Source != id || position.Topology != topology || position.Match != match {
			return nil, nil, errors.New("扫描游标已失效（拓扑或筛选变化），请重新扫描")
		}
	}
	nodes := map[string]*redis.Client{}
	var mu sync.Mutex
	if e = cluster.ForEachMaster(ctx, func(ctx context.Context, node *redis.Client) error {
		mu.Lock()
		nodes[node.Options().Addr] = node
		mu.Unlock()
		return nil
	}); e != nil {
		return nil, nil, errors.New("无法枚举主分片")
	}
	addresses := []string{}
	for address := range nodes {
		addresses = append(addresses, address)
	}
	sort.Strings(addresses)
	if position.Shard < 0 || position.Shard >= len(addresses) {
		return nil, nil, errors.New("扫描分片已变化，请重新扫描")
	}
	node := nodes[addresses[position.Shard]]
	keys, next, e := node.Scan(ctx, position.Cursor, match, count).Result()
	if e != nil {
		return nil, nil, errors.New("主分片 SCAN 失败")
	}
	out := []RedisKeySummary{}
	for _, key := range keys {
		kind, e := cluster.Type(ctx, key).Result()
		if e != nil {
			continue
		}
		ttl, e := cluster.TTL(ctx, key).Result()
		if e == nil {
			out = append(out, RedisKeySummary{Key: key, Type: kind, TTL: int64(ttl.Seconds())})
		}
	}
	position.Cursor = next
	if next == 0 {
		position.Shard++
	}
	if position.Shard == len(addresses) {
		return out, uint64(0), nil
	}
	raw, _ := json.Marshal(position)
	return out, base64.RawURLEncoding.EncodeToString(raw), nil
}
