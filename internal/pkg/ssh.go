// Copyright 2022 The Goploy Authors. All rights reserved.
// Use of this source code is governed by a GPLv3-style
// license that can be found in the LICENSE file.

package pkg

import (
	"bytes"
	"errors"
	"fmt"
	"golang.org/x/crypto/ssh"
	"net"
	"os"
	"strings"
	"time"
)

// SSHConfig -
type SSHConfig struct {
	User         string
	Password     string
	Path         string
	Host         string
	Port         int
	JumpUser     string
	JumpPassword string
	JumpPath     string
	JumpHost     string
	JumpPort     int
	Timeout      time.Duration
}

func (sshConfig SSHConfig) Dial() (*ssh.Client, error) {
	if sshConfig.JumpHost != "" {
		// 连接跳板机
		clientConfig, err := sshConfig.getConfig(sshConfig.JumpUser, sshConfig.JumpPassword, sshConfig.JumpPath)
		if err != nil {
			return nil, err
		}
		sshClient, err := ssh.Dial("tcp", sshConfig.jumpAddr(), clientConfig)
		if err != nil {
			return nil, err
		}

		// 连接目标机
		conn, err := sshClient.Dial("tcp", sshConfig.addr())
		if err != nil {
			return nil, err
		}
		targetConfig, err := sshConfig.getConfig(sshConfig.User, sshConfig.Password, sshConfig.Path)
		if err != nil {
			return nil, err
		}
		ncc, chans, reqs, err := ssh.NewClientConn(conn, sshConfig.addr(), targetConfig)
		if err != nil {
			return nil, err
		}

		sshClient = ssh.NewClient(ncc, chans, reqs)
		return sshClient, err
	} else {
		clientConfig, err := sshConfig.getConfig(sshConfig.User, sshConfig.Password, sshConfig.Path)
		if err != nil {
			return nil, err
		}
		// connect to ssh
		sshClient, err := ssh.Dial("tcp", sshConfig.addr(), clientConfig)
		if err != nil {
			return nil, err
		}
		return sshClient, err
	}
}

// version|cpu cores|mem

// osInfoScript returns the remote command that yields three lines --
// "version", "cpu cores", "mem(KB)" -- for the target OS. Linux reads
// /etc/os-release and /proc; Windows uses PowerShell + CIM (the SSH default
// shell on Windows is cmd.exe, so PowerShell is invoked explicitly). The output
// shape is identical across OSes so the OSInfo field stays format-stable.
func osInfoScript(os string) string {
	if os == "windows" {
		return `powershell -NoProfile -Command "$o=Get-CimInstance Win32_OperatingSystem; $o.Caption + '|' + $env:NUMBER_OF_PROCESSORS + '|' + $o.TotalVisibleMemorySize"`
	}
	return `cat /etc/os-release | grep "PRETTY_NAME" | awk -F\" '{print $2}' && cat /proc/cpuinfo  | grep "processor" | wc -l && cat /proc/meminfo | grep MemTotal | awk '{print $2}'`
}

func (sshConfig SSHConfig) GetOSInfo(os string) string {
	client, err := sshConfig.Dial()
	if err != nil {
		return ""
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return ""
	}
	defer session.Close()

	var sshOutbuf, sshErrbuf bytes.Buffer
	session.Stdout = &sshOutbuf
	session.Stderr = &sshErrbuf
	if err := session.Run(osInfoScript(os)); err != nil {
		return ""
	}

	// version|cpu cores|mem -- PowerShell emits CRLF, so trim \r as well
	return strings.Replace(strings.Trim(sshOutbuf.String(), "\r\n"), "\n", "|", -1)
}

func (sshConfig SSHConfig) getConfig(user, password, path string) (*ssh.ClientConfig, error) {
	if user == "" {
		return nil, errors.New("no user detect")
	}

	auth := make([]ssh.AuthMethod, 0)

	if path != "" {
		pemBytes, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var signer ssh.Signer
		if password == "" {
			signer, err = ssh.ParsePrivateKey(pemBytes)
		} else {
			signer, err = ssh.ParsePrivateKeyWithPassphrase(pemBytes, []byte(password))
		}
		if err != nil {
			return nil, err
		}
		auth = append(auth, ssh.PublicKeys(signer))
	} else if password != "" {
		auth = append(auth, ssh.Password(password))
	} else {
		return nil, errors.New("no password or private key available")
	}

	config := ssh.Config{
		Ciphers: []string{"aes128-ctr", "aes192-ctr", "aes256-ctr", "aes128-gcm@openssh.com", "arcfour256", "arcfour128", "aes128-cbc", "3des-cbc", "aes192-cbc", "aes256-cbc"},
	}

	timeout := 30 * time.Second
	if sshConfig.Timeout > 0 {
		timeout = sshConfig.Timeout
	}

	return &ssh.ClientConfig{
		User:    user,
		Auth:    auth,
		Timeout: timeout,
		Config:  config,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}, nil
}

func (sshConfig SSHConfig) SetTimeout(duration time.Duration) SSHConfig {
	sshConfig.Timeout = duration
	return sshConfig
}

func (sshConfig SSHConfig) jumpAddr() string {
	return fmt.Sprintf("%s:%d", sshConfig.JumpHost, sshConfig.JumpPort)
}

func (sshConfig SSHConfig) addr() string {
	return fmt.Sprintf("%s:%d", sshConfig.Host, sshConfig.Port)
}
