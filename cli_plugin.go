package podman

import (
	"arcaflow-engine-deployer-podman/cli_wrapper"
	"arcaflow-engine-deployer-podman/config"
	"bytes"
	"go.arcalot.io/log"
	"io"
	"sync"
	"time"
)

type CliPlugin struct {
	stdoutBuffer   bytes.Buffer
	wrapper        cli_wrapper.CliWrapper
	lock           *sync.Mutex
	containerImage string
	readIndex      int64
	config         *config.Config
	logger         log.Logger
	stdin          io.WriteCloser
	stdout         io.ReadCloser
}

// TODO: unwrap the whole config

func (p *CliPlugin) Write(b []byte) (n int, err error) {

	writtenBytes, err := p.stdin.Write(b)
	if err != nil {
		return 0, err
	}
	time.Sleep(500 * time.Millisecond)
	return writtenBytes, nil
}

func (p *CliPlugin) Read(b []byte) (n int, err error) {
	p.lock.Lock()
	defer p.lock.Unlock()
	defer p.wrapper.ClearBuffer()
	buf := p.wrapper.GetStdoutData()

	if len(buf) == 0 {
		return 0, io.EOF
	}
	copy(b, buf)
	return len(b), nil
}

func (p *CliPlugin) Close() error {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.stdoutBuffer.Reset()
	p.stdout.Close()
	p.stdin.Close()
	return nil
}
