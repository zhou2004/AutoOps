package util

import (
	"fmt"
	"io"

	"golang.org/x/crypto/ssh"
)

// ExecuteRemoteCommandStream forwards output as it arrives instead of waiting
// for command completion. The caller owns the writer and its lifetime.
func (s *SSHUtil) ExecuteRemoteCommandStream(auth *SSHConfig, command string, output io.Writer) error {
	config, err := s.getSSHConfig(auth)
	if err != nil {
		return err
	}
	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", auth.IP, auth.Port), config)
	if err != nil {
		return fmt.Errorf("连接日志通道失败: %w", err)
	}
	defer conn.Close()
	session, err := conn.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()
	session.Stdout = output
	session.Stderr = output
	return session.Run(command)
}
