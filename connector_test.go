package podman

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"go.flow.arcalot.io/podmandeployer/internal/util"

	"go.arcalot.io/assert"
	log "go.arcalot.io/log/v2"
	"go.flow.arcalot.io/deployer"
	"go.flow.arcalot.io/podmandeployer/tests"
)

func getConnector(t *testing.T, configJSON string) (deployer.Connector, *Config) {
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
	return connector, unserializedConfig
}

var inOutConfig = `
{
   "podman":{
      "path":"/usr/bin/podman"
   }
}
`

func TestSimpleInOut(t *testing.T) {
	logger := log.NewTestLogger(t)
	pongStr := "pong abc"
	endStr := "end abc"

	connector, _ := getConnector(t, inOutConfig)
	plugin, err := connector.Deploy(context.Background(), "quay.io/tsebastiani/arcaflow-engine-deployer-podman-test:latest")
	assert.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, plugin.Close())
	})
	var containerInput = []byte("ping abc\n")
	assert.NoErrorR[int](t)(plugin.Write(containerInput))
	readBuffer := readOutputUntil(t, plugin, pongStr)
	if len(readBuffer) == 0 {
		t.Fatalf("expected string not found: %s", pongStr)
	}

	logger.Infof(string(readBuffer[:7]))

	assert.NoErrorR[int](t)(plugin.Write(containerInput))
	readBuffer = readOutputUntil(t, plugin, endStr)
	if len(readBuffer) == 0 {
		t.Fatalf("expected string not found: %s", endStr)
	}
}

var envConfig = `
{
   "deployment":{
      "container":{
         "NetworkDisabled":true,
         "Env":[
            "DEPLOYER_PODMAN_TEST_1=TEST1",
            "DEPLOYER_PODMAN_TEST_2=TEST2"
         ]
      }
   },
   "podman":{
      "path":"/usr/bin/podman"
   }
}
`

func TestEnv(t *testing.T) {
	envVars := "DEPLOYER_PODMAN_TEST_1=TEST1\nDEPLOYER_PODMAN_TEST_2=TEST2"
	connector, _ := getConnector(t, envConfig)
	container, err := connector.Deploy(context.Background(), "quay.io/tsebastiani/arcaflow-engine-deployer-podman-test:latest")
	assert.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, container.Close())
	})
	
	var containerInput = []byte("env\n")

	assert.NoErrorR[int](t)(container.Write(containerInput))

	readBuffer := readOutputUntil(t, container, envVars)
	if len(readBuffer) == 0 {
		t.Fatalf("expected string not found: %s", envVars)
	}
}

var volumeConfig = `
{
   "deployment":{
      "host":{
         "Binds":[
            "./tests/volume:/test"
         ]
      }
   },
   "podman":{
      "path":"/usr/bin/podman"
   }
}
`

func TestSimpleVolume(t *testing.T) {
	logger := log.NewTestLogger(t)
	fileContent, err := os.ReadFile("./tests/volume/test_file.txt")
	if err != nil {
		t.Fatalf(err.Error())
	}
	connector, _ := getConnector(t, volumeConfig)

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf(err.Error())
	}
	// disable selinux on the test folder in order to make the file readable from within the container
	cmd := exec.Command("chcon", "-Rt", "svirt_sandbox_file_t", fmt.Sprintf("%s/tests/volume", cwd)) //nolint:gosec
	err = cmd.Run()
	if err != nil {
		logger.Warningf("failed to set SELinux permissions on folder, chcon error: %s, this may cause test failure, let's see...", err.Error())
	}

	container, err := connector.Deploy(context.Background(), "quay.io/tsebastiani/arcaflow-engine-deployer-podman-test:latest")
	if err != nil {
		t.Fatalf(err.Error())
	}
	t.Cleanup(func() {
		assert.NoError(t, container.Close())
	})

	var containerInput = []byte("volume\n")

	_, err = container.Write(containerInput)
	if err != nil {
		t.Fatalf(err.Error())
	}

	readBuffer := readOutputUntil(t, container, string(fileContent))
	if len(readBuffer) == 0 {
		t.Fatalf("expected string not found: %s", string(fileContent))
	}
}

var nameTemplate = `
{
  "podman":{
     "path":"/usr/bin/podman",
     "ContainerNameRoot":"%s"
  }
}
`

var defaultTemplate = `
{
   "podman":{
      "path":"/usr/bin/podman"
   }
}
`

