package sys

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

type MoveFilesTestSuite struct {
	suite.Suite
}

func (suite *MoveFilesTestSuite) SetupTest() {
}

func TestMoveFilesTestSuite(t *testing.T) {
	suite.Run(t, new(MoveFilesTestSuite))
}

func (suite *MoveFilesTestSuite) TestMoveFilesSimple() {
	existDir := make(map[string]int)
	fromDir, toDir := "/from", "/to"
	twalkDir := func(d string) []string {
		return []string{
			"/from/a/a1",
			"/from/a/b/b1",
			"/from/a/b/e/e1",
			"/from/c/c1",
		}
	}
	tmkDir := func(d string) error {
		existDir[d]++
		return nil
	}
	tisExist := func(d string) bool { return existDir[d] > 0 }
	tmvFile := func(string, string) error { return nil }
	err := _MoveFiles(fromDir, toDir, nil, twalkDir, tmkDir, tisExist, tmvFile)
	suite.Nil(err)
	suite.Equal(2, len(existDir))
	suite.Equal(1, existDir["/to/c"])
	suite.Equal(1, existDir["/to/a/b/e"])
}

func (suite *MoveFilesTestSuite) TestMoveFilesOmit() {
	existDir := make(map[string]int)
	fromDir, toDir := "/from", "/to"
	fshouldMove := func(f string, t string) bool {
		if f == "/from/a/b/e/e1" {
			return false
		}
		return true
	}
	twalkDir := func(d string) []string {
		return []string{
			"/from/a/a1",
			"/from/a/b/b1",
			"/from/a/b/e/e1",
			"/from/c/c1",
		}
	}
	tmkDir := func(d string) error {
		existDir[d]++
		return nil
	}
	tisExist := func(d string) bool { return existDir[d] > 0 }
	tmvFile := func(string, string) error { return nil }
	err := _MoveFiles(fromDir, toDir, fshouldMove, twalkDir, tmkDir, tisExist, tmvFile)
	suite.Nil(err)
	suite.Equal(2, len(existDir))
	suite.Equal(1, existDir["/to/c"])
	suite.Equal(1, existDir["/to/a/b"])
}

func (suite *MoveFilesTestSuite) TestAbs() {
	home, _ := os.UserHomeDir()
	os.Setenv("t1", "/T1")
	os.Setenv("t2", "T2")
	suite.Equal("/T1/sp/T2", Abs("$t1/sp/$t2"))

	suite.Equal(filepath.Join(home, "local", "T2"), Abs("~/local/$t2"))

	suite.Equal("/T1", Abs("$t1"))
	suite.Equal("/T1/a/b", Abs("$t1/a/b/c/./.."))
}
