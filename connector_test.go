package podman

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/opencontainers/selinux/go-selinux"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"go.arcalot.io/assert"
	log "go.arcalot.io/log/v2"
	"go.flow.arcalot.io/deployer"
	"go.flow.arcalot.io/podmandeployer/tests"
)

func getConnector(t *testing.T, configJSON string) (deployer.Connector, *Config) {
	var config any
	err := json.Unmarshal([]byte(configJSON), &config)
	assert.NoError(t, err)
	factory := NewFactory()
	schema := factory.ConfigurationSchema()
	unserializedConfig, err := schema.UnserializeType(config)
	assert.NoError(t, err)
	connector, err := factory.Create(unserializedConfig, log.NewTestLogger(t))
	assert.NoError(t, err)
	unserializedConfig.Podman.Path, err = binaryCheck(unserializedConfig.Podman.Path)
	if err != nil {
		t.Fatalf("Error checking Podman path (%s)", err)
	}
	return connector, unserializedConfig
}

var inOutConfig = `
{
   "podman":{
      "path":"podman"
   }
}
`

func TestSimpleInOut(t *testing.T) {
	logger := log.NewTestLogger(t)
	pongStr := "pong abc"
	endStr := "end abc"

	connector, _ := getConnector(t, inOutConfig)
	plugin, err := connector.Deploy(
		context.Background(),
		"quay.io/arcalot/podman-deployer-test-helper:0.1.0")
	assert.NoError(t, err)

	t.Cleanup(func() { assert.NoError(t, plugin.Close()) })

	var containerInput = []byte("ping abc\n")
	assert.NoErrorR[int](t)(plugin.Write(containerInput))

	readBuffer := readOutputUntil(t, plugin, pongStr)
	// assert output is not empty
	assert.Equals(t, len(readBuffer) > 0, true)

	logger.Infof(string(readBuffer[:7]))
	assert.NoErrorR[int](t)(plugin.Write(containerInput))

	readBuffer = readOutputUntil(t, plugin, endStr)
	// assert output is not empty
	assert.Equals(t, len(readBuffer) > 0, true)
}

var envConfig = `
{
   "deployment":{
      "container":{
         "Env":[
            "DEPLOYER_PODMAN_TEST_1=TEST1",
            "DEPLOYER_PODMAN_TEST_2=TEST2"
         ]
      }
   },
   "podman":{
      "path":"podman"
   }
}
`

func TestEnv(t *testing.T) {
	envVars := "DEPLOYER_PODMAN_TEST_1=TEST1\nDEPLOYER_PODMAN_TEST_2=TEST2"
	connector, _ := getConnector(t, envConfig)
	container, err := connector.Deploy(context.Background(), "quay.io/arcalot/podman-deployer-test-helper:0.1.0")
	assert.NoError(t, err)

	t.Cleanup(func() { assert.NoError(t, container.Close()) })

	assert.NoErrorR[int](t)(container.Write([]byte("env\n")))

	readBuffer := readOutputUntil(t, container, envVars)
	assert.GreaterThan(t, len(readBuffer), 0)
}

var volumeConfig = `
{
   "deployment":{
      "host":{
         "Binds":[
            "%s:/test/test_file.txt%s"
         ]
      }
   },
   "podman":{
      "path":"podman"
   }
}
`

