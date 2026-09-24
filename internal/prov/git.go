package prov

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// gitRepo reads git objects from a local clone. It never reads the working tree.
type gitRepo struct {
	dir string
}

func (g gitRepo) run(args ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{"-C", g.dir}, args...)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s: %s", strings.Join(args, " "), strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

func (g gitRepo) fetch() error {
	_, err := g.run("fetch", "--quiet", "origin")
	return err
}

// commit resolves sha to a full commit SHA and errors unless some remote
// branch or tag contains it: a commit on no branch is lost to a fresh clone.
func (g gitRepo) commit(sha string) (string, error) {
	out, err := g.run("rev-parse", "--verify", "--quiet", sha+"^{commit}")
	if err != nil {
		return "", fmt.Errorf("commit %s not found", sha)
	}
	full := strings.TrimSpace(string(out))
	refs, err := g.run("for-each-ref", "--count=1", "--contains", full, "refs/remotes", "refs/tags")
	if err != nil {
		return "", err
	}
	if len(bytes.TrimSpace(refs)) == 0 {
		return "", fmt.Errorf("commit %s is unreachable from any remote branch or tag", sha)
	}
	return full, nil
}

// blobID returns the blob SHA of path at rev, or an error if path is missing there.
func (g gitRepo) blobID(rev, path string) (string, error) {
	out, err := g.run("rev-parse", "--verify", "--quiet", rev+":"+path)
	if err != nil {
		return "", fmt.Errorf("%s missing at %s", path, short(rev))
	}
	return strings.TrimSpace(string(out)), nil
}

func (g gitRepo) show(rev, path string) ([]byte, error) {
	out, err := g.run("cat-file", "blob", rev+":"+path)
	if err != nil {
		return nil, fmt.Errorf("%s missing at %s", path, short(rev))
	}
	return out, nil
}

func short(sha string) string {
	if len(sha) > 8 {
		return sha[:8]
	}
	return sha
}
