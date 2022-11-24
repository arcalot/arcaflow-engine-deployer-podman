package podman

import (
	"arcaflow-engine-deployer-podman/cli_wrapper"
	"arcaflow-engine-deployer-podman/config"
	"bufio"
	"bytes"
	"fmt"
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

func (p *CliPlugin) _Write(b []byte) (n int, err error) {
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
	//p.stdoutBuffer.Write(stdoutBuf)
	/*	if err := cmd.Wait(); err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			return 0, exiterr
		}
		return 0, err
	}*/
	return writtenBytes, nil
}

func (p *CliPlugin) Write(b []byte) (n int, err error) {
	p.lock.Lock()
	defer p.lock.Unlock()
	buffer := new(bytes.Buffer)
	buffer.Read(b)
	_n, _err := io.Copy(p.stdin, buffer)
	err = _err
	n = int(_n)
	return
}

func (p *CliPlugin) _Read(b []byte) (n int, err error) {
	if p.readIndex >= int64(len(p.stdoutBuffer.Bytes())) {
		err = io.EOF
		return
	}
	n = copy(b, p.stdoutBuffer.Bytes()[p.readIndex:])
	p.readIndex += int64(n)
	return
}

func (p *CliPlugin) Read(b []byte) (n int, err error) {
	p.lock.Lock()
	defer p.lock.Unlock()
	reader := bufio.NewReader(p.stdout)
	go p.readStdout(reader)
	return 0, nil

}

func handleReader(reader *bufio.Reader) {
	for {
		str, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		fmt.Print(str)

	}
}

func (p *CliPlugin) Close() error {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.stdoutBuffer.Reset()
	p.stdout.Close()
	p.stdin.Close()
	return nil
}
