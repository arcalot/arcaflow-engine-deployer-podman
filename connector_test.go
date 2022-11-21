package podman

import (
	"context"
	"encoding/json"
	"go.arcalot.io/assert"
	"go.arcalot.io/log"
	"io"
	"strings"
	"testing"
)

func TestSimpleInOut(t *testing.T) {
	configJSON := `
{
   "deployment":{

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

	container, err := connector.Deploy(context.Background(), "quay.io/joconnel/io-test-script")
	assert.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, container.Close())
	})

	var containerInput = []byte("abc\n")
	assert.NoErrorR[int](t)(container.Write(containerInput))

	buf := new(strings.Builder)
	assert.NoErrorR[int64](t)(io.Copy(buf, container))
	assert.Contains(t, buf.String(), "This is what input was received: \"abc\"")
}
