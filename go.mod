module go.flow.arcalot.io/podmandeployer

go 1.21

require (
	github.com/docker/docker v25.0.4+incompatible
	github.com/docker/go-connections v0.5.0
	go.arcalot.io/assert v1.8.0
	go.arcalot.io/lang v1.1.0
	go.flow.arcalot.io/deployer v0.5.0
)

require (
	github.com/docker/go-units v0.5.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0 // indirect
	golang.org/x/sys v0.1.0 // indirect
	gotest.tools/v3 v3.5.1 // indirect
)

require (
	github.com/opencontainers/selinux v1.11.0
	go.arcalot.io/log/v2 v2.1.0
	go.flow.arcalot.io/pluginsdk v0.8.0
)
