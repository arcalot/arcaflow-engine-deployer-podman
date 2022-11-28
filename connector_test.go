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

	"arcaflow-engine-deployer-podman/tests"
	"go.arcalot.io/assert"
	"go.arcalot.io/log"
	"go.flow.arcalot.io/deployer"
)

func getTestRandomString(n int) string {
	var seededRand *rand.Rand = rand.New(
		rand.NewSource(time.Now().UnixNano()))
	const charset = "abcdefghijklmnopqrstuvwxyz" +
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func getConnector(t *testing.T, configJson string) (deployer.Connector, *Config) {
	var config any
	if err := json.Unmarshal([]byte(configJson), &config); err != nil {
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
		t.Fatal(fmt.Sprintf("expected string not found: %s", pongStr))
	}
	fmt.Println(string(readBuffer[:7]))

	assert.NoErrorR[int](t)(plugin.Write(containerInput))
	readBuffer = readOutputUntil(t, plugin, endStr)
	if len(readBuffer) == 0 {
		t.Fatal(fmt.Sprintf("expected string not found: %s", endStr))
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
		t.Fatal(fmt.Sprintf("expected string not found: %s", envVars))
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
	fileContent, err := os.ReadFile("./tests/volume/test_file.txt")
	assert.Nil(t, err)
	connector, _ := getConnector(t, volumeConfig)

	cwd, err := os.Getwd()
	assert.Nil(t, err)
	// disable selinux on the test folder in order to make the file readable from within the container
	cmd := exec.Command("chcon", "-Rt", "svirt_sandbox_file_t", fmt.Sprintf("%s/tests/volume", cwd))
	err = cmd.Run()
	assert.Nil(t, err)

	container, err := connector.Deploy(context.Background(), "quay.io/tsebastiani/arcaflow-engine-deployer-podman-test:latest")
	assert.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, container.Close())
	})

	var containerInput = []byte("volume\n")

	assert.NoErrorR[int](t)(container.Write(containerInput))

	readBuffer := readOutputUntil(t, container, string(fileContent))
	if len(readBuffer) == 0 {
		t.Fatal(fmt.Sprintf("expected string not found: %s", string(fileContent)))
	}
}

var nameTemplate = `
{
   "podman":{
      "path":"/usr/bin/podman",
      "containerName":"%s"
   }
}
`

func TestContainerName(t *testing.T) {
	containerName := fmt.Sprintf("test_%s", getTestRandomString(5))
	configTemplate := fmt.Sprintf(nameTemplate, containerName)
	connector, config := getConnector(t, configTemplate)

	container, err := connector.Deploy(context.Background(), "quay.io/tsebastiani/arcaflow-engine-deployer-podman-test:latest")
	assert.NoError(t, err)

	t.Cleanup(func() {
		cmd := exec.Command(config.Podman.Path, "container", "rm", containerName)
		cmd.Run()
		assert.NoError(t, container.Close())
	})

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		var containerInput = []byte("sleep 3\n")
		assert.NoErrorR[int](t)(container.Write(containerInput))
	}()

	var stdout bytes.Buffer
	cmd := exec.Command(config.Podman.Path, "ps", "--filter", fmt.Sprintf("name=%s", containerName), "--format", "{{.ID}}")
	cmd.Stdout = &stdout
	cmd.Run()
	stdoutStr := stdout.String()
	assert.NotNil(t, stdoutStr)
	wg.Wait()

}

var cgroupTemplate = `
{
   "podman":{
      "path":"/usr/bin/podman",
      "containerName":"%s",
      "cgroupNs":"%s"
   }
}
`