// bindMountHelper is a helper function which tests plugins with a file
// bind-mounted inside the container.  Options for the mount and the expected
// outcome of the test are provided by parameters.  The test creates a
// temporary file containing appropriate content, configures that file to be
// mounted inside the container, and then starts the plugin; the test then
// tells the plugin to output the contents of the mapped file and checks it
// against the value originally written to the file.
func bindMountHelper(t *testing.T, options string, expectedPass bool) {
	fileContent := fmt.Sprintf("bind mount test with option %q\n", options)
	mountFile := assert.NoErrorR[*os.File](t)(os.CreateTemp("", "bind_mount_test_*.txt"))
	t.Cleanup(func() { assert.NoError(t, os.Remove(mountFile.Name())) })
	assert.NoErrorR[int](t)(mountFile.WriteString(fileContent))
	assert.NoError(t, mountFile.Close())
	connector, _ := getConnector(t, fmt.Sprintf(volumeConfig, mountFile.Name(), options))

	// Run the plugin
	container := assert.NoErrorR[deployer.Plugin](t)(connector.Deploy(
		context.Background(),
		"quay.io/arcalot/podman-deployer-test-helper:0.1.0"))
	t.Cleanup(func() { assert.NoError(t, container.Close()) })

	// Tell the plugin to output the contents of the mapped file.
	assert.NoErrorR[int](t)(container.Write([]byte("volume\n")))

	// Note: If it ends up with length zero buffer, restarting the VM may help:
	// https://stackoverflow.com/questions/71977532/podman-mount-host-volume-return-error-statfs-no-such-file-or-directory-in-ma
	readBuffer := readOutputUntil(t, container, fileContent)
	if expectedPass {
		assert.Contains(t, string(readBuffer), fileContent)
	} else {
		// If this assertion fails, then we actually found what we were looking
		// for (despite our hopes to the contrary), so having the failure
		// message include the strings probably isn't important, and so we can
		// get by without an `assert.NotContains()` function.
		assert.Equals(t, strings.Contains(string(readBuffer), fileContent), false)
	}
}

func TestBindMount(t *testing.T) {
	type param struct {
		option       string
		expectedPass bool
	}
	scenarios := map[string]param{
		"ReadOnly":   {":ro", true},
		"Multiple":   {":ro,noexec", true},
		"No options": {"", true},
	}
	if tests.IsRunningOnLinux() {
		// The SELinux options seem to cause problems on Mac OS X, so only test
		// them on Linux.
		scenarios["Private"] = param{":Z", true}
		scenarios["Shared"] = param{":z", true}
		if selinux.GetEnabled() {
			// On Linux, bind mounts without relabeling options will fail when
			// SELinux is enabled.  So, reset expectations appropriately.
			scenarios["No options"] = param{scenarios["No options"].option, false}
		}

	}
	for name, p := range scenarios {
		param := p
		t.Run(name, func(t *testing.T) { bindMountHelper(t, param.option, param.expectedPass) })
	}
}

var nameTemplate = `
{
  "podman":{
     "path":"podman",
     "containerNamePrefix":"%s"
  }
}
`

func TestContainerName(t *testing.T) {
	logger := log.NewTestLogger(t)
	configTemplate1 := fmt.Sprintf(nameTemplate, "test_1")
	configTemplate2 := fmt.Sprintf(nameTemplate, "test_2")

	ctx := context.Background()
	connector1, cfg1 := getConnector(t, configTemplate1)
	connector2, cfg2 := getConnector(t, configTemplate2)

	container1, err := connector1.Deploy(
		ctx,
		"quay.io/arcalot/podman-deployer-test-helper:0.1.0")
	assert.NoError(t, err)

	t.Cleanup(func() { assert.NoError(t, container1.Close()) })

	container2, err := connector2.Deploy(
		ctx,
		"quay.io/arcalot/podman-deployer-test-helper:0.1.0")
	assert.NoError(t, err)

	t.Cleanup(func() { assert.NoError(t, container2.Close()) })

	assert.Equals(t, container1.ID() != container2.ID(), true)

	containerInput := []byte("sleep 3\n")
	assert.NoErrorR[int](t)(container1.Write(containerInput))
	assert.NoErrorR[int](t)(container2.Write(containerInput))

	// Wait for each of the containers to start running; arbitrarily fail the
	// test if it doesn't all happen within 30 seconds.
	end := time.Now().Add(30 * time.Second)
	for !tests.IsContainerRunning(logger, cfg1.Podman.Path, container1.ID()) {
		assert.Equals(t, time.Now().Before(end), true)
		time.Sleep(1 * time.Second)
	}
	for !tests.IsContainerRunning(logger, cfg2.Podman.Path, container2.ID()) {
		assert.Equals(t, time.Now().Before(end), true)
		time.Sleep(1 * time.Second)
	}
}

var cgroupTemplate = `
{
   "podman":{
      "path":"podman",
      "containerNamePrefix":"%s"
   },
   "deployment":{
	   "host":{
		  "CgroupnsMode":"%s"
	   }
   }
}
`

