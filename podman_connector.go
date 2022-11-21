package podman

import (
	"arcaflow-engine-deployer-podman/wrapper"
	"bufio"
	"bytes"
	"io"
	"sync"
)

type PodmanConnector struct {
	ContainerStdOut bytes.Buffer
	Wrapper         wrapper.PodmanWrapper
	Lock            *sync.Mutex
	Image           string
	readIndex       int64
}

func (p *PodmanConnector) readStdout(w io.Writer, r io.Reader) ([]byte, error) {
	var out []byte
	buf := make([]byte, 1024, 1024)
	for {
		n, err := r.Read(buf[:])
		if n > 0 {
			d := buf[:n]
			out = append(out, d...)
			_, err := w.Write(d)
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

func (p *PodmanConnector) Write(b []byte) (n int, err error) {
	p.Lock.Lock()
	defer p.Lock.Unlock()
	in, out, cmd, err := p.Wrapper.Deploy(p.Image)
	if err != nil {
		return 0, err
	}
	writtenBytes, err := in.Write(b)
	if err != nil {
		return 0, err
	}
	buffer := bytes.Buffer{}
	writer := bufio.NewWriter(&buffer)

	res, err := p.readStdout(writer, out)
	if err != nil {
		return 0, err
	}
	p.ContainerStdOut.Write(res)
	if err := cmd.Wait(); err != nil {
		return 0, nil
	}
	return writtenBytes, nil
}

func (p *PodmanConnector) Read(b []byte) (n int, err error) {
	if p.readIndex >= int64(len(p.ContainerStdOut.Bytes())) {
		err = io.EOF
		return
	}
	n = copy(b, p.ContainerStdOut.Bytes()[p.readIndex:])
	p.readIndex += int64(n)
	return
}

func (p *PodmanConnector) Close() error {
	p.Lock.Lock()
	defer p.Lock.Unlock()
	//p.ContainerOut = nil
	return nil
}
