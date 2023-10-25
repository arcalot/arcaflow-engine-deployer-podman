module go.flow.arcalot.io/podmandeployer

go 1.18

require (
	github.com/docker/docker v24.0.6+incompatible
	github.com/docker/go-connections v0.4.0
	github.com/joho/godotenv v1.5.1
	go.arcalot.io/assert v1.6.0
	go.arcalot.io/lang v1.0.0
	go.flow.arcalot.io/deployer v0.3.0
)

require (
	github.com/docker/go-units v0.5.0 // indirect
	github.com/stretchr/testify v1.8.4 // indirect
	gotest.tools/v3 v3.5.1 // indirect
)

require (
	go.arcalot.io/log/v2 v2.0.0
	go.flow.arcalot.io/pluginsdk v0.5.0
)