func TestCgroupNsByContainerName(t *testing.T) {
	if tests.IsRunningOnGithub() {
		t.Skipf("joining another container cgroup namespace by container name not supported on GitHub actions")
	}
	logger := log.NewTestLogger(t)

	containerNamePrefix1 := "test_1"
	// The first container will run with a private namespace that will be created at startup
	configtemplate1 := fmt.Sprintf(cgroupTemplate, containerNamePrefix1, "private")
	connector1, config := getConnector(t, configtemplate1)
	container1, err := connector1.Deploy(
		context.Background(),
		"quay.io/arcalot/podman-deployer-test-helper:0.1.0")
	assert.NoError(t, err)

	t.Cleanup(func() { assert.NoError(t, container1.Close()) })

	containerNamePrefix2 := "test_2"
	// The second one will join the newly created private namespace of the first container
	configtemplate2 := fmt.Sprintf(cgroupTemplate, containerNamePrefix2, fmt.Sprintf("container:%s", container1.ID()))
	connector2, _ := getConnector(t, configtemplate2)
	container2, err := connector2.Deploy(
		context.Background(),
		"quay.io/arcalot/podman-deployer-test-helper:0.1.0")
	assert.NoError(t, err)

	t.Cleanup(func() { assert.NoError(t, container2.Close()) })

	assert.NoErrorR[int](t)(container1.Write([]byte("sleep 7\n")))

	// Wait for each of the containers to start running so that we can collect
	// their cgroup names; arbitrarily fail the test if it doesn't all happen
	// within 30 seconds.
	end := time.Now().Add(30 * time.Second)
	var ns1, ns2 string
	for ns1 == "" {
		ns1 = tests.GetPodmanPsNsWithFormat(logger, config.Podman.Path, container1.ID(), "{{.CGROUPNS}}")
		assert.Equals(t, time.Now().Before(end), true)
		time.Sleep(1 * time.Second)
	}
	for ns2 == "" {
		ns2 = tests.GetPodmanPsNsWithFormat(logger, config.Podman.Path, container2.ID(), "{{.CGROUPNS}}")
		assert.Equals(t, time.Now().Before(end), true)
		time.Sleep(1 * time.Second)
	}
	assert.Equals(t, ns1 == ns2, true)

	// Release the second container from its input prompt via a no-op command.
	assert.NoErrorR[int](t)(container2.Write([]byte(":\n")))
}

func TestPrivateCgroupNs(t *testing.T) {
	// get the user cgroup ns
	logger := log.NewTestLogger(t)

	// Assume sleep is in the path. Because it's not in the same location for every user.
	userCgroupNs := tests.GetCommmandCgroupNs(logger, "sleep", []string{"3"})
	assert.NotNil(t, userCgroupNs)
	logger.Debugf("Detected cgroup namespace for user: %s", userCgroupNs)

	containerNamePrefix := "test"
	// The container will run with a private namespace that will be created at startup
	configtemplate := fmt.Sprintf(cgroupTemplate, containerNamePrefix, "private")
	connector, config := getConnector(t, configtemplate)
	container, err := connector.Deploy(
		context.Background(),
		"quay.io/arcalot/podman-deployer-test-helper:0.1.0")
	assert.NoError(t, err)

	t.Cleanup(func() { assert.NoError(t, container.Close()) })

	assert.NoErrorR[int](t)(container.Write([]byte("sleep 5\n")))

	// Wait for the container to start running so that we can collect its
	// cgroup name; arbitrarily fail the test if it doesn't all happen within
	// 30 seconds.
	end := time.Now().Add(30 * time.Second)
	var podmanCgroupNs string
	for podmanCgroupNs == "" {
		podmanCgroupNs = tests.GetPodmanCgroupNs(logger, config.Podman.Path, container.ID())
		assert.Equals(t, time.Now().Before(end), true)
		time.Sleep(1 * time.Second)
	}
	assert.Equals(t, userCgroupNs != podmanCgroupNs, true)
}

