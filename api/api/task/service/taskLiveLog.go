package service

import (
	"fmt"
	"io"
	"os"
)

func taskLogFollowCommand(logPath, exitPath, pid string) string {
	return fmt.Sprintf(`offset=0
while :; do
 finished=0
 if test -f %s || ! kill -0 %s 2>/dev/null; then finished=1; fi
 size=$(wc -c < %s) || exit 1
 if test "$size" -gt "$offset"; then
  tail -c +$((offset + 1)) %s | head -c $((size - offset))
  offset=$size
 fi
 test "$finished" -eq 1 && break
 sleep 0.2
done`, shellQuote(exitPath), shellQuote(pid), shellQuote(logPath), shellQuote(logPath))
}

func readLiveTaskLog(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return "", err
	}
	const limit = 2 * 1024 * 1024
	prefix := ""
	if info.Size() > limit {
		if _, err := file.Seek(info.Size()-limit, io.SeekStart); err != nil {
			return "", err
		}
		prefix = "[实时窗口仅显示最近 2 MiB，完整日志将在结束后保留]\n"
	}
	data, err := io.ReadAll(io.LimitReader(file, limit))
	return prefix + string(data), err
}
