package podman

import (
	"arcaflow-engine-deployer-podman/wrapper"
	"sync"
)

type PodmanConnector struct {
	ContainerOut []byte
	Wrapper      wrapper.PodmanWrapper
	Lock         *sync.Mutex
	Config       *Config
}

func (p *PodmanConnector) Write(b []byte) (n int, err error) {
	/*p.Lock.Lock()
	defer p.Lock.Unlock()*/
	in, out, cmd, err := p.Wrapper.Deploy(p.Config.Deployment.ContainerConfig.Image)
	if err != nil {
		return 0, err
	}
	writtenBytes, err := in.Write(b)
	if err != nil {
		return 0, err
	}

	if err := cmd.Wait(); err != nil {
		return 0, err
	}
	out.Read(p.ContainerOut)
	return writtenBytes, nil
}

func (p *PodmanConnector) Read(b []byte) (n int, err error) {
	p.Lock.Lock()
	defer p.Lock.Unlock()
	copy(b, p.ContainerOut)
	return len(b), nil
}

func (p *PodmanConnector) Close() error {
	p.Lock.Lock()
	defer p.Lock.Unlock()
	p.ContainerOut = nil
	return nil
}
