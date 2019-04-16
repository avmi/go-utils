package pkgmgmtutils

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	ErrNoSha256sFound = errors.New("pkgmgmtutils: did not find all needed sha256 files")
	ErrNoShaDataFound = errors.New("pkgmgmtutils: no data in SHA256 file")
)

type sha256Outputs struct {
	darwinSha  []byte // sha256 for <ctl>-darwin binary
	linuxSha   []byte // sha256 for <ctl>-linux binary
	windowsSha []byte // sha256 for <ctl>-windows binary
}

// getLocalBinarySha256 extracts the sha256 strings from existing .sha256 files created as part of the build process.
// Those .sha256 files need to be located in the outputDir directory.
// It returns the sha256s and any read errors encountered. It will also return ErrNoSha256sFound if any of the platform
// shas are found.
func getLocalBinarySha256(outputDir string) (*sha256Outputs, error) {
	shas := sha256Outputs{}

	// Scan outputDir directory looking for any files that match the reOS regular expression as targets for extraction
	reOS := regexp.MustCompile(`(darwin|linux|windows)`)

	files, _ := ioutil.ReadDir(outputDir)

	for _, f := range files {
		filename := filepath.Join(outputDir, f.Name())
		s := reOS.FindStringSubmatch(filename)
		if s == nil {
			continue
		}

		var err error

		switch s[1] {
		case "darwin":
			shas.darwinSha, err = extractShaFromFile(filename)
		case "linux":
			shas.linuxSha, err = extractShaFromFile(filename)
		case "windows":
			shas.windowsSha, err = extractShaFromFile(filename)
		}
		if err != nil {
			return nil, err
		}
	}
	if shas.darwinSha == nil || shas.linuxSha == nil || shas.windowsSha == nil {
		return nil, ErrNoSha256sFound
	}

	return &shas, nil
}

// extractShaFromFile extracts the first field from filename expecting it to be a sha.
// Expected file format is two string fields "<sha> <binary name>".
// It returns the sha field as a []byte and any read errors.
func extractShaFromFile(filename string) ([]byte, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	if !(len(b) > 0) {
		return nil, ErrNoShaDataFound
	}

	s := strings.Fields(string(b))
	if len(s) != 2 {
		return nil, fmt.Errorf("pkgmgmtutils: Sha256 file %s is not in expected format", filename)
	}

	return []byte(s[0]), nil
}

// replaceSubmatch will replace the submatch group with repl within the matching regular expression of the src input.
// It expects there to be only one submatch group within the re regular expression.
// Since there's no golang replaceAllSubmatch function, need to do this double regex hack
func replaceSubmatch(src []byte, repl []byte, re *regexp.Regexp) []byte {
	return re.ReplaceAllFunc(src, func(m []byte) []byte {
		ind := re.FindSubmatchIndex(m)
		if ind == nil {
			return src
		}

		if len(ind) != 4 || ind[0] != 0 || ind[1] != len(m) {
			panic("Regular expression does not meet preconditions")
		}

		var b bytes.Buffer
		b.Grow(ind[2] + len(repl) + (len(m) - ind[3]))
		b.Write(m[:ind[2]])
		b.Write(repl)
		b.Write(m[ind[3]:])
		return b.Bytes()
	})
}

// Since there's no golang replaceAllSubmatch function, need to do this double regex hack
// Will replace single submatch group within re against src with repl
// Assumes there is one and only one submatch group within re
func replaceSubmatchString(src string, repl string, re *regexp.Regexp) string {
	return re.ReplaceAllStringFunc(src, func(m string) string {
		ind := re.FindStringSubmatchIndex(m)
		if ind == nil {
			return src
		}

		if ind[0] != 0 || ind[1] != len(m) {
			panic("Regular expression does not meet preconditions")
		}

		m1 := m[:ind[2]]
		m2 := m[ind[3]:]

		var b strings.Builder
		b.Grow(len(m1) + len(repl) + len(m2))
		b.WriteString(m1)
		b.WriteString(repl)
		b.WriteString(m2)
		return b.String()
	})
}

// Since there's no golang replaceAllSubmatch function, need to do this double regex hack
// For each re1 matches in src, replace all re2 matches with repl
func replaceAllSubmatch(src []byte, repl []byte, re1 *regexp.Regexp, re2 *regexp.Regexp) []byte {
	return re1.ReplaceAllFunc(src, func(m []byte) []byte {
		return re2.ReplaceAll(m, repl)
	})
}

// Since there's no golang replaceSubmatchString function, need to do this double regex hack
// For each re1 matches in src, replace all re2 matches with repl
func replaceAllSubmatchString(src string, repl string, re1 *regexp.Regexp, re2 *regexp.Regexp) string {
	return re1.ReplaceAllStringFunc(src, func(m string) string {
		return re2.ReplaceAllString(m, repl)
	})
}
