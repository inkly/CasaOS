package common

import (
	"os"
	"regexp"
	"testing"
)

// These files ship with the installer and decide which release feed a host
// polls: 03-setup-casaos.sh rewrites /etc/casaos/casaos.conf on every install
// and upgrade, the two samples seed it on a fresh one, and update.sh is the
// manual escape hatch. The Go resolvers prefer the config file over the
// constants below, so retargeting the constants without these files leaves
// installed hosts fetching someone else's distribution.
func TestShippedUpdateURLsMatchConstants(t *testing.T) {
	url := regexp.MustCompile(`https://\S+?/(?:install\.sh|version\.json)`)

	for _, path := range []string{
		"../build/scripts/setup/script.d/03-setup-casaos.sh",
		"../build/sysroot/etc/casaos/casaos.conf.sample",
		"../conf/conf.conf.sample",
		"../build/sysroot/usr/share/casaos/shell/update.sh",
	} {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}

		found := url.FindAllString(string(content), -1)
		if len(found) == 0 {
			t.Errorf("%s: no installer or version URL found", path)
		}

		for _, got := range found {
			if got != FORK_UPDATE_URL && got != FORK_VERSION_URL {
				t.Errorf("%s: %q is neither FORK_UPDATE_URL (%q) nor FORK_VERSION_URL (%q)", path, got, FORK_UPDATE_URL, FORK_VERSION_URL)
			}
		}
	}
}
