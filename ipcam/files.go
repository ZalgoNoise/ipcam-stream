package ipcam

import (
	"io/fs"
	"os"
	"time"

	"github.com/zalgonoise/zlog/log"
)

type cache struct {
	root  string
	files []string
}

func (c *cache) load(path string) error {

	var err error

	c.files, err = fs.Glob(os.DirFS(path), "*")

	if err != nil {
		logCh <- log.NewMessage().Level(log.LLWarn).Sub("load()").Message("failed to parse cache's glob path").Metadata(log.Field{"path": path, "error": err.Error()}).Build()

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
			logCh <- log.NewMessage().Level(log.LLWarn).Sub("clear()").Message("failed to remove target file").Metadata(log.Field{"path": c.root + file, "error": err.Error()}).Build()

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
		logCh <- log.NewMessage().Level(log.LLWarn).Sub("load()").Message("failed to parse glob path").Metadata(log.Field{"path": path, "error": err.Error()}).Build()

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
			logCh <- log.NewMessage().Level(log.LLWarn).Sub("listOlder()").Message("failed to parse folder name").Metadata(log.Field{"target": folder, "error": err.Error()}).Build()

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
			logCh <- log.NewMessage().Level(log.LLWarn).Sub("rotate()").Message("failed to remove old stream files").Metadata(log.Field{"target": d.root + target, "error": err.Error()}).Build()
		}
	}

	if len(errs) > 0 {
		for _, err := range errs {
			logCh <- log.NewMessage().Level(log.LLWarn).Sub("rotate()").Message("failed to list stream files").Metadata(log.Field{"error": err.Error()}).Build()
		}
	}
}
