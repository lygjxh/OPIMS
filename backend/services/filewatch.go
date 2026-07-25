package services

import (
	"io/fs"
	"log"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
)

type FileWatcher struct {
	rootPath string
	watcher  *fsnotify.Watcher
	mu       sync.RWMutex
	onChange func(string)
}

func NewFileWatcher(rootPath string) *FileWatcher {
	return &FileWatcher{rootPath: rootPath}
}

func (fw *FileWatcher) SetRoot(path string) {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	fw.rootPath = path
}

func (fw *FileWatcher) Root() string {
	fw.mu.RLock()
	defer fw.mu.RUnlock()
	return fw.rootPath
}

func (fw *FileWatcher) OnChange(fn func(string)) {
	fw.onChange = fn
}

func (fw *FileWatcher) Start() {
	var err error
	fw.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		log.Println("FileWatcher error:", err)
		return
	}

	go func() {
		for {
			select {
			case event, ok := <-fw.watcher.Events:
				if !ok {
					return
				}
				if fw.onChange != nil {
					fw.onChange(event.Name)
				}
			case err, ok := <-fw.watcher.Errors:
				if !ok {
					return
				}
				log.Println("FileWatcher error:", err)
			}
		}
	}()

	// watch root recursively
	fw.addRecursive(fw.rootPath)
	log.Println("FileWatcher started for", fw.rootPath)
}

func (fw *FileWatcher) addRecursive(dir string) {
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			fw.watcher.Add(path)
		}
		return nil
	})
}

func (fw *FileWatcher) Stop() {
	if fw.watcher != nil {
		fw.watcher.Close()
	}
}