func TestContainerCgroupNs(t *testing.T) {
	log := log.NewTestLogger(t)
	containername1 := fmt.Sprintf("test%s", getTestRandomString(5))
	// The first container will run with a private namespace that will be created at startup
	configtemplate1 := fmt.Sprintf(cgroupTemplate, containername1, "private")
	connector1, config := getConnector(t, configtemplate1)

	container1, err := connector1.Deploy(context.Background(), "quay.io/tsebastiani/arcaflow-engine-deployer-podman-test:latest")
	assert.NoError(t, err)

	containername2 := fmt.Sprintf("test%s", getTestRandomString(5))
	// The second one will join the newly created private namespace of the first container
	configtemplate2 := fmt.Sprintf(cgroupTemplate, containername2, fmt.Sprintf("container:%s", containername1))
	connector2, _ := getConnector(t, configtemplate2)

	container2, err := connector2.Deploy(context.Background(), "quay.io/tsebastiani/arcaflow-engine-deployer-podman-test:latest")
	assert.NoError(t, err)

	t.Cleanup(func() {
		cmd := exec.Command(config.Podman.Path, "container", "rm", containername1)
		cmd.Run()
		cmd = exec.Command(config.Podman.Path, "container", "rm", containername2)
		cmd.Run()
		assert.NoError(t, container1.Close())
		assert.NoError(t, container2.Close())
	})

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		var containerInput = []byte("sleep 7\n")
		assert.NoErrorR[int](t)(container1.Write(containerInput))
	}()
	// sleeps to wait the first container become ready and attach to its cgroup ns
	time.Sleep(2 * time.Second)

	wg.Add(1)
	go func() {
		defer wg.Done()
		var containerInput = []byte("sleep 5\n")
		assert.NoErrorR[int](t)(container2.Write(containerInput))
	}()

	var stdoutContainer1 bytes.Buffer
	var stdoutContainer2 bytes.Buffer

	cmd1 := exec.Command(config.Podman.Path, "ps", "--ns", "--filter", fmt.Sprintf("name=%s", containername1), "--format", "{{.CGROUPNS}}")
	cmd1.Stdout = &stdoutContainer1
	cmd1.Run()

	cmd2 := exec.Command(config.Podman.Path, "ps", "--ns", "--filter", fmt.Sprintf("name=%s", containername2), "--format", "{{.CGROUPNS}}")
	cmd2.Stdout = &stdoutContainer2
	cmd2.Run()
	// check that both the container are running in the same namespace
	ns1 := strings.TrimSuffix(stdoutContainer1.String(), "\n")
	ns2 := strings.TrimSuffix(stdoutContainer2.String(), "\n")
	if ns1 != ns2 {
		t.Fail()
	} else {
		log.Debugf("container 1 namespace: %s, container 2 namespace: %s, they're the same!", ns1, ns2)
	}
	wg.Wait()

}

func TestPrivateCgroupNs(t *testing.T) {
	// get the user cgroup ns
	log := log.NewTestLogger(t)
	var wg sync.WaitGroup
	userCgroupNs := tests.GetCommmandCgroupNs("/usr/bin/sleep", []string{"3"})
	assert.NotNil(t, userCgroupNs)
	log.Debugf("Detected cgroup namespace for user: %s", userCgroupNs)

	containername := fmt.Sprintf("test%s", getTestRandomString(5))
	// The first container will run with a private namespace that will be created at startup
	configtemplate := fmt.Sprintf(cgroupTemplate, containername, "private")
	connector, config := getConnector(t, configtemplate)
	container, err := connector.Deploy(context.Background(), "quay.io/tsebastiani/arcaflow-engine-deployer-podman-test:latest")
	assert.NoError(t, err)

	t.Cleanup(func() {
		cmd := exec.Command(config.Podman.Path, "container", "rm", containername)
		cmd.Run()
		assert.NoError(t, container.Close())

	})

	wg.Add(1)
	go func() {
		defer wg.Done()
		var containerInput = []byte("sleep 5\n")
		assert.NoErrorR[int](t)(container.Write(containerInput))
	}()

	time.Sleep(2 * time.Second)

	var podmanCgroupNs = tests.GetPodmanCgroupNs(config.Podman.Path, containername)
	wg.Wait()

	// if the user's namespace is equal to the podman one the test must fail
	if userCgroupNs == podmanCgroupNs {
		t.Fail()
	} else {
		log.Debugf("user cgroup namespace: %s, podman private cgroup namespace: %s, they're different!", userCgroupNs, podmanCgroupNs)
	}
}

