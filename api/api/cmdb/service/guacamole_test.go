package service

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"reflect"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"dodevops-api/api/cmdb/model"
	configmodel "dodevops-api/api/configcenter/model"
	"dodevops-api/common/util"
)

func TestGuacamoleUnicodeAndFragmentedInstructions(t *testing.T) {
	values := []string{"connect", "用户😀", "密碼.,;\n", "", "𐐷"}
	encoded := EncodeGuacamole(values...)
	if !strings.Contains(encoded, "3.用户😀") {
		t.Fatal("length must count Unicode characters")
	}
	reader := bufio.NewReader(iotest.OneByteReader(strings.NewReader(encoded + EncodeGuacamole("sync", "123"))))
	decoded, err := ReadGuacamole(reader)
	if err != nil || !reflect.DeepEqual(values, decoded) {
		t.Fatalf("fragmented Unicode round trip: %#v %v", decoded, err)
	}
	next, err := ReadGuacamole(reader)
	if err != nil || !reflect.DeepEqual(next, []string{"sync", "123"}) {
		t.Fatalf("instruction boundary: %#v %v", next, err)
	}
}

func TestGuacamoleRejectsMalformedAndOversizedInput(t *testing.T) {
	cases := []string{".x;", "-1.x;", "999999999.x;", "4.sync!", "2.x;", "1.\xff;", "4194305.", strings.Repeat("0.,", 256) + "0.;"}
	for _, input := range cases {
		if _, err := ReadGuacamole(bufio.NewReader(strings.NewReader(input))); err == nil {
			t.Errorf("accepted malformed instruction %q", input)
		}
	}
}

func TestGuacamoleReadOnlyAndStreamFiltering(t *testing.T) {
	for _, op := range []string{"key", "mouse"} {
		if AllowedGuacamoleInput(op, true) || !AllowedGuacamoleInput(op, false) {
			t.Errorf("read-only control not enforced for %s", op)
		}
	}
	for _, op := range []string{"clipboard", "file", "pipe", "blob", "end", "put", "get", "audio", "connect", "select"} {
		if AllowedGuacamoleInput(op, true) || AllowedGuacamoleInput(op, false) {
			t.Errorf("stream/control bypass: %s", op)
		}
	}
	for _, op := range []string{"sync", "ack", "size", "disconnect", "nop", ""} {
		if !AllowedGuacamoleInput(op, true) {
			t.Errorf("required view-only instruction blocked: %s", op)
		}
	}
}

func TestGuacamoleMockGatewayHandshake(t *testing.T) {
	db := connectionTestDB(t)
	credential := configmodel.EcsAuth{Name: "desktop", Type: 1, Username: "管理员", Password: "test-only-密碼", CreateTime: util.HTime{Time: time.Now()}}
	if err := db.Create(&credential).Error; err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	t.Setenv("CMDB_GUACD_ADDR", listener.Addr().String())
	done := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			done <- err
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
		reader := bufio.NewReader(conn)
		selection, err := ReadGuacamole(reader)
		if err != nil || !reflect.DeepEqual(selection, []string{"select", "rdp"}) {
			done <- fmt.Errorf("select instruction invalid: %v", err)
			return
		}
		argNames := []string{"VERSION_1_5_0", "hostname", "port", "username", "password", "domain", "security", "ignore-cert", "read-only", "disable-copy", "disable-paste", "enable-drive", "enable-printing", "enable-sftp", "unknown-arg"}
		_, err = io.WriteString(conn, EncodeGuacamole(append([]string{"args"}, argNames...)...))
		if err != nil {
			done <- err
			return
		}
		expected := [][]string{{"size", "1280", "800", "96"}, {"audio"}, {"video"}, {"image", "image/png", "image/jpeg"}}
		for _, want := range expected {
			got, e := ReadGuacamole(reader)
			if e != nil || !reflect.DeepEqual(got, want) {
				done <- fmt.Errorf("initial instruction mismatch: %#v %v", got, e)
				return
			}
		}
		connect, err := ReadGuacamole(reader)
		if err != nil {
			done <- err
			return
		}
		want := []string{"connect", "VERSION_1_1_0", "192.0.2.30", "3389", "管理员", "test-only-密碼", "EXAMPLE", "nla", "false", "true", "true", "true", "false", "false", "false", ""}
		if !reflect.DeepEqual(connect, want) {
			done <- fmt.Errorf("connect parameter order or Unicode value mismatch")
			return
		}
		_, err = io.WriteString(conn, EncodeGuacamole("ready", "test-session"))
		done <- err
	}()
	profile := model.CmdbAssetConnection{Protocol: "rdp", Address: "192.0.2.30", Port: 3389, CredentialID: credential.ID, Domain: "EXAMPLE", Security: "nla", ReadOnly: true, Enabled: true}
	conn, _, err := ConnectGuacamole(db, &profile, 20, 99999)
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
