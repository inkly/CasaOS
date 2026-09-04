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

		// Creating a share chowns and chmods a caller-supplied directory as root,
		// so the share routes are privileged too. Exempting them would hand any
		// local process an unauthenticated chown of an arbitrary path.
		{"creating a share is not trusted from loopback", "/v1/samba/shares", "127.0.0.1", false},
		{"deleting a share is not trusted from loopback", "/v1/samba/shares/1", "::1", false},
		{"samba client connections are not trusted from loopback", "/v1/samba/connections", "127.0.0.1", false},

		// The file manager reads, writes and deletes caller-supplied absolute
		// paths as root, which is strictly more than the samba routes do, so it
		// needs a token from loopback too.
		{"file content is not trusted from loopback", "/v1/file/content", "127.0.0.1", false},
		{"file websocket is not trusted from loopback", "/v1/file/ws", "::1", false},
		{"folder listing is not trusted from loopback", "/v1/folder", "127.0.0.1", false},
		{"batch delete is not trusted from loopback", "/v1/batch", "::1", false},
		{"thumbnail is not trusted from loopback", "/v1/image", "127.0.0.1", false},

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
