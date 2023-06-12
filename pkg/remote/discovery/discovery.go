package discovery

import (
	"log"

	"github.com/fsnotify/fsnotify"
)

type Discovery struct {
	watchDir   string
	dirWatcher *fsnotify.Watcher
}

func New(watchDir string) (*Discovery, error) {
	return &Discovery{
		watchDir: watchDir,
	}, nil
}

func (d *Discovery) Start() error {
	var err error
	d.dirWatcher, err = fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case event, ok := <-d.dirWatcher.Events:
				if !ok {
					return
				}
				log.Println("event:", event)
				if event.Has(fsnotify.Create) {
					log.Println("created file:", event.Name)
				}
				if event.Has(fsnotify.Remove) {
					log.Println("removed file:", event.Name)
				}

			case err, ok := <-d.dirWatcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	return d.dirWatcher.Add(d.watchDir)
}

func (d *Discovery) Close() error {
	return d.dirWatcher.Close()
}
