package podman

import (
	args "arcaflow-engine-deployer-podman/args_builder"
	"arcaflow-engine-deployer-podman/cli_wrapper"
	"arcaflow-engine-deployer-podman/config"
	"bufio"
	"bytes"
	"github.com/docker/docker/api/types/container"
	"go.arcalot.io/log"
	"io"
	"os/exec"
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

// TODO: unwrap the whole config
func (p *CliPlugin) unwrapContainerConfig() container.Config {
	if p.config.Deployment.ContainerConfig != nil {
		return *p.config.Deployment.ContainerConfig
	} else {
		return container.Config{}
	}
}

func (p *CliPlugin) unwrapHostConfig() container.HostConfig {
	if p.config.Deployment.HostConfig != nil {
		return *p.config.Deployment.HostConfig
	} else {
		return container.HostConfig{}
	}
}

func (p *CliPlugin) Write(b []byte) (n int, err error) {
	p.lock.Lock()
	defer p.lock.Unlock()
	containerConfig := p.unwrapContainerConfig()
	hostConfig := p.unwrapHostConfig()
	commandArgs := []string{"run", "-i", "-a", "stdin", "-a", "stdout", "-a", "stderr"}
	args.NewBuilder(&commandArgs).
		SetContainerName(p.config.Podman.ContainerName).
		SetEnv(containerConfig.Env).
		SetVolumes(hostConfig.Binds).
		SetCgroupNs(p.config.Podman.CgroupNs)

	stdin, stdout, _, cmd, err := p.wrapper.Deploy(p.containerImage, p.config.Podman.ContainerName, commandArgs)

	if err != nil {
		return 0, err
	}
	writtenBytes, err := stdin.Write(b)
	if err != nil {
		return 0, err
	}

	stdoutBuf, err := p.readStdout(stdout)
	if err != nil {
		return 0, err
	}
	p.stdoutBuffer.Write(stdoutBuf)
	if err := cmd.Wait(); err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			return 0, exiterr
		}
		return 0, err
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
