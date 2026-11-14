package service

import (
	"context"
	"crypto/tls"
	monitormodel "dodevops-api/api/monitor/model"
	"dodevops-api/common/config"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"net"
	"net/mail"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"
)

type AccountSMTP struct{ Host, Port, User, Password, From, Mode string }

func AccountMailer(db *gorm.DB) AccountSMTP {
	values := map[string]string{}
	var rows []monitormodel.AlertConfig
	db.Where("conf_key IN ?", []string{"Email_host", "Email_port", "Email_user", "Email_password"}).Find(&rows)
	for _, r := range rows {
		values[r.ConfKey] = r.ConfValue
	}
	local := config.AccountSMTPConfig{}
	if config.Config != nil {
		local = config.Config.AccountSMTP
	}
	for key, value := range map[string]string{"Email_host": local.Host, "Email_port": local.Port, "Email_user": local.User, "Email_password": local.Password} {
		if value != "" {
			values[key] = value
		}
	}
	read := func(key, fallback string) string {
		if v, ok := os.LookupEnv(key); ok {
			return v
		}
		return fallback
	}
	c := AccountSMTP{Host: read("AUTOOPS_SMTP_HOST", values["Email_host"]), Port: read("AUTOOPS_SMTP_PORT", values["Email_port"]), User: read("AUTOOPS_SMTP_USER", values["Email_user"]), Password: read("AUTOOPS_SMTP_PASSWORD", values["Email_password"]), From: read("AUTOOPS_SMTP_FROM", local.From), Mode: read("AUTOOPS_SMTP_TLS", local.TLS)}
	if c.Port == "" {
		c.Port = "587"
	}
	if c.From == "" {
		c.From = c.User
	}
	if c.Mode == "" {
		c.Mode = "starttls"
		if c.Port == "465" {
			c.Mode = "tls"
		}
	}
	return c
}
func (c AccountSMTP) Ready() bool {
	n, err := strconv.Atoi(c.Port)
	_, addrErr := mail.ParseAddress(c.From)
	return c.Host != "" && n > 0 && n < 65536 && err == nil && addrErr == nil && (c.Mode == "tls" || c.Mode == "starttls")
}
func (c AccountSMTP) Send(ctx context.Context, to, purpose, code string) error {
	if !c.Ready() {
		return errors.New("SMTP not configured")
	}
	address, err := mail.ParseAddress(c.From)
	if err != nil {
		return err
	}
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	target := net.JoinHostPort(c.Host, c.Port)
	tlsConfig := &tls.Config{ServerName: c.Host, MinVersion: tls.VersionTLS12}
	var conn net.Conn
	if c.Mode == "tls" {
		conn, err = (&tls.Dialer{NetDialer: dialer, Config: tlsConfig}).DialContext(ctx, "tcp", target)
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", target)
	}
	if err != nil {
		return err
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(20 * time.Second))
	stop := context.AfterFunc(ctx, func() { conn.Close() })
	defer stop()
	client, err := smtp.NewClient(conn, c.Host)
	if err != nil {
		return err
	}
	defer client.Close()
	if c.Mode == "starttls" {
		if err = client.StartTLS(tlsConfig); err != nil {
			return err
		}
	}
	if c.User != "" {
		if err = client.Auth(smtp.PlainAuth("", c.User, c.Password, c.Host)); err != nil {
			return err
		}
	}
	if err = client.Mail(address.Address); err != nil {
		return err
	}
	if err = client.Rcpt(to); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	action := "account registration"
	if purpose == "reset" {
		action = "password reset"
	}
	body := fmt.Sprintf("From: AutoOps <%s>\r\nTo: %s\r\nSubject: AutoOps verification code\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nAutoOps %s\r\n\r\nYour verification code / 您的验证码: %s\r\n\r\nValid for 10 minutes, once only. / 10 分钟内有效，仅可使用一次。\r\nIf you did not request this, ignore this email. / 如非本人操作，请忽略此邮件。\r\n", address.Address, to, action, code)
	if _, err = writer.Write([]byte(strings.ReplaceAll(body, "\x00", ""))); err != nil {
		return err
	}
	if err = writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}
