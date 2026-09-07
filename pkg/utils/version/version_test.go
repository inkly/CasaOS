package version

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/inkly/CasaOS/common"
)

func TestIsVersionNewer(t *testing.T) {
	tests := []struct {
		name    string
		latest  string
		current string
		want    bool
	}{
		{name: "platform-neutral migration", latest: "v0.4.20", current: "v0.4.17-ubuntu26.3", want: true},
		{name: "next release", latest: "v0.4.20", current: "v0.4.19", want: true},
		{name: "next CasaOS minor", latest: "v0.5.0", current: "v0.4.20", want: true},
		{name: "same release", latest: "v0.4.20", current: "v0.4.20", want: false},
		{name: "older compatibility release", latest: "v0.4.19", current: "v0.4.20", want: false},
		{name: "missing latest", latest: "", current: "v0.4.20", want: false},
		{name: "invalid latest", latest: "latest", current: "v0.4.20", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := IsVersionNewer(test.latest, test.current); got != test.want {
				t.Fatalf("IsVersionNewer(%q, %q) = %v, want %v", test.latest, test.current, got, test.want)
			}
		})
	}
}

func TestCurrentVersionFromFile(t *testing.T) {
	// The version is reported without the "v" of the release tag: the dashboard
	// prefixes it itself, and both sources below hold the tag verbatim.
	fallback := strings.TrimPrefix(common.FORK_RELEASE_VERSION, "v")

	versionFile := filepath.Join(t.TempDir(), "fork-release")
	if got := currentVersionFromFile(versionFile); got != fallback {
		t.Fatalf("missing version file returned %q, want %q", got, fallback)
	}

	if err := os.WriteFile(versionFile, []byte("v0.4.20\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := currentVersionFromFile(versionFile); got != "0.4.20" {
		t.Fatalf("installed version returned %q", got)
	}
}
