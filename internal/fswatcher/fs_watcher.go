package fswatcher

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
)

// Watch calls 'run' for every file update inside one of 'rootDirectories' and the file matches 'filePattern'.
// Returns when 'ctx' is canceled.
func Watch(ctx context.Context, rootDirectories []string, filePattern string, run func()) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	for _, root := range rootDirectories {
		err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if err := watcher.Add(path); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	timer := time.NewTimer(0)
	timer.Stop() // avoid firing until update event
	defer timer.Stop()

	const debounce = 2 * time.Second
	for {
		select {
		case <-timer.C:
			run()
		case <-ctx.Done():
			return ctx.Err()
		case event := <-watcher.Events:
			matched, err := filepath.Match(filePattern, filepath.Base(event.Name))
			if err == nil && matched {
				if event.Op&(fsnotify.Remove|fsnotify.Create) != 0 {
					info, err := os.Stat(event.Name)
					isDir := err == nil && info.IsDir()
					switch {
					case isDir && event.Op&fsnotify.Remove != 0:
						_ = watcher.Remove(event.Name)
					case isDir && event.Op&fsnotify.Create != 0:
						_ = watcher.Add(event.Name)
					case !isDir && event.Op&fsnotify.Create != 0:
						_ = watcher.Add(filepath.Dir(event.Name))
					}
				}
				if event.Op&(fsnotify.Write|fsnotify.Remove|fsnotify.Create|fsnotify.Rename) != 0 {
					timer.Reset(debounce)
				}
			}
		case err := <-watcher.Errors:
			var pathErr *os.PathError
			if errors.As(err, &pathErr) && errors.Is(err, os.ErrNotExist) {
				_ = watcher.Remove(pathErr.Path)
			} else {
				fmt.Fprintln(os.Stderr, "Watch error:", err)
			}
		}
	}
}

type restarter struct {
	Context context.Context
	Cancel  context.CancelCauseFunc
}

// WatchRestarter is like [Watch], but automatically restarts 'run' on watch events.
// 'run' is called, its context is canceled when a watch event occurs, then repeats until 'ctx' is canceled.
func WatchRestarter(ctx context.Context, rootDirectories []string, filePattern string, run func(context.Context) error) error {
	var runRestarter atomic.Pointer[restarter]
	runRestarter.Store(&restarter{Context: nil, Cancel: func(error) {}})
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				runCtx, runCancel := context.WithCancelCause(ctx)
				runRestarter.Store(&restarter{
					Context: runCtx,
					Cancel:  runCancel,
				})
				err := run(runCtx)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
				}
				const commandFailedWait = 10 * time.Second
				select {
				case <-ctx.Done():
				case <-time.After(commandFailedWait):
				}
			}
		}
	}()
	return Watch(ctx, rootDirectories, filePattern, func() {
		runRestarter.Load().Cancel(errors.Errorf("files updated matching pattern: %s", filePattern))
	})
}
