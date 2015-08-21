package main

import (
	"io"

	"github.com/anacrolix/torrent"
)

type SeekableContent interface {
	io.ReadSeeker
	io.Closer
}

type FileEntry struct {
	File   *torrent.File
	Reader *torrent.Reader
}

func (f FileEntry) Read(p []byte) (n int, err error) {
	return f.Reader.Read(p)
}

func (f FileEntry) Seek(offset int64, whence int) (int64, error) {
	return f.Reader.Seek(offset+f.File.Offset(), whence)
}

func (f FileEntry) Close() error {
	return f.Reader.Close()
}
