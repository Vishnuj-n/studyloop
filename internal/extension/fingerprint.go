package extension

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// FingerprintFilename is the metadata file written inside the extension's .venv directory.
const FingerprintFilename = ".studyloop_fingerprint"

// EnvironmentFingerprint stores the recorded manifest version and requirements checksum.
type EnvironmentFingerprint struct {
	ManifestVersion string `json:"manifest_version"`
	RequirementsSHA string `json:"requirements_sha256"`
}

// ComputeFingerprint calculates the current fingerprint from the extension's manifest and requirements.txt.
func ComputeFingerprint(ext *Extension) EnvironmentFingerprint {
	if ext == nil {
		return EnvironmentFingerprint{}
	}
	ext.EnsureManifestLoaded()

	reqSHA := ""
	reqPath := filepath.Join(ext.Dir, "requirements.txt")
	if data, err := os.ReadFile(reqPath); err == nil {
		hash := sha256.Sum256(data)
		reqSHA = hex.EncodeToString(hash[:])
	}

	return EnvironmentFingerprint{
		ManifestVersion: strings.TrimSpace(ext.Version()),
		RequirementsSHA: reqSHA,
	}
}

// ReadFingerprint loads the stored fingerprint from an extension's virtual environment directory.
func ReadFingerprint(venvDir string) (EnvironmentFingerprint, error) {
	fpPath := filepath.Join(venvDir, FingerprintFilename)
	data, err := os.ReadFile(fpPath)
	if err != nil {
		return EnvironmentFingerprint{}, err
	}
	var fp EnvironmentFingerprint
	if err := json.Unmarshal(data, &fp); err != nil {
		return EnvironmentFingerprint{}, err
	}
	return fp, nil
}

// WriteFingerprint persists the current fingerprint in the extension's virtual environment directory.
func WriteFingerprint(venvDir string, fp EnvironmentFingerprint) error {
	fpPath := filepath.Join(venvDir, FingerprintFilename)
	data, err := json.MarshalIndent(fp, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(fpPath, data, 0644)
}

// ParseMajorVersion extracts the leading integer from a semver string (e.g. "1.0.1" -> 1).
func ParseMajorVersion(v string) (int, bool) {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	parts := strings.Split(v, ".")
	if len(parts) == 0 || parts[0] == "" {
		return 0, false
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, false
	}
	return major, true
}

// ShouldRecreateVenv returns true if the major version changed, indicating a breaking update
// where virtual environment recreation from scratch is required.
func ShouldRecreateVenv(oldVer, newVer string) bool {
	oldMajor, okOld := ParseMajorVersion(oldVer)
	newMajor, okNew := ParseMajorVersion(newVer)
	if okOld && okNew && oldMajor != newMajor {
		return true
	}
	return false
}

// IsExtensionVenvStale checks if the virtual environment is missing, outdated compared to manifest version,
// or has modified requirements.txt. Returns (isStale, isMajorChange, reason).
func IsExtensionVenvStale(ext *Extension, venvDir string) (bool, bool, string) {
	if ext == nil {
		return true, false, "extension is nil"
	}
	ext.EnsureManifestLoaded()

	pyPath := GetVenvPython(venvDir)
	if info, err := os.Stat(pyPath); err != nil || info.IsDir() {
		return true, false, "Virtual environment not initialized"
	}

	storedFp, err := ReadFingerprint(venvDir)
	if err != nil {
		return true, false, "Environment fingerprint missing (update required)"
	}

	currentFp := ComputeFingerprint(ext)

	// 1. Check major version bump -> requires full venv recreation
	if ShouldRecreateVenv(storedFp.ManifestVersion, currentFp.ManifestVersion) {
		return true, true, fmt.Sprintf("Extension major version changed (%s -> %s), full environment recreation required", storedFp.ManifestVersion, currentFp.ManifestVersion)
	}

	// 2. Check minor/patch version bump -> requires update
	if storedFp.ManifestVersion != currentFp.ManifestVersion {
		return true, false, fmt.Sprintf("Extension version changed (%s -> %s), update required", storedFp.ManifestVersion, currentFp.ManifestVersion)
	}

	// 3. Check requirements.txt modified -> requires update
	if storedFp.RequirementsSHA != currentFp.RequirementsSHA {
		return true, false, "Extension requirements.txt modified, dependency update required"
	}

	return false, false, ""
}
