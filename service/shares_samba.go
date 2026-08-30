package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	model2 "github.com/IceWhaleTech/CasaOS/service/model"
)

const (
	sambaConfigDir       = "/etc/samba"
	sambaConfigFile      = "/etc/samba/smb.conf"
	sambaShareConfigName = "smb.casa.conf"
	sambaShareConfigFile = "/etc/samba/smb.casa.conf"
)

// sambaSection renders one share as an smb.conf section.
//
// A share with no Username is written exactly as CasaOS has always written it:
// guest accessible, world-writable, files forced to root. Every row that
// predates authenticated shares carries an empty Username, so an upgrade leaves
// those shares byte-identical.
//
// An authenticated share drops all three of those together. Leaving any single
// one behind would make the authentication decorative: "guest ok" alone re-opens
// the share, and "force user = root" alone hands every file back to root no
// matter who connected.
func sambaSection(share model2.SharesDBModel) string {
	name := filepath.Base(share.Path)

	section := "\n[" + name + "]\n" +
		"comment = CasaOS share " + name + "\n" +
		"path = " + share.Path + "\n" +
		"browseable = Yes\n" +
		"read only = No\n"

	if share.Username == "" {
		return section +
			"public = Yes\n" +
			"guest ok = Yes\n" +
			"create mask = 0777\n" +
			"directory mask = 0777\n" +
			"force user = root\n\n"
	}

	return section +
		"public = No\n" +
		"guest ok = No\n" +
		"valid users = " + share.Username + "\n" +
		"create mask = 0660\n" +
		"directory mask = 0770\n" +
		"force user = " + share.Username + "\n\n"
}

// validateSambaConfig asks Samba's own parser whether the generated file is
// usable.
//
// smb.casa.conf is included by smb.conf, and casaos-wsdd.service declares
// PartOf=smbd.service: a configuration smbd refuses does not merely break
// sharing, it takes network discovery down with it, and neither failure is
// reported anywhere. Checking first is what stops one bad share from silently
// disabling both.
//
// A host without Samba's tooling installed has nothing to validate against,
// which is not an error.
func validateSambaConfig() error {
	if _, err := exec.LookPath("testparm"); err != nil {
		return nil
	}

	output, err := exec.Command("testparm", "--suppress-prompt", sambaConfigFile).CombinedOutput()
	if err != nil {
		return fmt.Errorf("samba rejected the generated share configuration: %s", strings.TrimSpace(string(output)))
	}

	return nil
}

// migrateGuestMapping turns "map to guest = bad user" into "map to guest = never"
// in an smb.conf CasaOS has already written.
//
// InitSambaConfig only writes that file on a host that has never seen CasaOS, so
// on every existing installation this is the only route by which the setting can
// land. It matters because under "bad user" a client whose credentials are
// refused is silently remapped to the guest account and receives ACCESS_DENIED
// instead of being asked for a password: an authenticated share would look
// broken rather than protected.
//
// Only that one line is rewritten. Regenerating the file would discard the hand
// edits users made while working around the absence of authentication, which the
// upstream issue thread is full of.
func migrateGuestMapping() error {
	content, err := os.ReadFile(sambaConfigFile)
	if err != nil {
		return err
	}

	updated := strings.ReplaceAll(string(content), "map to guest = bad user", "map to guest = never")
	if updated == string(content) {
		return nil
	}

	return os.WriteFile(sambaConfigFile, []byte(updated), 0o644)
}
