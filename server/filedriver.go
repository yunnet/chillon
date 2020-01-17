package server


import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type FileDriver struct {
	RootPath string
	Perm
}

type FileInfoEx struct {
	os.FileInfo

	mode  os.FileMode
	owner string
	group string
}

func (f *FileInfoEx) Mode() os.FileMode {
	return f.mode
}

func (f *FileInfoEx) Owner() string {
	return f.owner
}

func (f *FileInfoEx) Group() string {
	return f.group
}

func (f *FileDriver) realPath(path string) string {
	paths := strings.Split(path, "/")
	return filepath.Join(append([]string{f.RootPath}, paths...)...)
}

func (f *FileDriver) Init(conn *Conn) {
	//driver.conn = conn
}

func (f *FileDriver) ChangeDir(path string) error {
	rPath := f.realPath(path)
	r, err := os.Lstat(rPath)
	if err != nil {
		return err
	}
	if r.IsDir() {
		return nil
	}
	return errors.New("Not a directory")
}

func (f *FileDriver) Stat(path string) (FileInfo, error) {
	basepath := f.realPath(path)
	rPath, err := filepath.Abs(basepath)
	if err != nil {
		return nil, err
	}

	r, err := os.Lstat(rPath)
	if err != nil {
		//fmt.Println("file not exists. " + rPath)
		return nil, err
	}

	if jsonStr, err := json.Marshal(r); err == nil{
		fmt.Println("ok file name: " + path)
		fmt.Println(string(jsonStr))
	}

	mode, err := f.Perm.GetMode(path)
	if err != nil {
		return nil, err
	}
	if r.IsDir() {
		mode |= os.ModeDir
	}
	owner, err := f.Perm.GetOwner(path)
	if err != nil {
		return nil, err
	}
	group, err := f.Perm.GetGroup(path)
	if err != nil {
		return nil, err
	}
	return &FileInfoEx{r, mode, owner, group}, nil
}

func (c *FileDriver) ListDir(path string, callback func(FileInfo) error) error {
	basepath := c.realPath(path)
	return filepath.Walk(basepath, func(f string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rPath, _ := filepath.Rel(basepath, f)
		if rPath == info.Name() {
			mode, err := c.Perm.GetMode(rPath)
			if err != nil {
				return err
			}
			if info.IsDir() {
				mode |= os.ModeDir
			}

			owner, err := c.Perm.GetOwner(rPath)
			if err != nil {
				return err
			}

			group, err := c.Perm.GetGroup(rPath)
			if err != nil {
				return err
			}

			err = callback(&FileInfoEx{info, mode, owner, group})
			if err != nil {
				return err
			}
			if info.IsDir() {
				return filepath.SkipDir
			}
		}
		return nil
	})
}

func (f *FileDriver) DeleteDir(path string) error {
	rPath := f.realPath(path)
	r, err := os.Lstat(rPath)
	if err != nil {
		return err
	}
	if r.IsDir() {
		return os.Remove(rPath)
	}
	return errors.New("Not a directory")
}

func (c *FileDriver) DeleteFile(path string) error {
	rPath := c.realPath(path)
	f, err := os.Lstat(rPath)
	if err != nil {
		return err
	}
	if !f.IsDir() {
		return os.Remove(rPath)
	}
	return errors.New("Not a file")
}

func (f *FileDriver) Rename(fromPath string, toPath string) error {
	oldPath := f.realPath(fromPath)
	newPath := f.realPath(toPath)
	return os.Rename(oldPath, newPath)
}

func (f *FileDriver) MakeDir(path string) error {
	rPath := f.realPath(path)
	return os.MkdirAll(rPath, os.ModePerm)
}

func (f *FileDriver) GetFile(path string, offset int64) (int64, io.ReadCloser, error) {
	rPath := f.realPath(path)
	r, err := os.Open(rPath)
	if err != nil {
		return 0, nil, err
	}

	info, err := r.Stat()
	if err != nil {
		return 0, nil, err
	}

	r.Seek(offset, io.SeekStart)

	return info.Size(), r, nil
}

func (f *FileDriver) PutFile(destPath string, data io.Reader, appendData bool) (int64, error) {
	rPath := f.realPath(destPath)

	var isExist bool
	r, err := os.Lstat(rPath)
	if err == nil {
		isExist = true
		if r.IsDir() {
			return 0, errors.New("A dir has the same name")
		}
	} else {
		if os.IsNotExist(err) {
			isExist = false
		} else {
			return 0, errors.New(fmt.Sprintln("Put File error:", err))
		}
	}

	if appendData && !isExist {
		appendData = false
	}

	if !appendData {
		if isExist {
			err = os.Remove(rPath)
			if err != nil {
				return 0, err
			}
		}
		r, err := os.Create(rPath)
		if err != nil {
			return 0, err
		}
		defer r.Close()
		bytes, err := io.Copy(r, data)
		if err != nil {
			return 0, err
		}
		return bytes, nil
	}

	of, err := os.OpenFile(rPath, os.O_APPEND|os.O_RDWR, 0660)
	if err != nil {
		return 0, err
	}
	defer of.Close()

	_, err = of.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, err
	}

	bytes, err := io.Copy(of, data)
	if err != nil {
		return 0, err
	}

	return bytes, nil
}

type FileDriverFactory struct {
	RootPath string
	Perm
}

func (f *FileDriverFactory) NewDriver() (Driver, error) {
	return &FileDriver{f.RootPath, f.Perm}, nil
}
