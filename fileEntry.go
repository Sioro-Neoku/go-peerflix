package main

import (
	"io"
	"os"

	"github.com/anacrolix/torrent"
)

// SeekableContent describes an io.ReadSeeker that can be closed as well.
type SeekableContent interface {
	io.ReadSeeker
	io.Closer
}

// FileEntry helps reading a torrent file.
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

func NewFileReader(f torrent.File) SeekableContent {
	reader := t.NewReader()
	reader.SetReadahead(f.Length() / 100)
	reader.SetResponsive()
	reader.Seek(f.Offset(), os.SEEK_SET)

	return &FileEntry{
		File:   &f,
		Reader: reader,
	}
}
