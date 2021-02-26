package mfs

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

type FSTestSuite struct {
	suite.Suite
}

func (suite *FSTestSuite) SetupTest() {
}

func TestFSTestSuite(t *testing.T) {
	suite.Run(t, new(FSTestSuite))
}

func (suite *FSTestSuite) TestEventOnUnEvent() {
	drv := NewFSEventDriver()
	var count int
	var count2 int
	fn := func(FileRenamedEvent) { count += 1 }
	stub := drv.OnEvent(fn)
	drv.OnEvent(func(e interface{}) {
		suite.T().Logf("%+v", e)
		count2 += 1
	})
	drv.Trigger(FileRenamedEvent{New: "new"})
	suite.Equal(1, count)
	stub.Unbind()
	drv.Trigger(FileRenamedEvent{})
	suite.Equal(1, count)
	suite.Equal(2, count2)
}

func (suite *FSTestSuite) TestFileRename() {
	drv := NewFSEventDriver()
	fs := Create(drv, newOs(), "")
	var delFile, renameFrom, renameTo string
	fn := func(e FileRenamedEvent) {
		suite.T().Logf("FileRenamedEvent %s->%s", e.Old, e.New)
		renameFrom = e.Old
		renameTo = e.New
	}
	fn2 := func(e FileRemovedEvent) {
		suite.T().Logf("FileRemovedEvent %s", e.Name)
		delFile = e.Name
	}
	fn3 := func(e FileCreatedEvent) {
		suite.T().Logf("FileCreatedEvent %s", e.Name)
	}
	drv.OnEvent(fn)
	drv.OnEvent(fn2)
	drv.OnEvent(fn3)
	drv.OnEvent(func(e FileModifiedEvent) {
		suite.T().Logf("FileModifiedEvent %s", e.Name)
	})

	fs.CreateFile("/aaa")
	fs.CreateFile("/bbb")
	fs.Rename("/aaa", "/bbb")
	fs.GetFile("/bbb").SetContent([]byte("Hello"))

	suite.Equal("/bbb", delFile)
	suite.Equal("/aaa", renameFrom)
	suite.Equal("/bbb", renameTo)

	files := fs.ReadDir("/", true)
	suite.Len(files, 1)
	suite.Equal("/bbb", files[0].Name())
	suite.Equal("Hello", string(files[0].Content()))
}

func (suite *FSTestSuite) TestDiskRead() {
	dir, err := ioutil.TempDir("/tmp", "fstest")
	suite.Nil(err)
	defer os.RemoveAll(dir)
	filename := filepath.Join(dir, "ttt")
	content1 := `aaa`
	content2 := `bbb`
	ioutil.WriteFile(filename, []byte(content1), 0755)
	body, _ := ioutil.ReadFile(filename)
	suite.Equal(content1, string(body))

	fs := New(dir)
	file := fs.GetFile("/ttt")
	suite.Equal(content1, string(file.Content()))
	suite.Equal(content1, string(file.Content()))

	file.SetContent([]byte(content2))
	suite.Equal(content2, string(file.Content()))
	suite.Equal(content2, string(file.Content()))
}

func (suite *FSTestSuite) TestFile() {
	f := createFile(fakeED{}, newFakeOS(), "", "/aaa")
	suite.Equal("/aaa", f.Name())
	suite.Nil(f.Content())
	suite.False(f.IsDirty())

	f.SetContent([]byte("A"))
	suite.Equal("A", string(f.Content()))
	suite.True(f.IsDirty())

	buf := new(bytes.Buffer)
	f.Read(func(r io.Reader) { io.Copy(buf, r) })
	suite.Equal("A", buf.String())

	f.Map(func(r io.Reader, w io.Writer) {
		w.Write([]byte("B"))
	})
	suite.Equal("B", string(f.Content()))

	f.Map(func(r io.Reader, w io.Writer) {
		w.Write([]byte("C"))
	})
	suite.Equal("C", string(f.Content()))
}

