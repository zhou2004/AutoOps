package service

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"dodevops-api/api/cmdb/model"
	"gorm.io/gorm"
)

const maxGuacamoleInstruction = 4 * 1024 * 1024

// Guacamole lengths count Unicode characters, including multibyte credentials.
func EncodeGuacamole(values ...string) string {
	var out strings.Builder
	for i, value := range values {
		if i > 0 {
			out.WriteByte(',')
		}
		fmt.Fprintf(&out, "%d.%s", utf8.RuneCountInString(value), value)
	}
	out.WriteByte(';')
	return out.String()
}

// ReadGuacamole parses one bounded instruction, preserving UTF-8 boundaries
// when TCP packets split inside an image or a multibyte character.
func ReadGuacamole(reader *bufio.Reader) ([]string, error) {
	values := make([]string, 0, 8)
	total := 0
	for {
		var digits strings.Builder
		for {
			b, err := reader.ReadByte()
			if err != nil {
				return nil, err
			}
			if b == '.' {
				break
			}
			if b < '0' || b > '9' || digits.Len() >= 8 {
				return nil, errors.New("invalid guacamole length")
			}
			digits.WriteByte(b)
		}
		n, err := strconv.Atoi(digits.String())
		if err != nil || n > maxGuacamoleInstruction || total+n > maxGuacamoleInstruction {
			return nil, errors.New("guacamole instruction too large")
		}
		var value strings.Builder
		for i := 0; i < n; i++ {
			r, size, err := reader.ReadRune()
			if err != nil {
				return nil, err
			}
			if r == utf8.RuneError && size == 1 {
				return nil, errors.New("invalid guacamole UTF-8")
			}
			total += size
			if total > maxGuacamoleInstruction {
				return nil, errors.New("guacamole instruction too large")
			}
			value.WriteRune(r)
		}
		values = append(values, value.String())
		if len(values) > 256 {
			return nil, errors.New("too many guacamole arguments")
		}
		b, err := reader.ReadByte()
		if err != nil {
			return nil, err
		}
		if b == ';' {
			return values, nil
		}
		if b != ',' {
			return nil, errors.New("invalid guacamole delimiter")
		}
	}
}

// AllowedGuacamoleInput blocks clipboard/file/pipe streams server-side and
// enforces view-only connections even when a client sends crafted events.
func AllowedGuacamoleInput(op string, readOnly bool) bool {
	switch op {
	case "sync", "ack", "size", "disconnect", "nop", "":
		return true
	case "key", "mouse":
		return !readOnly
	default:
		return false
	}
}

func ConnectGuacamole(db *gorm.DB, p *model.CmdbAssetConnection, width, height int) (net.Conn, *bufio.Reader, error) {
	address := GuacdAddress()
	if address == "" {
		return nil, nil, errors.New("尚未配置图形连接网关 CMDB_GUACD_ADDR")
	}
	if !p.Enabled || (p.Protocol != "vnc" && p.Protocol != "rdp") {
		return nil, nil, errors.New("图形连接未启用")
	}
	credential, err := connectionCredential(db, p)
	if err != nil {
		return nil, nil, err
	}
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return nil, nil, errors.New("图形连接网关不可达")
	}
	success := false
	defer func() {
		if !success {
			conn.Close()
		}
	}()
	_ = conn.SetDeadline(time.Now().Add(20 * time.Second))
	reader := bufio.NewReader(conn)
	write := func(values ...string) error { _, e := io.WriteString(conn, EncodeGuacamole(values...)); return e }
	if err := write("select", p.Protocol); err != nil {
		return nil, nil, err
	}
	args, err := ReadGuacamole(reader)
	if err != nil || len(args) < 2 || args[0] != "args" {
		return nil, nil, errors.New("图形网关握手失败")
	}
	if width < 320 || width > 3840 {
		width = 1280
	}
	if height < 240 || height > 2160 {
		height = 800
	}
	for _, instruction := range [][]string{{"size", strconv.Itoa(width), strconv.Itoa(height), "96"}, {"audio"}, {"video"}, {"image", "image/png", "image/jpeg"}} {
		if err := write(instruction...); err != nil {
			return nil, nil, err
		}
	}
	username := p.Username
	if username == "" {
		username = credential.Username
	}
	params := map[string]string{
		"hostname": p.Address, "port": strconv.Itoa(p.Port), "username": username, "password": credential.Password,
		"domain": p.Domain, "security": p.Security, "ignore-cert": strconv.FormatBool(p.IgnoreCert),
		"read-only": strconv.FormatBool(p.ReadOnly), "disable-copy": "true", "disable-paste": "true",
		"enable-drive": "false", "enable-printing": "false", "enable-sftp": "false", "disable-audio": "true",
		"resize-method": "display-update", "color-depth": "24", "server-layout": "en-us-qwerty",
	}
	connect := []string{"connect"}
	for _, arg := range args[1:] {
		if strings.HasPrefix(arg, "VERSION_") {
			connect = append(connect, "VERSION_1_1_0")
		} else {
			connect = append(connect, params[arg])
		}
	}
	if err := write(connect...); err != nil {
		return nil, nil, err
	}
	ready, err := ReadGuacamole(reader)
	if err != nil || len(ready) < 2 || ready[0] != "ready" {
		return nil, nil, errors.New("远程桌面连接失败，请检查服务、凭据和证书设置")
	}
	_ = conn.SetDeadline(time.Time{})
	success = true
	return conn, reader, nil
}
