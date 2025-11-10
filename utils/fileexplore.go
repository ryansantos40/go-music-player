package utils

import (
	"os"
	"path/filepath"
	"sort"
)

type DirEntry struct {
	Name  string
	Path  string
	IsDir bool
}

type FileExplorer struct {
	CurrentPath string
	Entries     []DirEntry
	Error       error
}

func NewFileExplorer(startPath string) *FileExplorer {
	if startPath == "" {
		startPath, _ = os.Getwd()
	}

	fe := &FileExplorer{
		CurrentPath: startPath,
	}

	fe.LoadEntries()
	return fe
}

func (fe *FileExplorer) LoadEntries() {
	entries, err := os.ReadDir(fe.CurrentPath)
	if err != nil {
		fe.Error = err
		return
	}

	fe.Entries = []DirEntry{}
	fe.Error = nil

	parent := filepath.Dir(fe.CurrentPath)
	if parent != fe.CurrentPath {
		fe.Entries = append(fe.Entries, DirEntry{
			Name:  "..",
			Path:  parent,
			IsDir: true,
		})
	}

	for _, entry := range entries {
		if entry.IsDir() {
			fullPath := filepath.Join(fe.CurrentPath, entry.Name())
			fe.Entries = append(fe.Entries, DirEntry{
				Name:  entry.Name(),
				Path:  fullPath,
				IsDir: true,
			})
		}
	}

	sort.Slice(fe.Entries, func(i, j int) bool {
		if fe.Entries[i].Name == ".." {
			return true
		}

		if fe.Entries[j].Name == ".." {
			return false
		}

		return fe.Entries[i].Name < fe.Entries[j].Name
	})
}

func (fe *FileExplorer) EnterDirectory(index int) bool {
	if index < 0 || index >= len(fe.Entries) {
		return false
	}

	entry := fe.Entries[index]
	if !entry.IsDir {
		return false
	}

	fe.CurrentPath = entry.Path
	fe.LoadEntries()
	return true
}

func (fe *FileExplorer) GoToParent() bool {
	parent := filepath.Dir(fe.CurrentPath)
	if parent == fe.CurrentPath {
		return false
	}

	fe.CurrentPath = parent
	fe.LoadEntries()
	return true
}

func (fe *FileExplorer) GetCurrentPath() string {
	return fe.CurrentPath
}
