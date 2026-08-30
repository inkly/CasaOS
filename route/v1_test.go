package route

import "testing"

func TestSkipJWT(t *testing.T) {
	cases := []struct {
		name   string
		path   string
		realIP string
		want   bool
	}{
		{"loopback IPv4 on a regular route is trusted", "/v1/sys/hardware", "127.0.0.1", true},
		{"loopback IPv6 on a regular route is trusted", "/v1/sys/hardware", "::1", true},
		{"a remote address is never trusted", "/v1/sys/hardware", "192.168.1.20", false},

		// The point of the guard: these three run apt as root.
		{"package list is not trusted from loopback", "/v1/sys/packages", "127.0.0.1", false},
		{"package update is not trusted from loopback", "/v1/sys/packages/update", "127.0.0.1", false},
		{"package status is not trusted from loopback", "/v1/sys/packages/update/status", "::1", false},
		{"package update is not trusted from anywhere else either", "/v1/sys/packages/update", "192.168.1.20", false},

		// Creating a share account runs useradd and smbpasswd as root, so it sits
		// behind the same rule as package installation.
		{"samba account list is not trusted from loopback", "/v1/samba/users", "127.0.0.1", false},
		{"samba account creation is not trusted from loopback", "/v1/samba/users", "::1", false},
		{"samba password change is not trusted from loopback", "/v1/samba/users/alice/password", "127.0.0.1", false},

		// Shares themselves are not privileged in the same way and keep the
		// existing loopback behaviour.
		{"listing shares keeps the loopback exemption", "/v1/samba/shares", "127.0.0.1", true},

		// A shorter path that merely resembles the prefix stays trusted.
		{"a similarly named route keeps the loopback exemption", "/v1/sys/package", "127.0.0.1", true},
		// Anything under the prefix fails closed, even an unknown suffix.
		{"an unknown route under the prefix fails closed", "/v1/sys/packages-experimental", "127.0.0.1", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := skipJWT(tc.path, tc.realIP); got != tc.want {
				t.Fatalf("skipJWT(%q, %q) = %v, want %v", tc.path, tc.realIP, got, tc.want)
			}
		})
	}
}
