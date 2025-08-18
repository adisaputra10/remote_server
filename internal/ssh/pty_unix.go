// +build !windows

package ssh

import (
	"os"
	"os/signal"
	"syscall"
	
	"golang.org/x/crypto/ssh"
)

func (c *PTYClient) setupSignalHandling() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for sig := range sigCh {
			switch sig {
			case syscall.SIGWINCH:
				c.handleResize()
			case syscall.SIGINT, syscall.SIGTERM:
				c.logSession("Received interrupt signal, closing session")
				if c.session != nil {
					c.session.Signal(ssh.SIGINT)
				}
			}
		}
	}()
}
