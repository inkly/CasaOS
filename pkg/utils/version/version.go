/*
 * @Author: LinkLeong link@icewhale.com
 * @Date: 2022-05-13 18:15:46
 * @LastEditors: LinkLeong
 * @LastEditTime: 2022-07-21 15:27:53
 * @FilePath: /CasaOS/pkg/utils/version/version.go
 * @Description:
 * @Website: https://www.casaos.io
 * Copyright (c) 2022 by icewhale, All Rights Reserved.
 */
package version

import (
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/inkly/CasaOS/common"
	"github.com/inkly/CasaOS/model"
)

var numericVersionPart = regexp.MustCompile(`[0-9]+`)

func versionParts(value string) ([]uint64, bool) {
	matches := numericVersionPart.FindAllString(value, -1)
	if len(matches) == 0 {
		return nil, false
	}

	parts := make([]uint64, 0, len(matches))
	for _, match := range matches {
		part, err := strconv.ParseUint(match, 10, 64)
		if err != nil {
			return nil, false
		}
		parts = append(parts, part)
	}
	return parts, true
}

func IsVersionNewer(latest string, current string) bool {
	latestParts, latestOK := versionParts(latest)
	currentParts, currentOK := versionParts(current)
	if !latestOK || !currentOK {
		return false
	}

	length := len(latestParts)
	if len(currentParts) > length {
		length = len(currentParts)
	}
	for i := 0; i < length; i++ {
		var latestPart uint64
		var currentPart uint64
		if i < len(latestParts) {
			latestPart = latestParts[i]
		}
		if i < len(currentParts) {
			currentPart = currentParts[i]
		}
		if latestPart != currentPart {
			return latestPart > currentPart
		}
	}
	return false
}

func CurrentVersion() string {
	return currentVersionFromFile(common.FORK_RELEASE_FILE)
}

func currentVersionFromFile(path string) string {
	// FORK_RELEASE_FILE is the truth: the installer writes the distribution's tag
	// there on every install and every upgrade, so on any host that has run a
	// recent installer this is what answers.
	//
	// FORK_RELEASE_VERSION is the fallback, and it is a FLOOR rather than a second
	// copy of the same value: the distribution this binary was BUILT for. The two
	// may differ, because a distribution release need not ship this component --
	// and wherever they differ, the marker exists and wins. A host with no marker
	// has not run a recent installer, so it is running the binary that shipped with
	// the distribution the constant names, and the constant is the honest answer
	// for it. Reading them as equal is what made every distribution release drag a
	// release of this component along for one string.
	//
	// Both hold the tag with its "v". The API answers a bare version - the dashboard
	// adds the "v" itself, so the tag's own prefix showed up as "vv0.4.40".
	version := common.FORK_RELEASE_VERSION
	if data, err := os.ReadFile(path); err == nil {
		if installedVersion := strings.TrimSpace(string(data)); installedVersion != "" {
			version = installedVersion
		}
	}
	return strings.TrimPrefix(version, "v")
}

func IsNeedUpdate(version model.Version) (bool, model.Version) {
	return IsVersionNewer(version.Version, CurrentVersion()), version
}
