package integration

import (
	"time"

	. "github.com/containers/podman/v4/test/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Podman create with --ip flag", func() {

	It("Podman create --ip with garbage address", func() {
		result := podmanTest.Podman([]string{"create", "--name", "test", "--ip", "114232346", ALPINE, "ls"})
		result.WaitWithDefaultTimeout()
		Expect(result).To(ExitWithError())
	})

	It("Podman create --ip with non-allocatable IP", func() {
		SkipIfRootless("--ip not supported without network in rootless mode")
		result := podmanTest.Podman([]string{"create", "--name", "test", "--ip", "203.0.113.124", ALPINE, "ls"})
		result.WaitWithDefaultTimeout()
		Expect(result).Should(Exit(0))

		result = podmanTest.Podman([]string{"start", "test"})
		result.WaitWithDefaultTimeout()
		Expect(result).To(ExitWithError())
	})

	It("Podman create with specified static IP has correct IP", func() {
		ip := GetRandomIPAddress()
		result := podmanTest.Podman([]string{"create", "--name", "test", "--ip", ip, ALPINE, "ip", "addr"})
		result.WaitWithDefaultTimeout()
		// Rootless static ip assignment without network should error
		if isRootless() {
			Expect(result).Should(Exit(125))
		} else {
			Expect(result).Should(Exit(0))

			result = podmanTest.Podman([]string{"start", "-a", "test"})
			result.WaitWithDefaultTimeout()
			Expect(result).Should(Exit(0))
			Expect(result.OutputToString()).To(ContainSubstring(ip + "/16"))
		}
	})

	It("Podman create two containers with the same IP", func() {
		SkipIfRootless("--ip not supported without network in rootless mode")
		ip := GetRandomIPAddress()
		result := podmanTest.Podman([]string{"create", "--log-driver", "k8s-file", "--name", "test1", "--ip", ip, ALPINE, "sleep", "999"})
		result.WaitWithDefaultTimeout()
		Expect(result).Should(Exit(0))
		result = podmanTest.Podman([]string{"create", "--log-driver", "k8s-file", "--name", "test2", "--ip", ip, ALPINE, "ip", "addr"})
		result.WaitWithDefaultTimeout()
		Expect(result).Should(Exit(0))
		result = podmanTest.Podman([]string{"start", "test1"})
		result.WaitWithDefaultTimeout()
		Expect(result).Should(Exit(0))

		// race prevention: wait until IP address is assigned and
		// container is running.
		for i := 0; i < 5; i++ {
			result = podmanTest.Podman([]string{"inspect", "--format", "{{.State.Status}} {{.NetworkSettings.IPAddress}}", "test1"})
			result.WaitWithDefaultTimeout()
			Expect(result).Should(Exit(0))
			if result.OutputToString() == "running "+ip {
				break
			}
			time.Sleep(1 * time.Second)
		}
		Expect(result.OutputToString()).To(Equal("running " + ip))

		// test1 container is running with the given IP.
		result = podmanTest.Podman([]string{"start", "-a", "test2"})
		result.WaitWithDefaultTimeout()
		Expect(result).To(ExitWithError())
		if podmanTest.NetworkBackend == CNI {
			Expect(result.ErrorToString()).To(ContainSubstring("requested IP address %s is not available", ip))
		} else if podmanTest.NetworkBackend == Netavark {
			Expect(result.ErrorToString()).To(ContainSubstring("requested ip address %s is already allocated", ip))
		}
	})
})
