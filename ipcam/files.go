package ipcam

import (
	"fmt"
	"io/fs"
	"os"
	"time"
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

type dir struct {
	root string
	dirs []string
}

func (d *dir) load(path string) error {

	var err error

	d.dirs, err = fs.Glob(os.DirFS(path), "*")

	if err != nil {
		return err
	}

	d.root = path

	return nil

}

func (d *dir) exists(name string) bool {
	for _, dir := range d.dirs {
		if name == dir {
			return true
		}
	}
	return false
}

func (d *dir) mkdir(name string) error {
	return os.Mkdir(d.root+name, 0755)
}

func (d *dir) listOlder(from time.Time, days int) ([]string, []error) {
	var older []string
	var errs []error

	day := time.Duration(24 * time.Hour)

	tresh := from.Add(time.Duration(-days) * day)

	for _, folder := range d.dirs {
		if folder == "cache" {
			continue
		}

		t, err := time.Parse("2006-01-02", folder)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if t.Before(tresh) {
			older = append(older, folder)
		}

	}
	return older, errs
}

func (d *dir) rotate(from time.Time, days int) {
	targets, errs := d.listOlder(from, days)

	for _, target := range targets {
		if err := os.RemoveAll(d.root + target); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		for _, err := range errs {
			fmt.Printf("[ipcam-stream] [rotate] ERR: %s\n", err)
		}
	}
}
