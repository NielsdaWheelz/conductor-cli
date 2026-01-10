package fs

import (
	"os"
	"path/filepath"
)

// WriteFileAtomic writes data to path atomically using a temp file + rename.
// The temp file is created in the same directory as path to ensure atomic rename on POSIX.
// If the operation fails, the original file (if any) is left unchanged.
// The caller must ensure the parent directory exists.
func WriteFileAtomic(fs FS, path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	pattern := ".agency-tmp-*"

	// Create temp file in the same directory
	tmpPath, w, err := fs.CreateTemp(dir, pattern)
	if err != nil {
		return err
	}

	// Ensure cleanup on any error path
	success := false
	defer func() {
		if !success {
			fs.Remove(tmpPath)
		}
	}()

	// Write data to temp file
	_, err = w.Write(data)
	if err != nil {
		w.Close()
		return err
	}

	// Close the file before rename
	if err := w.Close(); err != nil {
		return err
	}

	// Set permissions on temp file before rename
	if err := fs.Chmod(tmpPath, perm); err != nil {
		return err
	}

	// Atomic rename
	if err := fs.Rename(tmpPath, path); err != nil {
		return err
	}

	success = true
	return nil
}