func TestContainerName(t *testing.T) {
	//logger := log.NewTestLogger(t)
	//ContainerNameRoot := fmt.Sprintf("test_%s", util.GetRandomString(5))
	//configTemplate := fmt.Sprintf(nameTemplate, ContainerNameRoot)
	//seed := int64(1)
	//rng := *rand.New(rand.NewSource(seed))
	configTemplate := defaultTemplate
	ctx := context.Background()
	connector, _ := getConnector(t, configTemplate)

	container, err := connector.Deploy(
		ctx,
		"quay.io/tsebastiani/arcaflow-engine-deployer-podman-test:latest")
	assert.NoError(t, err)
	//containerName1 := container.ContainerNameRoot

	container2, err := connector.Deploy(
		ctx,
		"quay.io/tsebastiani/arcaflow-engine-deployer-podman-test:latest")
	assert.NoError(t, err)
	//containerName2 := container2.ContainerNameRoot

	t.Cleanup(func() {
		assert.NoError(t, container.Close())
		assert.NoError(t, container2.Close())
	})

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		var containerInput = []byte("sleep 3\n")
		assert.NoErrorR[int](t)(container.Write(containerInput))
		assert.NoErrorR[int](t)(container2.Write(containerInput))
	}()
	//time.Sleep(1 * time.Second)
	//if tests.IsContainerRunning(logger, config.Podman.Path, containerNameRoot) == false {
	//	t.Fatalf("container with name %s not found", containerNameRoot)
	//}

	wg.Wait()
}

var cgroupTemplate = `
{
   "podman":{
      "path":"/usr/bin/podman",
      "ContainerNameRoot":"%s",
      "cgroupNs":"%s"
   }
}
`

func TestCgroupNsByContainerName(t *testing.T) {
	if tests.IsRunningOnGithub() {
		t.Skipf("joining another container cgroup namespace by container name not supported on GitHub actions")
	}
	logger := log.NewTestLogger(t)
	seed := int64(1)
	rng := *rand.New(rand.NewSource(seed))
	containername1 := fmt.Sprintf("test%s", util.GetRandomString(&rng, 5))
	// The first container will run with a private namespace that will be created at startup
	configtemplate1 := fmt.Sprintf(cgroupTemplate, containername1, "private")
	connector1, config := getConnector(t, configtemplate1)
	container1, err := connector1.Deploy(context.Background(), "quay.io/tsebastiani/arcaflow-engine-deployer-podman-test:latest")
	if err != nil {
		t.Fatalf(err.Error())
	}
	containername2 := fmt.Sprintf("test%s", util.GetRandomString(&rng, 5))
	// The second one will join the newly created private namespace of the first container
	configtemplate2 := fmt.Sprintf(cgroupTemplate, containername2, fmt.Sprintf("container:%s", containername1))
	connector2, _ := getConnector(t, configtemplate2)
	container2, err := connector2.Deploy(context.Background(), "quay.io/tsebastiani/arcaflow-engine-deployer-podman-test:latest")
	if err != nil {
		t.Fatalf(err.Error())
	}
	t.Cleanup(func() {
		assert.NoError(t, container1.Close())
		assert.NoError(t, container2.Close())
	})

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		var containerInput = []byte("sleep 7\n")
		_, err := container1.Write(containerInput)
		if err != nil {
			t.Errorf(err.Error())
			return
		}
	}()
	// sleeps to wait the first container become ready and attach to its cgroup ns
	time.Sleep(2 * time.Second)
	wg.Add(1)
	go func() {
		defer wg.Done()
		var containerInput = []byte("sleep 5\n")
		_, err := container2.Write(containerInput)
		if err != nil {
			t.Errorf(err.Error())
			return
		}
	}()
	ns1 := tests.GetPodmanPsNsWithFormat(logger, config.Podman.Path, containername1, "{{.CGROUPNS}}")
	ns2 := tests.GetPodmanPsNsWithFormat(logger, config.Podman.Path, containername2, "{{.CGROUPNS}}")
	if ns1 != ns2 {
		t.Errorf("namespace1: %s and namespace2: %s do not match, failing", ns1, ns2)
	} else {
		logger.Debugf("container 1 namespace: %s, container 2 namespace: %s, they're the same!", ns1, ns2)
	}
	wg.Wait()
}