func TestHostCgroupNs(t *testing.T) {
	// get the user cgroup ns
	log := log.NewTestLogger(t)
	var wg sync.WaitGroup

	userCgroupNs := tests.GetCommmandCgroupNs("/usr/bin/sleep", []string{"3"})
	assert.NotNil(t, userCgroupNs)

	log.Debugf("Detected cgroup namespace for user: %s", userCgroupNs)
	containername := fmt.Sprintf("test%s", getTestRandomString(5))
	// The first container will run with the host namespace
	configtemplate := fmt.Sprintf(cgroupTemplate, containername, "host")
	connector, config := getConnector(t, configtemplate)
	container, err := connector.Deploy(context.Background(), "quay.io/tsebastiani/arcaflow-engine-deployer-podman-test:latest")
	assert.NoError(t, err)

	t.Cleanup(func() {
		cmd := exec.Command(config.Podman.Path, "container", "rm", containername)
		cmd.Run()
		assert.NoError(t, container.Close())

	})

	wg.Add(1)
	go func() {
		defer wg.Done()
		var containerInput = []byte("sleep 5\n")
		assert.NoErrorR[int](t)(container.Write(containerInput))
	}()
	// waits for the container to become ready
	time.Sleep(2 * time.Second)

	var podmanCgroupNs = tests.GetPodmanCgroupNs(config.Podman.Path, containername)
	assert.NotNil(t, podmanCgroupNs)
	wg.Wait()
	// if the container is running in a different cgroup namespace than the user the test must fail
	if userCgroupNs != podmanCgroupNs {
		t.Fail()
	} else {
		log.Debugf("user cgroup namespace: %s, podman cgroup namespace: %s, the same!", userCgroupNs, podmanCgroupNs)
	}
}

func TestNamespacePathCgroupNs(t *testing.T) {
	log := log.NewTestLogger(t)
	containername1 := fmt.Sprintf("test%s", getTestRandomString(5))
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
		assert.NoErrorR[int](t)(container1.Write(containerInput))
	}()
	// sleeps to wait the first container become ready and attach to its cgroup ns
	time.Sleep(2 * time.Second)

	var stdoutPid bytes.Buffer
	cmdGetPid := exec.Command(config.Podman.Path, "ps", "--ns", "--filter", fmt.Sprintf("name=%s", containername1), "--format", "{{.Pid}}")
	cmdGetPid.Stdout = &stdoutPid
	cmdGetPid.Run()

	containername2 := fmt.Sprintf("test%s", getTestRandomString(5))
	// The second one will join the newly created private namespace of the first container
	namespacePath := fmt.Sprintf("ns:/proc/%s/ns/cgroup", strings.TrimSuffix(stdoutPid.String(), "\n"))
	configtemplate2 := fmt.Sprintf(cgroupTemplate, containername2, namespacePath)
	connector2, _ := getConnector(t, configtemplate2)

	container2, err := connector2.Deploy(context.Background(), "quay.io/tsebastiani/arcaflow-engine-deployer-podman-test:latest")
	assert.NoError(t, err)

	t.Cleanup(func() {
		cmd := exec.Command(config.Podman.Path, "container", "rm", containername1)
		cmd.Run()
		cmd = exec.Command(config.Podman.Path, "container", "rm", containername2)
		cmd.Run()
		assert.NoError(t, container1.Close())
		assert.NoError(t, container2.Close())
	})

	wg.Add(1)
	go func() {
		defer wg.Done()
		var containerInput = []byte("sleep 5\n")
		assert.NoErrorR[int](t)(container2.Write(containerInput))
	}()

	var stdoutContainer1 bytes.Buffer
	var stdoutContainer2 bytes.Buffer

	cmd1 := exec.Command(config.Podman.Path, "ps", "--ns", "--filter", fmt.Sprintf("name=%s", containername1), "--format", "{{.CGROUPNS}}")
	cmd1.Stdout = &stdoutContainer1
	cmd1.Run()

	cmd2 := exec.Command(config.Podman.Path, "ps", "--ns", "--filter", fmt.Sprintf("name=%s", containername2), "--format", "{{.CGROUPNS}}")
	cmd2.Stdout = &stdoutContainer2
	cmd2.Run()
	// check that both the container are running in the same namespace
	ns1 := strings.TrimSuffix(stdoutContainer1.String(), "\n")
	ns2 := strings.TrimSuffix(stdoutContainer2.String(), "\n")
	if ns1 != ns2 {
		t.Fail()
	} else {
		log.Debugf("container 1 namespace: %s, container 2 namespace: %s, they're the same!", ns1, ns2)
		log.Debugf("Container 2 joined the namespace via namespace path: %s", namespacePath)
	}
	wg.Wait()

}

var networkTemplate = `
{
   "podman":{
      "containerName":"%s",
      "path":"/usr/bin/podman",
      "networkMode":"%s"
   }
}
`

