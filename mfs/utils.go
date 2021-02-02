package mfs

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

func absOfFile(f string) string {
	if isStrBlank(f) {
		return f
	}
	if strings.HasPrefix(f, "~") {
		if u, _ := user.Current(); u != nil {
			f = join(u.HomeDir, f[1:])
		}
	} else if strings.HasPrefix(f, "$") {
		env := scanString(f, "/")
		f = join(os.Getenv(strings.TrimPrefix(env, "$")), strings.TrimPrefix(f, env))
	}
	f, _ = filepath.Abs(f)
	return f
}

func baseName(f string) string {
	return filepath.Base(f)
}

func dirName(f string) string {
	return filepath.Dir(f)
}

func fmtDirWithSlash(d string) string {
	if d == "/" {
		return d
	}
	if !strings.HasSuffix(d, "/") {
		return d + "/"
	}
	return d
}

func fmtDirWithoutSlash(d string) string {
	if d == "/" {
		return d
	}
	if strings.HasSuffix(d, "/") {
		return strings.TrimSuffix(d, "/")
	}
	return d
}

func prependSlash(f string) string {
	if !strings.HasPrefix(f, "/") && !isStrBlank(f) {
		return "/" + f
	}
	return f
}

func trimPrefix(filename string, prefix string) string {
	if isStrBlank(prefix) {
		return filename
	}
	prefix = fmtDirWithoutSlash(prefix)
	if prefix != "/" {
		filename = strings.TrimPrefix(filename, prefix)
	}
	return filename
}

func join(names ...string) string { return filepath.Join(names...) }

func scanString(str, stop string) string {
	if idx := strings.Index(str, stop); idx != -1 {
		return str[:idx]
	}
	return str
}

func isFileInDir(file, dir string) bool {
	if !strings.HasSuffix(dir, "/") {
		dir += "/"
	}
	return strings.HasPrefix(file, dir) && file != dir
}

func isStrBlank(s string) bool { return s == "" }

func getFileNames(fs []File) (out []string) {
	for _, f := range fs {
		out = append(out, f.Name())
	}
	return
}
