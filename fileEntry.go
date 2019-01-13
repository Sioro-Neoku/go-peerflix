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
	*torrent.File
	torrent.Reader
}

// Seek seeks to the correct file position, paying attention to the offset.
func (f FileEntry) Seek(offset int64, whence int) (int64, error) {
	return f.Reader.Seek(offset+f.File.Offset(), whence)
}

// NewFileReader sets up a torrent file for streaming reading.
func NewFileReader(f *torrent.File) (SeekableContent, error) {
	torrent := f.Torrent()
	reader := torrent.NewReader()

	// We read ahead 1% of the file continuously.
	reader.SetReadahead(f.Length() / 100)
	reader.SetResponsive()
	_, err := reader.Seek(f.Offset(), os.SEEK_SET)

	return &FileEntry{
		File:   f,
		Reader: reader,
	}, err
}