func TestNetworkHost(t *testing.T) {
	// get the user cgroup ns
	ifconfig := checkIfconfig(t)

	containername := fmt.Sprintf("test%s", getTestRandomString(5))
	// The first container will run with the host namespace
	configtemplate := fmt.Sprintf(networkTemplate, containername, "host")
	connector, config := getConnector(t, configtemplate)
	plugin, err := connector.Deploy(context.Background(), "quay.io/tsebastiani/arcaflow-engine-deployer-podman-test:latest")
	assert.NoError(t, err)

	t.Cleanup(func() {
		cmd := exec.Command(config.Podman.Path, "container", "rm", containername)
		cmd.Run()
		assert.NoError(t, plugin.Close())
	})
	var containerInput = []byte("network host\n")
	//the test script will run "ifconfig" in the container
	assert.NoErrorR[int](t)(plugin.Write(containerInput))
	var ifconfigOut bytes.Buffer
	//runs ifconfig in the host machine in order to check if the container has exactly the same network configuration
	cmd := exec.Command(ifconfig)
	cmd.Stdout = &ifconfigOut
	cmd.Run()

	readBuffer := readOutputUntil(t, plugin, ifconfigOut.String())
	if len(readBuffer) == 0 {
		t.Fatal(fmt.Sprintf("the container did not produce any output"))
	}
	if strings.Contains(string(readBuffer), ifconfigOut.String()) == false {
		t.Fatal(fmt.Sprintf("expected string not found: %s", ifconfigOut.String()))
	}

}

func TestNetworkBridge(t *testing.T) {
	log := log.NewTestLogger(t)
	checkIfconfig(t)
	containername := fmt.Sprintf("test%s", getTestRandomString(5))
	// The first container will run with the host namespace
	configtemplate := fmt.Sprintf(networkTemplate, containername, "bridge:ip=10.88.0.123,mac=44:33:22:11:00:99,interface_name=testif0")
	connector, config := getConnector(t, configtemplate)
	plugin, err := connector.Deploy(context.Background(), "quay.io/tsebastiani/arcaflow-engine-deployer-podman-test:latest")
	assert.NoError(t, err)

	t.Cleanup(func() {
		cmd := exec.Command(config.Podman.Path, "container", "rm", containername)
		cmd.Run()
		assert.NoError(t, plugin.Close())
	})
	var containerInput = []byte("network bridge\n")
	//the test script will output a string containing the desired ip address and mac address filtered by the desired interface name
	assert.NoErrorR[int](t)(plugin.Write(containerInput))
	expectedOutput := "10.88.0.123;44:33:22:11:00:99"

	readBuffer := readOutputUntil(t, plugin, expectedOutput)
	log.Infof(string(readBuffer))
	if len(readBuffer) == 0 {
		t.Fatal(fmt.Sprintf("the container did not produce any output"))
	}
	if strings.Contains(string(readBuffer), expectedOutput) == false {
		t.Fatal(fmt.Sprintf("expected string not found: %s", expectedOutput))
	}

}
func TestNetworkNone(t *testing.T) {
	log := log.NewTestLogger(t)
	checkIfconfig(t)
	containername := fmt.Sprintf("test%s", getTestRandomString(5))
	// The first container will run with the host namespace
	configtemplate := fmt.Sprintf(networkTemplate, containername, "none")
	connector, config := getConnector(t, configtemplate)
	plugin, err := connector.Deploy(context.Background(), "quay.io/tsebastiani/arcaflow-engine-deployer-podman-test:latest")
	assert.NoError(t, err)

	t.Cleanup(func() {
		cmd := exec.Command(config.Podman.Path, "container", "rm", containername)
		cmd.Run()
		assert.NoError(t, plugin.Close())
	})
	var containerInput = []byte("network none\n")
	//the test script will output a string containing the desired ip address and mac address filtered by the desired interface name
	assert.NoErrorR[int](t)(plugin.Write(containerInput))
	expectedOutput := "1;lo"

	readBuffer := readOutputUntil(t, plugin, expectedOutput)
	log.Infof(string(readBuffer))
	if len(readBuffer) == 0 {
		t.Fatal(fmt.Sprintf("the container did not produce any output"))
	}
	if strings.Contains(string(readBuffer), expectedOutput) == false {
		t.Fatal(fmt.Sprintf("expected string not found: %s", expectedOutput))
	}
}

// readOutputUntil helper function, reads from plugin (io.Reader) until finds lookforOutput
func readOutputUntil(t *testing.T, plugin deployer.Plugin, lookForOutput string) []byte {
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
