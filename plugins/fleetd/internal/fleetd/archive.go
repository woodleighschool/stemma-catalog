package fleetd

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
)

// Transcode into the payload TAR without extracting or executing vendor files.
func appendBundle(ctx context.Context, output *tar.Writer, directory, bundle, source string) (string, error) {
	f, err := os.Open(source)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	hash := sha512.New()
	gz, err := gzip.NewReader(io.TeeReader(contextReader{ctx, f}, hash))
	if err != nil {
		return "", err
	}
	defer func() { _ = gz.Close() }()
	limited := &io.LimitedReader{R: gz, N: (2 << 30) + 1}
	reader := tar.NewReader(limited)
	seen := map[string]bool{}
	executable := false
	for {
		h, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", err
		}
		name := strings.TrimSuffix(strings.TrimPrefix(h.Name, "./"), "/")
		if name == "." && h.Typeflag == tar.TypeDir {
			continue
		}
		if !fs.ValidPath(name) || strings.ContainsAny(name, "\\\x00") || (name != bundle && !strings.HasPrefix(name, bundle+"/")) || seen[name] {
			return "", errors.New("archive contains an unexpected or repeated bundle path")
		}
		seen[name] = true
		if len(seen) > 100000 || h.Size < 0 || h.Size > limited.N || limited.N <= 0 {
			return "", errors.New("bundle exceeds extraction limits")
		}
		for key := range h.PAXRecords {
			if strings.HasPrefix(key, "SCHILY.xattr.") || strings.HasPrefix(key, "LIBARCHIVE.xattr.") {
				return "", errors.New("bundle contains extended attributes the package cannot preserve")
			}
		}
		if h.Mode & ^int64(0o777) != 0 {
			return "", errors.New("bundle contains special permission bits")
		}
		switch h.Typeflag {
		case tar.TypeReg, tar.TypeDir:
		case tar.TypeSymlink:
			target := path.Join(path.Dir(name), h.Linkname)
			if path.IsAbs(h.Linkname) || strings.ContainsAny(h.Linkname, "\\\x00") || (target != bundle && !strings.HasPrefix(target, bundle+"/")) {
				return "", errors.New("bundle symlink escapes the application")
			}
		default:
			return "", fmt.Errorf("unsupported bundle archive entry type %d", h.Typeflag)
		}
		if h.Typeflag == tar.TypeReg && strings.HasPrefix(name, bundle+"/Contents/MacOS/") && h.Mode&0o111 != 0 {
			executable = true
		}
		copyHeader := &tar.Header{Name: path.Join(directory, name), Typeflag: h.Typeflag, Mode: h.Mode,
			Size: h.Size, Linkname: h.Linkname, Format: tar.FormatPAX}
		if err := output.WriteHeader(copyHeader); err != nil {
			return "", err
		}
		if _, err := io.Copy(output, contextReader{ctx, reader}); err != nil {
			return "", err
		}
	}
	if !executable {
		return "", errors.New("bundle has no executable in Contents/MacOS")
	}
	// Finish gzip validation and hashing, including padding after the TAR terminator.
	if _, err := io.Copy(io.Discard, limited); err != nil {
		return "", err
	}
	if limited.N == 0 {
		return "", errors.New("bundle exceeds extraction limits")
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
