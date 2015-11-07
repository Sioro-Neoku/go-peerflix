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
	File *torrent.File
	*torrent.Reader
}

// Seek seeks to the correct file position, paying attention to the offset.
func (f FileEntry) Seek(offset int64, whence int) (int64, error) {
	return f.Reader.Seek(offset+f.File.Offset(), whence)
}

// NewFileReader sets up a torrent file for streaming reading.
func NewFileReader(c Client, f *torrent.File) (SeekableContent, error) {
	// We read ahead 1% of the file continuously.
	var readahead = f.Length() / 100

	reader := c.Torrent.NewReader()
	reader.SetReadahead(readahead)
	reader.SetResponsive()
	_, err := reader.Seek(f.Offset(), os.SEEK_SET)

	return &FileEntry{
		File:   f,
		Reader: reader,
	}, err
}
