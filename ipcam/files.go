package ipcam

import (
	"io/fs"
	"os"
)

type cache struct {
	root  string
	files []string
}

func (c *cache) load(path string) error {

	var err error

	c.files, err = fs.Glob(os.DirFS(path), "*")

	if err != nil {
		return err
	}

	c.root = path

	return nil

}

func (c *cache) clear() []error {
	if len(c.files) <= 0 {
		return nil
	}

	var errs []error

	for _, file := range c.files {
		if err := os.RemoveAll(c.root + file); err != nil {
			errs = append(errs, err)
		}
	}

	return errs

}
