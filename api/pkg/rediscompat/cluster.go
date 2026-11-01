// Package rediscompat isolates Redis 7 networking metadata compatibility from
// the existing v8 clients used by the rest of AutoOps.
package rediscompat

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"

	"github.com/go-redis/redis/v8"
)

type commandClient interface {
	Do(context.Context, ...interface{}) *redis.Cmd
}

func Slots(ctx context.Context, client commandClient, fallback string) ([]redis.ClusterSlot, error) {
	raw, e := client.Do(ctx, "CLUSTER", "SLOTS").Result()
	if e != nil {
		return nil, e
	}
	return ParseSlots(raw, fallback)
}

func ParseSlots(raw interface{}, fallback string) ([]redis.ClusterSlot, error) {
	rows, ok := raw.([]interface{})
	if !ok {
		return nil, errors.New("invalid CLUSTER SLOTS response")
	}
	result := []redis.ClusterSlot{}
	integer := func(v interface{}) (int, error) { n, e := strconv.Atoi(fmt.Sprint(v)); return n, e }
	for _, row := range rows {
		fields, ok := row.([]interface{})
		if !ok || len(fields) < 3 {
			return nil, errors.New("invalid slot range")
		}
		start, e := integer(fields[0])
		end, e2 := integer(fields[1])
		if e != nil || e2 != nil || start < 0 || end > 16383 || end < start {
			return nil, errors.New("invalid slot bounds")
		}
		slot := redis.ClusterSlot{Start: start, End: end}
		for _, rawNode := range fields[2:] {
			node, ok := rawNode.([]interface{})
			if !ok || len(node) < 2 {
				return nil, errors.New("invalid slot node")
			}
			host, _ := node[0].(string)
			if host == "" {
				host = fallback
			}
			port, e := integer(node[1])
			if e != nil || port < 1 || port > 65535 || host == "" || host == "?" {
				return nil, errors.New("unknown slot endpoint")
			}
			id := ""
			if len(node) > 2 {
				id, _ = node[2].(string)
			}
			slot.Nodes = append(slot.Nodes, redis.ClusterNode{ID: id, Addr: net.JoinHostPort(host, strconv.Itoa(port))})
		}
		result = append(result, slot)
	}
	return result, nil
}

// Network failures must escape the cluster redirect loop. MOVED/ASK are Redis
// errors proving the command was not executed; these retain native routing.
type noUnknownReplay struct{}

func (noUnknownReplay) BeforeProcess(ctx context.Context, _ redis.Cmder) (context.Context, error) {
	return ctx, nil
}
func (noUnknownReplay) BeforeProcessPipeline(ctx context.Context, _ []redis.Cmder) (context.Context, error) {
	return ctx, nil
}
func (noUnknownReplay) AfterProcess(_ context.Context, cmd redis.Cmder) error {
	e := cmd.Err()
	if e == nil || e == redis.Nil {
		return nil
	}
	if _, ok := e.(redis.Error); ok {
		return nil
	}
	return errors.New("Redis request result is unknown; command was not replayed")
}
func (h noUnknownReplay) AfterProcessPipeline(ctx context.Context, cmds []redis.Cmder) error {
	for _, cmd := range cmds {
		if e := h.AfterProcess(ctx, cmd); e != nil {
			return e
		}
	}
	return nil
}

func NewCluster(options *redis.ClusterOptions) *redis.ClusterClient {
	options.MaxRetries = -1
	options.NewClient = func(o *redis.Options) *redis.Client {
		o.MaxRetries = -1
		client := redis.NewClient(o)
		client.AddHook(noUnknownReplay{})
		return client
	}
	options.ClusterSlots = func(ctx context.Context) ([]redis.ClusterSlot, error) {
		var last error
		for _, address := range options.Addrs {
			node := redis.NewClient(&redis.Options{Addr: address, Username: options.Username, Password: options.Password, TLSConfig: options.TLSConfig, MaxRetries: -1, DialTimeout: options.DialTimeout, ReadTimeout: options.ReadTimeout})
			host, _, _ := net.SplitHostPort(address)
			slots, e := Slots(ctx, node, host)
			node.Close()
			if e == nil {
				return slots, nil
			}
			last = e
		}
		return nil, last
	}
	return redis.NewClusterClient(options)
}
