package files

import (
	"io/ioutil"
	. "launchpad.net/gocheck"
	"os"
	"path/filepath"
	"syscall"
)

type PublishedStorageSuite struct {
	root    string
	storage *PublishedStorage
}

var _ = Suite(&PublishedStorageSuite{})

func (s *PublishedStorageSuite) SetUpTest(c *C) {
	s.root = c.MkDir()
	s.storage = NewPublishedStorage(s.root)
}

func (s *PublishedStorageSuite) TestPublicPath(c *C) {
	c.Assert(s.storage.PublicPath(), Equals, filepath.Join(s.root, "public"))
}

func (s *PublishedStorageSuite) TestMkDir(c *C) {
	err := s.storage.MkDir("ppa/dists/squeeze/")
	c.Assert(err, IsNil)

	_, err = os.Stat(filepath.Join(s.storage.rootPath, "ppa/dists/squeeze/"))
	c.Assert(err, IsNil)
}

func (s *PublishedStorageSuite) TestCreateFile(c *C) {
	err := s.storage.MkDir("ppa/dists/squeeze/")
	c.Assert(err, IsNil)

	file, err := s.storage.CreateFile("ppa/dists/squeeze/Release")
	c.Assert(err, IsNil)
	defer file.Close()

	_, err = os.Stat(filepath.Join(s.storage.rootPath, "ppa/dists/squeeze/Release"))
	c.Assert(err, IsNil)
}

func (s *PublishedStorageSuite) TestFilelist(c *C) {
	err := s.storage.MkDir("ppa/pool/main/a/ab/")
	c.Assert(err, IsNil)

	file, err := s.storage.CreateFile("ppa/pool/main/a/ab/a.deb")
	c.Assert(err, IsNil)
	defer file.Close()

	file2, err := s.storage.CreateFile("ppa/pool/main/a/ab/b.deb")
	c.Assert(err, IsNil)
	defer file2.Close()

	list, err := s.storage.Filelist("ppa/pool/main/")
	c.Check(err, IsNil)
	c.Check(list, DeepEquals, []string{"a/ab/a.deb", "a/ab/b.deb"})
}

func (s *PublishedStorageSuite) TestRenameFile(c *C) {
	err := s.storage.MkDir("ppa/dists/squeeze/")
	c.Assert(err, IsNil)

	file, err := s.storage.CreateFile("ppa/dists/squeeze/Release")
	c.Assert(err, IsNil)
	defer file.Close()

	err = s.storage.RenameFile("ppa/dists/squeeze/Release", "ppa/dists/squeeze/InRelease")
	c.Check(err, IsNil)

	_, err = os.Stat(filepath.Join(s.storage.rootPath, "ppa/dists/squeeze/InRelease"))
	c.Assert(err, IsNil)
}

func (s *PublishedStorageSuite) TestRemoveDirs(c *C) {
	err := s.storage.MkDir("ppa/dists/squeeze/")
	c.Assert(err, IsNil)

	file, err := s.storage.CreateFile("ppa/dists/squeeze/Release")
	c.Assert(err, IsNil)
	defer file.Close()

	err = s.storage.RemoveDirs("ppa/dists/", nil)

	_, err = os.Stat(filepath.Join(s.storage.rootPath, "ppa/dists/squeeze/Release"))
	c.Assert(err, NotNil)
	c.Assert(os.IsNotExist(err), Equals, true)
}

func (s *PublishedStorageSuite) TestRemove(c *C) {
	err := s.storage.MkDir("ppa/dists/squeeze/")
	c.Assert(err, IsNil)

	file, err := s.storage.CreateFile("ppa/dists/squeeze/Release")
	c.Assert(err, IsNil)
	defer file.Close()

	err = s.storage.Remove("ppa/dists/squeeze/Release")

	_, err = os.Stat(filepath.Join(s.storage.rootPath, "ppa/dists/squeeze/Release"))
	c.Assert(err, NotNil)
	c.Assert(os.IsNotExist(err), Equals, true)
}

func (s *PublishedStorageSuite) TestLinkFromPool(c *C) {
	tests := []struct {
		prefix           string
		component        string
		sourcePath       string
		poolDirectory    string
		expectedFilename string
	}{
		{ // package name regular
			prefix:           "",
			component:        "main",
			sourcePath:       "pool/01/ae/mars-invaders_1.03.deb",
			poolDirectory:    "m/mars-invaders",
			expectedFilename: "pool/main/m/mars-invaders/mars-invaders_1.03.deb",
		},
		{ // lib-like filename
			prefix:           "",
			component:        "main",
			sourcePath:       "pool/01/ae/libmars-invaders_1.03.deb",
			poolDirectory:    "libm/libmars-invaders",
			expectedFilename: "pool/main/libm/libmars-invaders/libmars-invaders_1.03.deb",
		},
		{ // duplicate link, shouldn't panic
			prefix:           "",
			component:        "main",
			sourcePath:       "pool/01/ae/mars-invaders_1.03.deb",
			poolDirectory:    "m/mars-invaders",
			expectedFilename: "pool/main/m/mars-invaders/mars-invaders_1.03.deb",
		},
		{ // prefix & component
			prefix:           "ppa",
			component:        "contrib",
			sourcePath:       "pool/01/ae/libmars-invaders_1.04.deb",
			poolDirectory:    "libm/libmars-invaders",
			expectedFilename: "pool/contrib/libm/libmars-invaders/libmars-invaders_1.04.deb",
		},
	}

	pool := NewPackagePool(s.root)

	for _, t := range tests {
		t.sourcePath = filepath.Join(s.root, t.sourcePath)

		err := os.MkdirAll(filepath.Dir(t.sourcePath), 0755)
		c.Assert(err, IsNil)

		err = ioutil.WriteFile(t.sourcePath, []byte("Contents"), 0644)
		c.Assert(err, IsNil)

		err = s.storage.LinkFromPool(filepath.Join(t.prefix, "pool", t.component, t.poolDirectory), pool, t.sourcePath)
		c.Assert(err, IsNil)

		st, err := os.Stat(filepath.Join(s.storage.rootPath, t.prefix, t.expectedFilename))
		c.Assert(err, IsNil)

		info := st.Sys().(*syscall.Stat_t)
		c.Check(int(info.Nlink), Equals, 2)
	}
}
