package vfile

import "fmt"

type InvalidFilePath struct {
	Mountpoint string
	Path       string
}

func (e InvalidFilePath) Error() string {
	return fmt.Sprintf("virtual file mountpoint: %s invalid path: %s", e.Mountpoint, e.Path)
}