func TestHostCgroupNs(t *testing.T) {
	if !tests.IsRunningOnLinux() {
		t.Skipf("Not running on Linux. Skipping cgroup test.")
	}
	logger := log.NewTestLogger(t)

	// Assume sleep is in the path. Because it's not in the same location for every user.
	userCgroupNs := tests.GetCommmandCgroupNs(logger, "sleep", []string{"3"})
	assert.NotNil(t, userCgroupNs)

	logger.Debugf("Detected cgroup namespace for user: %s", userCgroupNs)
	containerNamePrefix := "host_cgroupns"
	// The first container will run with the host namespace
	configtemplate := fmt.Sprintf(cgroupTemplate, containerNamePrefix, "host")
	connector, config := getConnector(t, configtemplate)
	container, err := connector.Deploy(
		context.Background(),
		"quay.io/arcalot/podman-deployer-test-helper:0.1.0")
	assert.NoError(t, err)

	t.Cleanup(func() { assert.NoError(t, container.Close()) })

	assert.NoErrorR[int](t)(container.Write([]byte("sleep 5\n")))

	// Wait for the container to start running so that we can collect its
	// cgroup name; arbitrarily fail the test if it doesn't all happen within
	// 30 seconds.
	end := time.Now().Add(30 * time.Second)
	var podmanCgroupNs string
	for podmanCgroupNs == "" {
		podmanCgroupNs = tests.GetPodmanCgroupNs(logger, config.Podman.Path, container.ID())
		assert.Equals(t, time.Now().Before(end), true)
		time.Sleep(1 * time.Second)
	}
	assert.Equals(t, userCgroupNs, podmanCgroupNs)
}

func TestCgroupNsByNamespacePath(t *testing.T) {
	if tests.IsRunningOnGithub() {
		t.Skipf("joining another container cgroup namespace by namespace path ns:/proc/<PID>/ns/cgroup not supported on GitHub actions")
	}
	logger := log.NewTestLogger(t)
	containerNamePrefix1 := "test_1"
	// The first container will run with a private namespace that will be created at startup
	configtemplate1 := fmt.Sprintf(cgroupTemplate, containerNamePrefix1, "private")
	connector1, config := getConnector(t, configtemplate1)
	container1, err := connector1.Deploy(context.Background(), "quay.io/arcalot/podman-deployer-test-helper:0.1.0")
	assert.NoError(t, err)

	t.Cleanup(func() { assert.NoError(t, container1.Close()) })

	assert.NoErrorR[int](t)(container1.Write([]byte("sleep 10\n")))

	// Wait for each of the containers to start running so that we can collect
	// their cgroup names; arbitrarily fail the test if it doesn't all happen
	// within 30 seconds.
	end := time.Now().Add(30 * time.Second)
	var ns1 string
	for ns1 == "" {
		ns1 = tests.GetPodmanPsNsWithFormat(logger, config.Podman.Path, container1.ID(), "{{.CGROUPNS}}")
		assert.Equals(t, time.Now().Before(end), true)
		time.Sleep(1 * time.Second)
	}
	assert.NotNil(t, ns1)

	pid := tests.GetPodmanPsNsWithFormat(logger, config.Podman.Path, container1.ID(), "{{.Pid}}")

	containerNamePrefix2 := "test_2"
	// The second one will join the newly created private namespace of the first container
	namespacePath := fmt.Sprintf("ns:/proc/%s/ns/cgroup", pid)
	configtemplate2 := fmt.Sprintf(cgroupTemplate, containerNamePrefix2, namespacePath)
	connector2, _ := getConnector(t, configtemplate2)

	container2, err := connector2.Deploy(
		context.Background(),
		"quay.io/arcalot/podman-deployer-test-helper:0.1.0")
	assert.NoError(t, err)

	t.Cleanup(func() { assert.NoError(t, container2.Close()) })

	assert.NoErrorR[int](t)(container2.Write([]byte("sleep 5\n")))

	var ns2 string
	for ns2 == "" {
		ns2 = tests.GetPodmanPsNsWithFormat(logger, config.Podman.Path, container2.ID(), "{{.CGROUPNS}}")
		assert.Equals(t, time.Now().Before(end), true)
		time.Sleep(1 * time.Second)
	}
	assert.Equals(t, ns1 == ns2, true)
}

var networkTemplate = `
{
   "podman":{
      "containerNamePrefix":"%s",
      "path":"podman"
   },
   "deployment":{
	   "host":{
		  "NetworkMode":"%s"
	   }
   }
}
`

