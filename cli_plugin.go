package podman

import (
	"arcaflow-engine-deployer-podman/cli_wrapper"
	"arcaflow-engine-deployer-podman/config"
	"bufio"
	"bytes"
	"go.arcalot.io/log"
	"io"
	"sync"
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

func (p *CliPlugin) readStdout(r io.Reader) {
	buffer := bytes.Buffer{}
	writer := bufio.NewWriter(&buffer)
	var out []byte
	buf := make([]byte, 1024, 1024)
	for {
		n, err := r.Read(buf[:])
		if n > 0 {
			d := buf[:n]
			out = append(out, d...)
			_, err := writer.Write(d)
			if err != nil {
				return
			}
		}
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return
		}
	}
}

// TODO: unwrap the whole config

func (p *CliPlugin) Write(b []byte) (n int, err error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	if err != nil {
		return 0, err
	}
	writtenBytes, err := p.stdin.Write(b)
	if err != nil {
		return 0, err
	}

	if err != nil {
		return 0, err
	}
	p.stdoutBuffer.Write(stdoutBuf)
	/*	if err := cmd.Wait(); err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			return 0, exiterr
		}
		return 0, err
	}*/
	return writtenBytes, nil
}

func (p *CliPlugin) _Write(b []byte) (n int, err error) {
	p.lock.Lock()
	defer p.lock.Unlock()
	buffer := new(bytes.Buffer)
	buffer.Read(b)
	_n, _err := io.Copy(p.stdin, buffer)
	err = _err
	n = int(_n)
	return
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
