package core

/*	License: GPLv3
	Authors:
		Mirko Brombin <mirko@fabricators.ltd>
		Vanilla OS Contributors <https://github.com/vanilla-os/>
	Copyright: 2023
	Description:
		ABRoot is utility which provides full immutability and
		atomicity to a Linux system, by transacting between
		two root filesystems. Updates are performed using OCI
		images, to ensure that the system is always in a
		consistent state.
*/

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/vanilla-os/abroot/extras/dpkg"
	"github.com/vanilla-os/abroot/settings"
	"github.com/vanilla-os/differ/diff"
)

// BaseImagePackageDiff retrieves the added, removed, upgraded and downgraded
// base packages (the ones bundled with the image).
func BaseImagePackageDiff(currentDigest, newDigest string) (
	added, upgraded, downgraded, removed []diff.PackageDiff,
	err error,
) {
	PrintVerbose("PackageDiff.BaseImagePackageDiff: running...")

	imageComponents := strings.Split(settings.Cnf.Name, "/")
	imageName := imageComponents[len(imageComponents)-1]
	reqUrl := fmt.Sprintf("%s/images/%s/diff", settings.Cnf.DifferURL, imageName)
	body := fmt.Sprintf("{\"old_digest\": \"%s\", \"new_digest\": \"%s\"}", currentDigest, newDigest)

	PrintVerbose("PackageDiff.BaseImagePackageDiff: Requesting base image diff to %s with body:\n%s", reqUrl, body)

	request, err := http.NewRequest(http.MethodGet, reqUrl, strings.NewReader(body))
	if err != nil {
		PrintVerbose("PackageDiff.BaseImagePackageDiff:err: %s", err)
		return
	}
	defer request.Body.Close()

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		PrintVerbose("PackageDiff.BaseImagePackageDiff(1):err: %s", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		PrintVerbose("PackageDiff.BaseImagePackageDiff(2):err: received non-ok status %s", resp.Status)
		err = fmt.Errorf("package diff server returned non-OK status %s", resp.Status)
		return
	}

	contents, err := io.ReadAll(resp.Body)
	if err != nil {
		PrintVerbose("PackageDiff.BaseImagePackageDiff(3):err: %s", err)
		return
	}

	pkgDiff := struct {
		Added, Upgraded, Downgraded, Removed []diff.PackageDiff
	}{}
	err = json.Unmarshal(contents, &pkgDiff)
	if err != nil {
		PrintVerbose("PackageDiff.BaseImagePackageDiff(4):err: %s", err)
		return
	}

	added = pkgDiff.Added
	upgraded = pkgDiff.Upgraded
	downgraded = pkgDiff.Downgraded
	removed = pkgDiff.Removed

	return
}

// OverlayPackageDiff retrieves the added, removed, upgraded and downgraded
// overlay packages (the ones added manually via `abroot pkg add`).
func OverlayPackageDiff() (
	added, upgraded, downgraded, removed []diff.PackageDiff,
	err error,
) {
	PrintVerbose("OverlayPackageDiff: running...")

	pkgM := NewPackageManager(false)
	addedPkgs, err := pkgM.GetAddPackages()
	if err != nil {
		PrintVerbose("PackageDiff.OverlayPackageDiff:err: %s", err)
		return
	}

	localAddedVersions := dpkg.DpkgBatchGetPackageVersion(addedPkgs)
	localAdded := map[string]string{}
	for i := 0; i < len(addedPkgs); i++ {
		if localAddedVersions[i] != "" {
			localAdded[addedPkgs[i]] = localAddedVersions[i]
		}
	}

	remoteAdded := map[string]string{}
	var pkgInfo map[string]any
	for pkgName := range localAdded {
		pkgInfo, err = GetRepoContentsForPkg(pkgName)
		if err != nil {
			PrintVerbose("PackageDiff.OverlayPackageDiff(1):err: %s", err)
			return
		}
		version, ok := pkgInfo["version"].(string)
		if !ok {
			err = fmt.Errorf("PackageDiff.OverlayPackageDiff(2):err: Unexpected value when retrieving upstream version of '%s'", pkgName)
			return
		}
		remoteAdded[pkgName] = version
	}

	added, upgraded, downgraded, removed = diff.DiffPackages(localAdded, remoteAdded)
	return
}
