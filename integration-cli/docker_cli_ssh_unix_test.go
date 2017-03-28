// +build !windows

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"text/template"
	"time"

	"github.com/docker/docker/integration-cli/cli"
	icmd "github.com/docker/docker/pkg/testutil/cmd"
	"github.com/go-check/check"
)

func (s *DockerDaemonSuite) TestSSHClientWithSSHAgent(c *check.C) {
	testRequires(c, SameHostDaemon, DaemonIsLinux)
	sshAuthSock, dockerHost, cleanup := setupSSHTest(s, c, 60022)
	defer cleanup()
	type testCase struct {
		Args     []string
		Env      []string
		Expected icmd.Expected
	}
	cases := []testCase{
		{
			Args:     []string{"-H", dockerHost},
			Expected: icmd.Expected{Err: "ssh: handshake failed", ExitCode: 1},
		},
		{
			Args:     []string{"-H", dockerHost},
			Env:      []string{"SSH_AUTH_SOCK=" + sshAuthSock},
			Expected: icmd.Expected{Out: "hello ssh", ExitCode: 0},
		},
	}
	for _, x := range cases {
		result := cli.Docker(
			cli.Args(
				append(x.Args, []string{"run", "--rm", "busybox",
					"echo", "hello ssh"}...)...),
			// FIXME: go-connections requires HOME to be set even when unneeded
			cli.WithEnvironmentVariables(
				append(os.Environ(), x.Env...)...))
		c.Assert(result, icmd.Matches, x.Expected)
	}
}

// setupSSHTest returns sshAuthSock, dockerHost, and cleanup
func setupSSHTest(s *DockerDaemonSuite, c *check.C, sshPort int) (string, string, func()) {
	tempDir, err := ioutil.TempDir("", "test-ssh-client")
	if err != nil {
		c.Fatal(err)
	}
	apiSock := filepath.Join(tempDir, "docker.sock")
	sshAuthSock := filepath.Join(tempDir, "ssh-agent.sock")
	dockerHost := fmt.Sprintf("ssh://localhost:%d%s", sshPort, apiSock)

	s.d.StartWithBusybox(c, "-H", "unix://"+apiSock)
	shutdownSSHD := startSSHD(c, tempDir, sshPort)
	shutdownSSHAgent := startSSHAgent(c, sshAuthSock)
	addKeyToSSHAgent(c, tempDir, sshAuthSock)
	cleanup := func() {
		shutdownSSHAgent()
		shutdownSSHD()
		s.d.Stop(c)
		os.RemoveAll(tempDir)
	}
	return sshAuthSock, dockerHost, cleanup
}

// sshFixturesDir must contain:
//  ssh_host_rsa_key
//  authorized_keys
//  id_rsa
func sshFixturesDir(c *check.C) string {
	wd, err := os.Getwd()
	if err != nil {
		c.Fatal(err)
	}
	return filepath.Join(wd, "fixtures", "ssh")
}

var sshdConfigTemplate = template.Must(template.New("").Parse(`Protocol 2
UsePrivilegeSeparation no
HostKey {{.SSHFixturesDir}}/ssh_host_rsa_key
AuthorizedKeysFile {{.SSHFixturesDir}}/authorized_keys
AllowStreamLocalForwarding yes
`))

func lookupSSHD(c *check.C) string {
	sshd, err := exec.LookPath("sshd")
	if err != nil {
		c.Skip("sshd not installed")
	}
	return sshd
}

func startSSHD(c *check.C, tempDir string, port int) func() {
	sshdConfig := filepath.Join(tempDir, "sshd_config")
	sshdConfigWriter, err := os.Create(sshdConfig)
	if err != nil {
		c.Fatal(err)
	}
	if err = sshdConfigTemplate.Execute(sshdConfigWriter,
		map[string]string{"SSHFixturesDir": sshFixturesDir(c)}); err != nil {
		c.Fatal(err)
	}
	// sshd requires argv0 to be absolute path
	cmd := exec.Command(lookupSSHD(c), "-f", sshdConfig, "-p", strconv.Itoa(port), "-D")
	if err = cmd.Start(); err != nil {
		c.Fatal(err)
	}
	cleanup := func() {
		if err = cmd.Process.Kill(); err != nil {
			c.Fatal(err)
		}
	}
	return cleanup
}

func startSSHAgent(c *check.C, sshAuthSock string) func() {
	// -D (foreground) is not supported in older ssh-agent;
	// so we use -d (foreground + debug)
	cmd := exec.Command("ssh-agent", "-a", sshAuthSock, "-d")
	if err := cmd.Start(); err != nil {
		c.Fatal(err)
	}
	cleanup := func() {
		if err := cmd.Process.Kill(); err != nil {
			c.Fatal(err)
		}
	}
	time.Sleep(3 * time.Second) // FIXME
	return cleanup
}

func addKeyToSSHAgent(c *check.C, tempDir, sshAuthSock string) {
	idRSABytes, err := ioutil.ReadFile(filepath.Join(sshFixturesDir(c), "id_rsa"))
	if err != nil {
		c.Fatal(err)
	}
	idRSA := filepath.Join(tempDir, "id_rsa")
	if err = ioutil.WriteFile(idRSA, idRSABytes, 0400); err != nil {
		c.Fatal(err)
	}
	cmd := icmd.Cmd{
		Command: []string{"ssh-add", idRSA},
		Env:     []string{"SSH_AUTH_SOCK=" + sshAuthSock},
	}
	if res := icmd.RunCmd(cmd); res.Error != nil {
		c.Fatal(res.String())
	}
}