func (suite *FSTestSuite) TestRepeatRead() {
	f := createFile(fakeED{}, newFakeOS(), "", "/aaa")
	suite.Equal("/aaa", f.Name())
	suite.Nil(f.Content())
	suite.False(f.IsDirty())

	f.SetContent([]byte("A"))
	suite.Equal("A", string(f.Content()))
	suite.Equal("A", string(f.Content()))
}

func (suite *FSTestSuite) TestRemove() {
	dir, err := ioutil.TempDir("/tmp", "rt")
	suite.Nil(err)
	defer os.RemoveAll(dir)
	fs := New(dir)
	fs.CreateFile("/a/b/c")
	fs.CreateFile("/ab/c")
	fs.Remove("/a")
	suite.Equal("/ab/c", fs.ListFile()[0].Name())

	fs = New(dir)
	fs.CreateFile("/a/b/c")
	fs.CreateFile("/ab/c")
	fs.Remove("/a/b/c")
	suite.Equal("/ab/c", fs.ListFile()[0].Name())
}

func (suite *FSTestSuite) TestWithPrefix() {
	dir, err := ioutil.TempDir("/tmp", "rt")
	suite.Nil(err)
	defer os.RemoveAll(dir)
	ioutil.WriteFile(join(dir, "aaa"), []byte("000"), 0755)
	fs := NewWithOptions(dir, WithPrefix("/chroot"))
	suite.Len(fs.ListFile(), 1)
	suite.Equal("000", string(fs.GetFile("/chroot/aaa").Content()))
}

func (suite *FSTestSuite) TestDirRename() {
	dir, err := ioutil.TempDir("/tmp", "2rt")
	suite.Nil(err)
	defer os.RemoveAll(dir)
	fs := New(dir)
	fs.CreateFile("/a/b/c")
	fs.CreateFile("/a/b/e")
	fs.CreateFile("/ab/c")
	fs.CreateFile("/ab/d")
	fs.Rename("/a/b", "/ab")

	suite.Len(fs.ListFile(), 3)
	suite.ElementsMatch([]string{"/ab/c", "/ab/e", "/ab/d"}, getFileNames(fs.ListFile()))
}

func (suite *FSTestSuite) TestDirRename2() {
	dir, err := ioutil.TempDir("/tmp", "22rt")
	suite.Nil(err)
	defer os.RemoveAll(dir)
	fs := New(dir)
	fs.CreateFile("/a/c")
	fs.CreateFile("/a/e")
	fs.Rename("/", "/ext")

	suite.Len(fs.ListFile(), 2)
	suite.ElementsMatch([]string{"/ext/a/c", "/ext/a/e"}, getFileNames(fs.ListFile()))
}

func (suite *FSTestSuite) TestCopyFile() {
	dir, err := ioutil.TempDir("/tmp", "21rt")
	suite.Nil(err)
	defer os.RemoveAll(dir)

	fs := New(dir)
	fs.CreateFile("/a/b/c").SetContent([]byte("A"))
	fs.Copy("/a/b/c", "/a/x")

	suite.Equal("A", string(fs.GetFile("/a/b/c").Content()))
	suite.Equal("A", string(fs.GetFile("/a/x").Content()))

	fs.GetFile("/a/x").SetContent([]byte("B"))
	suite.Equal("A", string(fs.GetFile("/a/b/c").Content()))
	suite.Equal("B", string(fs.GetFile("/a/x").Content()))
}

func (suite *FSTestSuite) TestDropIfExist() {
	dir, err := ioutil.TempDir("/tmp", "22rt")
	suite.Nil(err)
	defer os.RemoveAll(dir)

	dir1, err := ioutil.TempDir("/tmp", "23rt")
	suite.Nil(err)
	defer os.RemoveAll(dir1)
	afile := filepath.Join(dir1, "a")
	ioutil.WriteFile(afile, []byte("X"), 0755)

	fs := New(dir)
	fs.CreateFile("/a").SetContent([]byte("A"))
	fs.CreateFile("/b").SetContent([]byte("B"))
	fs.DropIfExist("/a")

	fs.Persist(dir1)

	data, _ := ioutil.ReadFile(afile)
	suite.Equal("X", string(data))
	data, _ = ioutil.ReadFile(filepath.Join(dir1, "b"))
	suite.Equal("B", string(data))
}