func TestPrivateCgroupNs(t *testing.T) {
	// get the user cgroup ns
	logger := log.NewTestLogger(t)
	seed := int64(1)
	rng := *rand.New(rand.NewSource(seed))
	var wg sync.WaitGroup
	userCgroupNs := tests.GetCommmandCgroupNs(logger, "/usr/bin/sleep", []string{"3"})
	assert.NotNil(t, userCgroupNs)
	logger.Debugf("Detected cgroup namespace for user: %s", userCgroupNs)

	containername := fmt.Sprintf("test%s", util.GetRandomString(&rng, 5))
	// The first container will run with a private namespace that will be created at startup
	configtemplate := fmt.Sprintf(cgroupTemplate, containername, "private")
	connector, config := getConnector(t, configtemplate)
	container, err := connector.Deploy(context.Background(), "quay.io/tsebastiani/arcaflow-engine-deployer-podman-test:latest")
	assert.NoError(t, err)

	t.Cleanup(func() {
		assert.NoError(t, container.Close())
	})

	wg.Add(1)
	go func() {
		defer wg.Done()
		var containerInput = []byte("sleep 5\n")
		assert.NoErrorR[int](t)(container.Write(containerInput))
	}()

	time.Sleep(2 * time.Second)

	var podmanCgroupNs = tests.GetPodmanCgroupNs(logger, config.Podman.Path, containername)
	wg.Wait()

	// if the user's namespace is equal to the podman one the test must fail
	if userCgroupNs == podmanCgroupNs {
		t.Fail()
	} else {
		logger.Debugf("user cgroup namespace: %s, podman private cgroup namespace: %s, they're different!", userCgroupNs, podmanCgroupNs)
	}
}

func TestHostCgroupNs(t *testing.T) {
	logger := log.NewTestLogger(t)
	seed := int64(1)
	rng := *rand.New(rand.NewSource(seed))
	var wg sync.WaitGroup

	userCgroupNs := tests.GetCommmandCgroupNs(logger, "/usr/bin/sleep", []string{"3"})
	assert.NotNil(t, userCgroupNs)

	logger.Debugf("Detected cgroup namespace for user: %s", userCgroupNs)
	containername := fmt.Sprintf("test%s", util.GetRandomString(&rng, 5))
	// The first container will run with the host namespace
	configtemplate := fmt.Sprintf(cgroupTemplate, containername, "host")
	connector, config := getConnector(t, configtemplate)
	container, err := connector.Deploy(context.Background(), "quay.io/tsebastiani/arcaflow-engine-deployer-podman-test:latest")
	assert.NoError(t, err)

	t.Cleanup(func() {
		assert.NoError(t, container.Close())
	})

	wg.Add(1)
	go func() {
		defer wg.Done()
		var containerInput = []byte("sleep 5\n")
		assert.NoErrorR[int](t)(container.Write(containerInput))
	}()
	wg.Wait()
	// waits for the container to become ready
	time.Sleep(2 * time.Second)

	var podmanCgroupNs = tests.GetPodmanCgroupNs(logger, config.Podman.Path, containername)
	assert.NotNil(t, podmanCgroupNs)
	wg.Wait()
	// if the container is running in a different cgroup namespace than the user the test must fail
	if userCgroupNs != podmanCgroupNs {
		t.Fail()
	} else {
		logger.Debugf("user cgroup namespace: %s, podman cgroup namespace: %s, the same!", userCgroupNs, podmanCgroupNs)
	}
}

