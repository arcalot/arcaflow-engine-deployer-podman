package podman

import (
	"bufio"
	"bytes"
	"github.com/docker/docker/api/types/container"
	"io"
	"sync"
)

//TODO: namespace setting
//volume mounts
//envvars
//networking

type CliPlugin struct {
	stdoutBuffer   bytes.Buffer
	wrapper        CliWrapper
	lock           *sync.Mutex
	containerImage string
	readIndex      int64
	config         *Config
}

func (p *CliPlugin) readStdout(r io.Reader) ([]byte, error) {
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
				return out, err
			}
		}
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return out, err
		}
	}
}

func (p *CliPlugin) unwrapContainerConfig() container.Config {
	if p.config.Deployment.ContainerConfig != nil {
		return *p.config.Deployment.ContainerConfig
	} else {
		return container.Config{}
	}
}

func (p *CliPlugin) Write(b []byte) (n int, err error) {
	p.lock.Lock()
	defer p.lock.Unlock()
	containerConfig := p.unwrapContainerConfig()
	in, out, cmd, err := p.wrapper.Deploy(p.containerImage, &containerConfig.Env)
	if err != nil {
		return 0, err
	}
	writtenBytes, err := in.Write(b)
	if err != nil {
		return 0, err
	}

	stdout, err := p.readStdout(out)
	if err != nil {
		return 0, err
	}
	p.stdoutBuffer.Write(stdout)
	if err := cmd.Wait(); err != nil {
		return 0, nil
	}
	return writtenBytes, nil
}

func (p *CliPlugin) Read(b []byte) (n int, err error) {
	if p.readIndex >= int64(len(p.stdoutBuffer.Bytes())) {
		err = io.EOF
		return
	}
	n = copy(b, p.stdoutBuffer.Bytes()[p.readIndex:])
	p.readIndex += int64(n)
	return
}

func (p *CliPlugin) Close() error {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.stdoutBuffer.Reset()
	return nil
}
