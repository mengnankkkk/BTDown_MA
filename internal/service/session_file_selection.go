package service

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/anacrolix/torrent"
)

type sessionFileDescriptor struct {
	file   *torrent.File
	name   string
	length int64
}

var preferredMediaExtensions = map[string]struct{}{
	".mp4":  {},
	".mkv":  {},
	".avi":  {},
	".mov":  {},
	".ts":   {},
	".m2ts": {},
	".wmv":  {},
	".flv":  {},
	".webm": {},
	".mpg":  {},
	".mpeg": {},
	".m4v":  {},
	".mp3":  {},
	".flac": {},
	".aac":  {},
	".wav":  {},
	".ogg":  {},
}

func selectPrimarySessionFile(files []*torrent.File) *sessionFileDescriptor {
	descriptors := make([]sessionFileDescriptor, 0, len(files))
	for _, file := range files {
		if file == nil {
			continue
		}
		descriptors = append(descriptors, sessionFileDescriptor{
			file:   file,
			name:   file.DisplayPath(),
			length: file.Length(),
		})
	}

	return choosePrimaryFileDescriptor(descriptors)
}

func choosePrimaryFileDescriptor(descriptors []sessionFileDescriptor) *sessionFileDescriptor {
	if len(descriptors) == 0 {
		return nil
	}

	candidates := make([]sessionFileDescriptor, 0, len(descriptors))
	for _, descriptor := range descriptors {
		if _, ok := preferredMediaExtensions[strings.ToLower(filepath.Ext(descriptor.name))]; ok {
			candidates = append(candidates, descriptor)
		}
	}
	if len(candidates) == 0 {
		candidates = append(candidates, descriptors...)
	}

	slices.SortFunc(candidates, func(left, right sessionFileDescriptor) int {
		switch {
		case left.length > right.length:
			return -1
		case left.length < right.length:
			return 1
		default:
			return strings.Compare(left.name, right.name)
		}
	})

	selected := candidates[0]
	return &selected
}