func TestCgroupNsByNamespacePath(t *testing.T) {
	if tests.IsRunningOnGithub() {
		t.Skipf("joining another container cgroup namespace by namespace path ns:/proc/<PID>/ns/cgroup not supported on GitHub actions")
	}
	logger := log.NewTestLogger(t)
	seed := int64(1)
	rng := *rand.New(rand.NewSource(seed))
	containername1 := fmt.Sprintf("test%s", util.GetRandomString(&rng, 5))
	// The first container will run with a private namespace that will be created at startup
	configtemplate1 := fmt.Sprintf(cgroupTemplate, containername1, "private")
	connector1, config := getConnector(t, configtemplate1)
	container1, err := connector1.Deploy(context.Background(), "quay.io/tsebastiani/arcaflow-engine-deployer-podman-test:latest")
	assert.NoError(t, err)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		var containerInput = []byte("sleep 10\n")
		if _, err := container1.Write(containerInput); err != nil {
			t.Errorf(err.Error())
			return
		}
	}()
	// sleeps to wait the first container become ready and attach to its cgroup ns
	time.Sleep(2 * time.Second)

	pid := tests.GetPodmanPsNsWithFormat(logger, config.Podman.Path, containername1, "{{.Pid}}")

	containername2 := fmt.Sprintf("test%s", util.GetRandomString(&rng, 5))
	// The second one will join the newly created private namespace of the first container
	namespacePath := fmt.Sprintf("ns:/proc/%s/ns/cgroup", pid)
	configtemplate2 := fmt.Sprintf(cgroupTemplate, containername2, namespacePath)
	connector2, _ := getConnector(t, configtemplate2)

	container2, err := connector2.Deploy(context.Background(), "quay.io/tsebastiani/arcaflow-engine-deployer-podman-test:latest")
	if err != nil {
		t.Fatalf(err.Error())
	}

	t.Cleanup(func() {
		assert.NoError(t, container1.Close())
		assert.NoError(t, container2.Close())
	})

	wg.Add(1)
	go func() {
		defer wg.Done()
		var containerInput = []byte("sleep 5\n")
		if _, err := container2.Write(containerInput); err != nil {
			t.Errorf(err.Error())
			return
		}
	}()

	ns1 := tests.GetPodmanPsNsWithFormat(logger, config.Podman.Path, containername1, "{{.CGROUPNS}}")
	ns2 := tests.GetPodmanPsNsWithFormat(logger, config.Podman.Path, containername2, "{{.CGROUPNS}}")
	if ns1 != ns2 {
		t.Errorf("namespace1: %s and namespace2: %s do not match, failing", ns1, ns2)
	} else {
		logger.Debugf("container 1 namespace: %s, container 2 namespace: %s, they're the same!", ns1, ns2)
		logger.Debugf("Container 2 joined the namespace via namespace path: %s", namespacePath)
	}
	wg.Wait()
}

var networkTemplate = `
{
   "podman":{
      "ContainerNameRoot":"%s",
      "path":"/usr/bin/podman",
      "networkMode":"%s"
   }
}
`

func TestNetworkHost(t *testing.T) {
	logger := log.NewTestLogger(t)
	seed := int64(1)
	rng := *rand.New(rand.NewSource(seed))
	containername := fmt.Sprintf("test%s", util.GetRandomString(&rng, 5))
	// The first container will run with the host namespace
	configtemplate := fmt.Sprintf(networkTemplate, containername, "host")
	connector, _ := getConnector(t, configtemplate)
	plugin, err := connector.Deploy(context.Background(), "quay.io/tsebastiani/arcaflow-engine-deployer-podman-test:latest")
	assert.NoError(t, err)

	t.Cleanup(func() {
		assert.NoError(t, plugin.Close())
	})
	var containerInput = []byte("network host\n")
	// the test script will run "ifconfig" in the container
	assert.NoErrorR[int](t)(plugin.Write(containerInput))
	var ifconfigOut bytes.Buffer
	// runs ifconfig in the host machine in order to check if the container has exactly the same network configuration
	cmd := exec.Command("/bin/bash", "-c", "ifconfig | grep -P \"^.+:\\s+.+$\" | awk '{ gsub(\":\",\"\");print $1 }'")
	cmd.Stdout = &ifconfigOut
	err = cmd.Run()
	assert.Nil(t, err)
	ifconfigOutStr := ifconfigOut.String()
	logger.Infof(ifconfigOutStr)
	readBuffer := readOutputUntil(t, plugin, ifconfigOutStr)
	if len(readBuffer) == 0 {
		t.Fatal("the container did not produce any output")
	}
	containerOutString := string(readBuffer)
	if strings.Contains(containerOutString, ifconfigOutStr) == false {
		t.Fatalf("expected string not found: %s", ifconfigOutStr)
	}
}

func TestNetworkBridge(t *testing.T) {
	// forces the container to have the following
	// network settings:
	// ip 10.88.0.123
	// mac 44:33:22:11:00:99
	// then asks to the container to run an ifconfig (tests/test_script.sh, test_network())
	// through ATP to check if the settings have been effectively accepted
	if tests.IsRunningOnGithub() {
		t.Skipf("bridge networking not supported on GitHub actions")
	}
	ip := "10.88.0.123"
	mac := "44:33:22:11:00:99"

	testNetworking(
		t,
		"bridge:ip=10.88.0.123,mac=44:33:22:11:00:99,interface_name=testif0",
		"network bridge\n",
		nil,
		&ip,
		&mac,
	)
}
func TestNetworkNone(t *testing.T) {
	expectedOutput := "1;lo"
	testNetworking(t, "none", "network none\n", &expectedOutput, nil, nil)
}