func (suite *FSTestSuite) TestDirDropIfExist() {
	dir, err := ioutil.TempDir("/tmp", "22rt2")
	suite.Nil(err)
	defer os.RemoveAll(dir)

	dir1, err := ioutil.TempDir("/tmp", "23rt1")
	suite.Nil(err)
	defer os.RemoveAll(dir1)
	afile := filepath.Join(dir1, "a")
	ioutil.WriteFile(afile, []byte("X"), 0755)

	fs := New(dir)
	fs.CreateFile("/a").SetContent([]byte("A"))
	fs.CreateFile("/b").SetContent([]byte("B"))
	fs.DropIfExist("/")

	fs.Persist(dir1)

	data, _ := ioutil.ReadFile(afile)
	suite.Equal("X", string(data))
	data, _ = ioutil.ReadFile(filepath.Join(dir1, "b"))
	suite.Equal("B", string(data))
}

func (suite *FSTestSuite) TestDirDropIfExistBeforeOp() {
	dir, err := ioutil.TempDir("/tmp", "22rt2")
	suite.Nil(err)
	defer os.RemoveAll(dir)

	dir1, err := ioutil.TempDir("/tmp", "23rt1")
	suite.Nil(err)
	defer os.RemoveAll(dir1)
	afile := filepath.Join(dir1, "a")
	ioutil.WriteFile(afile, []byte("X"), 0755)

	fs := New(dir)
	fs.DropIfExist("/")
	fs.CreateFile("/a").SetContent([]byte("A"))
	fs.CreateFile("/b").SetContent([]byte("B"))

	fs.Persist(dir1)

	data, _ := ioutil.ReadFile(afile)
	suite.Equal("X", string(data))
	data, _ = ioutil.ReadFile(filepath.Join(dir1, "b"))
	suite.Equal("B", string(data))
}
func (suite *FSTestSuite) TestDirDropIfExist2() {
	dir, err := ioutil.TempDir("/tmp", "22rt")
	suite.Nil(err)
	defer os.RemoveAll(dir)

	dir1, err := ioutil.TempDir("/tmp", "23rt")
	suite.Nil(err)
	defer os.RemoveAll(dir1)
	afile := filepath.Join(dir1, "f", "a")
	os.MkdirAll(filepath.Dir(afile), 0755)
	ioutil.WriteFile(afile, []byte("X"), 0755)

	fs := New(dir)
	fs.CreateFile("/f/a").SetContent([]byte("A"))
	fs.CreateFile("/b").SetContent([]byte("B"))
	fs.DropIfExist("/f")

	fs.Persist(dir1)

	data, _ := ioutil.ReadFile(afile)
	suite.Equal("X", string(data))
	data, _ = ioutil.ReadFile(filepath.Join(dir1, "b"))
	suite.Equal("B", string(data))
}

func (suite *FSTestSuite) TestCopy() {
	dir, err := ioutil.TempDir("/tmp", "221rt")
	suite.Nil(err)
	defer os.RemoveAll(dir)

	fs := New(dir)
	fs.CreateFile("/a/b/c").SetContent([]byte("A"))
	fs.CreateFile("/a/d").SetContent([]byte("A1"))
	fs.Copy("/a", "/b")

	suite.Equal("A", string(fs.GetFile("/a/b/c").Content()))
	suite.Equal("A1", string(fs.GetFile("/a/d").Content()))
	suite.Equal("A", string(fs.GetFile("/b/b/c").Content()))
	suite.Equal("A1", string(fs.GetFile("/b/d").Content()))
}

