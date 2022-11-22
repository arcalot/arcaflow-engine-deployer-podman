package podman

import (
	"context"
	"encoding/json"
	"go.arcalot.io/assert"
	"go.arcalot.io/log"
	"go.flow.arcalot.io/deployer"
	"io"
	"strings"
	"testing"
)

func getConnector(t *testing.T) deployer.Connector {
	configJSON := `
{
   "deployment":{
      "container":{
         "NetworkDisabled":true,
         "Env":{
            "ENV1":"TEST1",
            "ENV2":"TEST2"
         }
      }
   },
   "podman":{
      "path":"/usr/bin/podman"
   }
}
`
	var config any
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		t.Fatal(err)
	}

	factory := NewFactory()
	schema := factory.ConfigurationSchema()
	unserializedConfig, err := schema.UnserializeType(config)
	assert.NoError(t, err)
	connector, err := factory.Create(unserializedConfig, log.NewTestLogger(t))
	assert.NoError(t, err)
	return connector
}
func TestSimpleInOut(t *testing.T) {
	connector := getConnector(t)
	container, err := connector.Deploy(context.Background(), "quay.io/tsebastiani/arcaflow-engine-deployer-podman-test:latest")
	assert.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, container.Close())
	})

	var containerInput = []byte("input abc\n")
	assert.NoErrorR[int](t)(container.Write(containerInput))

	buf := new(strings.Builder)
	assert.NoErrorR[int64](t)(io.Copy(buf, container))
	assert.Contains(t, buf.String(), "This is what input was received: \"abc\"")
}
