package rediscompat

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
)

func TestRedis7SlotMetadataAndNullEndpoint(t *testing.T) {
	rows := []interface{}{[]interface{}{int64(0), int64(16383), []interface{}{nil, int64(6379), "primary", []interface{}{"hostname", ""}}, []interface{}{"::1", int64(6380), "replica"}}}
	slots, e := ParseSlots(rows, "192.0.2.1")
	if e != nil || len(slots) != 1 || slots[0].Nodes[0].Addr != "192.0.2.1:6379" || slots[0].Nodes[1].Addr != "[::1]:6380" {
		t.Fatalf("Redis 7 metadata parse failed: %v %v", slots, e)
	}
	if _, e = ParseSlots([]interface{}{[]interface{}{int64(0), int64(20000), []interface{}{"a", int64(1)}}}, ""); e == nil {
		t.Fatal("invalid slots accepted")
	}
}

func fakeRedis(t *testing.T, fn func([]string) (string, bool)) string {
	t.Helper()
	listener, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { listener.Close() })
	go func() {
		for {
			c, e := listener.Accept()
			if e != nil {
				return
			}
			go func() {
				defer c.Close()
				c.SetDeadline(time.Now().Add(5 * time.Second))
				r := bufio.NewReader(c)
				for {
					line, e := r.ReadString('\n')
					if e != nil {
						return
					}
					n, e := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "*")))
					if e != nil {
						return
					}
					args := []string{}
					for i := 0; i < n; i++ {
						if _, e = r.ReadString('\n'); e != nil {
							return
						}
						arg, e := r.ReadString('\n')
						if e != nil {
							return
						}
						args = append(args, strings.TrimSuffix(strings.TrimSuffix(arg, "\n"), "\r"))
					}
					response, closeNow := fn(args)
					if closeNow {
						return
					}
					if _, e = c.Write([]byte(response)); e != nil {
						return
					}
				}
			}()
		}
	}()
	return listener.Addr().String()
}
func slotReply(address string) string {
	host, port, _ := net.SplitHostPort(address)
	return fmt.Sprintf("*1\r\n*3\r\n:0\r\n:16383\r\n*4\r\n$%d\r\n%s\r\n:%s\r\n$4\r\nnode\r\n*0\r\n", len(host), host, port)
}

func TestUnknownWriteIsNeverReplayedByClusterRedirectLoop(t *testing.T) {
	var writes atomic.Int32
	var address string
	address = fakeRedis(t, func(args []string) (string, bool) {
		switch strings.ToLower(args[0]) {
		case "cluster":
			return slotReply(address), false
		case "command":
			return "*0\r\n", false
		case "incr":
			writes.Add(1)
			return "", true
		default:
			return "+OK\r\n", false
		}
	})
	client := NewCluster(&redis.ClusterOptions{Addrs: []string{address}, MaxRedirects: 8, DialTimeout: time.Second, ReadTimeout: time.Second})
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := client.Incr(ctx, "counter").Err()
	if err == nil || !strings.Contains(err.Error(), "not replayed") || writes.Load() != 1 {
		t.Fatalf("uncertain write replayed: writes=%d err=%v", writes.Load(), err)
	}
}

func TestMovedAndAskStillRouteWritesOnce(t *testing.T) {
	for _, redirect := range []string{"MOVED", "ASK"} {
		t.Run(redirect, func(t *testing.T) {
			var applied atomic.Int32
			var asking atomic.Bool
			destination := fakeRedis(t, func(args []string) (string, bool) {
				switch strings.ToLower(args[0]) {
				case "asking":
					asking.Store(true)
					return "+OK\r\n", false
				case "incr":
					if redirect == "ASK" && !asking.Load() {
						return "-ERR ASKING required\r\n", false
					}
					applied.Add(1)
					return ":1\r\n", false
				default:
					return "+OK\r\n", false
				}
			})
			var source string
			source = fakeRedis(t, func(args []string) (string, bool) {
				switch strings.ToLower(args[0]) {
				case "cluster":
					return slotReply(source), false
				case "command":
					return "*0\r\n", false
				case "incr":
					return "-" + redirect + " 0 " + destination + "\r\n", false
				default:
					return "+OK\r\n", false
				}
			})
			client := NewCluster(&redis.ClusterOptions{Addrs: []string{source}, MaxRedirects: 8, DialTimeout: time.Second, ReadTimeout: time.Second})
			defer client.Close()
			if n, e := client.Incr(context.Background(), "key").Result(); e != nil || n != 1 || applied.Load() != 1 {
				t.Fatalf("%s failed: %d %v applied=%d", redirect, n, e, applied.Load())
			}
		})
	}
}