func (suite *FSTestSuite) TestFS() {
	fakeFiles := map[string]string{}
	fos := newFakeOSWithContent(fakeFiles)
	fs := Create(NewFSEventDriver(), fos, "")
	f1 := fs.CreateFile("/e/bbb")
	f1.SetContent([]byte("BBB"))

	files := fs.ListFile()
	suite.Len(files, 1)
	files = fs.ReadDir("/e", true)
	suite.Len(files, 1)

	fs.Rename("/e/bbb", "/fx/ttt")
	files = fs.ListFile()
	suite.Len(files, 1)
	files = fs.ReadDir("/e", true)
	suite.Len(files, 0)
	files = fs.ReadDir("/fx", true)
	suite.Len(files, 1)

	suite.Equal("BBB", string(fs.GetFile("/fx/ttt").Content()))

	suite.True(fs.Exist("/fx/ttt"))
	suite.False(fs.Exist("/e/bbb"))

	fs.Remove("/fx/ttt")
	suite.Len(fs.ListFile(), 0)
	suite.Len(fs.ReadDir("/fx", true), 0)
	suite.Nil(fs.GetFile("/fx/ttt"))

	fos = newFakeOSWithContent(map[string]string{
		"/t/1": "EEE",
	})
	fs = Create(NewFSEventDriver(), fos, "")
	// create f1
	f1 = fs.CreateFile("/a/b/c/e/1")
	f1.Map(func(r io.Reader, w io.Writer) {
		w.Write([]byte("111"))
	})
	// create f2
	f2 := fs.CreateFile("/a/b/ggg/1")
	f2.SetContent([]byte("222"))
	// create f3
	f3 := fs.CreateFile("/t/1")
	f3.SetContent([]byte("333"))

	fs.Rename("/a/b/ggg/1", "/a/b/ggg/2")
	suite.Equal("/a/b/ggg/2", f2.Name())
	fs.Rename("/a/b/c/e/1", "/a/b/ggg/1")

	gggFiles := fs.ReadDir("/a/b/", true)
	suite.Len(gggFiles, 2)
	gggFiles = fs.ReadDir("/a/b/ggg", true)
	suite.Len(gggFiles, 2)

	fs.Persist("/eeee")
	suite.Equal("333", string(fos.Files["/eeee/t/1"]))
	suite.T().Log(fos.Files)
}

func (suite *FSTestSuite) TestReadDir() {
	fs := Create(NewFSEventDriver(), newFakeOS(), "")
	fs.CreateFile("/a/b/c/e/1")
	fs.CreateFile("/a/b/ggg/1")
	fs.CreateFile("/t/1")
	suite.Len(fs.ListFile(), 3)
	suite.Len(fs.ReadDir("/", true), 3)
	suite.Len(fs.ReadDir("/", false), 0)
	suite.Len(fs.ReadDir("/a", false), 0)
	suite.Len(fs.ReadDir("/t", false), 1)
	suite.Len(fs.ReadDir("/a/b/ggg", false), 1)
}

type fakeOS struct {
	Files map[string]string
}

func newFakeOS() *fakeOS {
	return &fakeOS{Files: make(map[string]string)}
}

func newFakeOSWithContent(f map[string]string) *fakeOS {
	return &fakeOS{Files: f}
}

func (fos *fakeOS) Walk(rootDir string, includeHidden bool, cb func(string)) {
	for _, f := range fos.Files {
		cb(f)
	}
}

func (fos *fakeOS) Exist(f string) bool {
	_, ok := fos.Files[f]
	return ok
}

func (fos *fakeOS) Open(file string) (io.ReadCloser, error) {
	if c, ok := fos.Files[file]; ok {
		return NewBuffer([]byte(c)), nil
	}
	return nil, nil
}

func (fos fakeOS) Remove(file string) error {
	delete(fos.Files, file)
	return nil
}

func (fos fakeOS) MkdirAll(dir string) error {
	return nil
}

func (fos fakeOS) WriteToFile(filename string, r io.Reader) error {
	data, _ := ioutil.ReadAll(r)
	fos.Files[filename] = string(data)
	return nil
}

type fakeED struct{}

func (fakeED) Trigger(interface{})      {}
func (fakeED) OnEvent(interface{}) Stub { return nil }