func TestClose(t *testing.T) {
	seed := int64(1)
	rng := *rand.New(rand.NewSource(seed))
	containerName := fmt.Sprintf("test_%s", util.GetRandomString(&rng, 5))
	configTemplate := fmt.Sprintf(nameTemplate, containerName)
	connector, _ := getConnector(t, configTemplate)

	container, err := connector.Deploy(context.Background(), "quay.io/tsebastiani/arcaflow-engine-deployer-podman-test:latest")
	assert.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		var containerInput = []byte("sleep 10\n")
		assert.NoErrorR[int](t)(container.Write(containerInput))
	}()

	time.Sleep(2 * time.Second)
	err = container.Close()
	assert.Nil(t, err)
}

// readOutputUntil helper function, reads from plugin (io.Reader) until finds lookforOutput
func readOutputUntil(t *testing.T, plugin io.Reader, lookForOutput string) []byte {
	var n int
	readBuffer := make([]byte, 10240)
	for {
		currentBuffer := make([]byte, 1024)
		readBytes, err := plugin.Read(currentBuffer)
		if err != nil {
			if err != io.EOF {
				t.Fatalf("error while reading stdout: %s", err.Error())
			} else {
				return readBuffer[:n]
			}
		}
		copy(readBuffer[n:], currentBuffer[:readBytes])
		n += readBytes
		if strings.Contains(string(readBuffer[:n]), lookForOutput) {
			return readBuffer[:n]
		}
	}
}

func checkIfconfig(t *testing.T) string {
	path, err := exec.LookPath("ifconfig")
	if err != nil {
		t.Fatalf("impossible to run test: %s , ifconfig not installed, skipping.", t.Name())
	}
	return path
}

func testNetworking(t *testing.T, podmanNetworking string, containerTest string, expectedOutput *string, ip *string, mac *string) {
	logger := log.NewTestLogger(t)
	checkIfconfig(t)
	seed := int64(1)
	rng := *rand.New(rand.NewSource(seed))
	containername := fmt.Sprintf("test%s", util.GetRandomString(&rng, 5))
	// The first container will run with the host namespace
	configtemplate := fmt.Sprintf(networkTemplate, containername, podmanNetworking)
	connector, _ := getConnector(t, configtemplate)
	plugin, err := connector.Deploy(context.Background(), "quay.io/tsebastiani/arcaflow-engine-deployer-podman-test:latest")
	if err != nil {
		t.Fatalf(err.Error())
	}

	t.Cleanup(func() {
		assert.NoError(t, plugin.Close())
	})
	var containerInput = []byte(containerTest)
	// the test script will output a string containing the desired ip address and mac address filtered by the desired interface name
	if _, err := plugin.Write(containerInput); err != nil {
		t.Fatalf(err.Error())
	}
	var readBuffer []byte
	if expectedOutput != nil {
		// in the networking none the token is exactly the output of ifconfig
		readBuffer = readOutputUntil(t, plugin, *expectedOutput)
	} else if mac != nil {
		// if an ip is passed instead the output contains the ipv6 interface ID as well so
		// the output is read until the mac address that is the last token in the ifconfig output.
		readBuffer = readOutputUntil(t, plugin, *mac)
	}
	logger.Infof(string(readBuffer))
	if len(readBuffer) == 0 {
		t.Fatalf("the container did not produce any output")
	}
	if ip == nil && mac == nil && expectedOutput != nil {
		if strings.Contains(string(readBuffer), *expectedOutput) == false {
			t.Fatalf("expected string not found: %s, %s found instead", *expectedOutput, string(readBuffer))
		}
	} else {
		if strings.Contains(string(readBuffer), *ip) == false {
			t.Fatalf("expected string not found: %s, %s found instead", *ip, string(readBuffer))
		}
		if strings.Contains(string(readBuffer), *mac) == false {
			t.Fatalf("expected string not found: %s, %s found instead", *mac, string(readBuffer))
		}
	}
}
