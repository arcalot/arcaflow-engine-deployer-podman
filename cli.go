package podman

import (
	"bufio"
	"bytes"
	"io"
	"sync"
)

//TODO: namespace setting
//volume mounts
//envvars
//networking

type Cli struct {
	stdoutBuffer   bytes.Buffer
	wrapper        Wrapper
	lock           *sync.Mutex
	containerImage string
	readIndex      int64
}

func (p *Cli) readStdout(r io.Reader) ([]byte, error) {
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

func (p *Cli) Write(b []byte) (n int, err error) {
	p.lock.Lock()
	defer p.lock.Unlock()
	in, out, cmd, err := p.wrapper.Deploy(p.containerImage)
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

func (p *Cli) Read(b []byte) (n int, err error) {
	if p.readIndex >= int64(len(p.stdoutBuffer.Bytes())) {
		err = io.EOF
		return
	}
	n = copy(b, p.stdoutBuffer.Bytes()[p.readIndex:])
	p.readIndex += int64(n)
	return
}

func (p *Cli) Close() error {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.stdoutBuffer.Reset()
	return nil
}