func TestNetworkHost(t *testing.T) {
	logger := log.NewTestLogger(t)
	containerNamePrefix := "networkhost"
	// The first container will run with the host namespace
	configtemplate := fmt.Sprintf(networkTemplate, containerNamePrefix, "host")
	connector, _ := getConnector(t, configtemplate)
	plugin, err := connector.Deploy(
		context.Background(),
		"quay.io/arcalot/podman-deployer-test-helper:0.1.0")
	assert.NoError(t, err)

	t.Cleanup(func() { assert.NoError(t, plugin.Close()) })

	var containerInput = []byte("network host\n")
	// the test script will run "ifconfig" in the container
	assert.NoErrorR[int](t)(plugin.Write(containerInput))

	var ifconfigOut bytes.Buffer
	// runs ifconfig in the host machine in order to check if the container has exactly the same network configuration
	cmd := exec.Command(
		"/bin/bash", "-c", "ifconfig | grep -P \"^.+:\\s+.+$\" | awk '{ gsub(\":\",\"\");print $1 }'")
	cmd.Stdout = &ifconfigOut
	assert.NoError(t, cmd.Run())

	ifconfigOutStr := ifconfigOut.String()
	logger.Infof(ifconfigOutStr)
	readBuffer := readOutputUntil(t, plugin, ifconfigOutStr)
	containerOutString := string(readBuffer)
	assert.Contains(t, containerOutString, ifconfigOutStr)
}

func TestNetworkBridge(t *testing.T) {
	// If this test breaks again, delete it.

	// This test forces the container to have the following
	// network settings:
	// ip 10.88.0.123
	// mac 44:33:22:11:00:99
	// then asks the container to run an ifconfig (tests/test_script.sh, test_network())
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
	containerNamePrefix := "close"
	configTemplate := fmt.Sprintf(nameTemplate, containerNamePrefix)
	connector, _ := getConnector(t, configTemplate)

	container, err := connector.Deploy(
		context.Background(),
		"quay.io/arcalot/podman-deployer-test-helper:0.1.0")
	assert.NoError(t, err)

	assert.NoErrorR[int](t)(container.Write([]byte("sleep 10\n")))

	time.Sleep(2 * time.Second)
	err = container.Close()
	assert.NoError(t, err)
}

// readOutputUntil is a helper function which reads from the provided io.Reader
// until it receives the specified string or EOF; returns the bytes read.
func readOutputUntil(t *testing.T, plugin io.Reader, lookForOutput string) []byte {
	var n int
	readBuffer := make([]byte, 10240)
	for !strings.Contains(string(readBuffer[:n]), lookForOutput) {
		currentBuffer := make([]byte, 1024)
		readBytes, err := plugin.Read(currentBuffer)
		copy(readBuffer[n:], currentBuffer[:readBytes])
		n += readBytes

		if err == io.EOF {
			break
		} else if err != nil {
			t.Fatalf("error while reading stdout: %s", err.Error())
		}
	}
	return readBuffer[:n]
}

func testNetworking(t *testing.T, podmanNetworking string, containerTest string, expectedOutput *string, ip *string, mac *string) {
	logger := log.NewTestLogger(t)
	assert.NoErrorR[string](t)(exec.LookPath("ifconfig"))

	containerNamePrefix := "networking"
	// The first container will run with the host namespace
	configtemplate := fmt.Sprintf(networkTemplate, containerNamePrefix, podmanNetworking)
	connector, _ := getConnector(t, configtemplate)
	plugin, err := connector.Deploy(
		context.Background(),
		"quay.io/arcalot/podman-deployer-test-helper:0.1.0")
	assert.NoError(t, err)

	t.Cleanup(func() { assert.NoError(t, plugin.Close()) })

	var containerInput = []byte(containerTest)
	// the test script will output a string containing the desired ip address and mac address
	// filtered by the desired interface name
	assert.NoErrorR[int](t)(plugin.Write(containerInput))

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

	// assert the container input is not empty
	assert.Equals(t, len(readBuffer) > 0, true)

	if ip == nil && mac == nil && expectedOutput != nil {
		assert.Contains(t, string(readBuffer), *expectedOutput)
	} else {
		assert.Contains(t, string(readBuffer), *ip)
		assert.Contains(t, string(readBuffer), *mac)
	}
}
