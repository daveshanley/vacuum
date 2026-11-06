package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

// writeRepairFile publishes a complete repaired file after checking the source.
// Rename is atomic per file; the source checks are not a filesystem CAS.
func writeRepairFile(source, output string, expected, repaired []byte) error {
	return writeRepairFileMode(source, output, expected, repaired, false)
}

// writeNewRepairFile creates a separate file without replacing an existing path,
// including a destination created after review or while the bytes are staged.
func writeNewRepairFile(source, output string, expected, repaired []byte) error {
	return writeRepairFileMode(source, output, expected, repaired, true)
}

func writeRepairFileMode(source, output string, expected, repaired []byte, createOnly bool) error {
	source, err := filepath.Abs(source)
	if err != nil {
		return err
	}
	if output == "" {
		output = source
	} else if output, err = filepath.Abs(output); err != nil {
		return err
	}
	if createOnly && output == source {
		return fmt.Errorf("new file must have a different path from the source")
	}
	info, err := os.Lstat(source)
	if err != nil {
		return fmt.Errorf("inspect repair source: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("repair source must be a regular file: %s", source)
	}
	original, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	if expected != nil && !bytes.Equal(original, expected) {
		return fmt.Errorf("repair source changed since lint: %s", source)
	}
	if target, e := os.Lstat(output); e == nil {
		if createOnly {
			return fmt.Errorf("new file already exists: %s", output)
		}
		if !target.Mode().IsRegular() {
			return fmt.Errorf("repair output must be a regular file: %s", output)
		}
		if output != source && os.SameFile(info, target) {
			return fmt.Errorf("repair output aliases source: %s", output)
		}
	} else if !os.IsNotExist(e) {
		return e
	}
	staged, err := os.CreateTemp(filepath.Dir(output), ".vacuum-repair-*")
	if err != nil {
		return err
	}
	defer os.Remove(staged.Name())
	defer staged.Close()
	if err = staged.Chmod(info.Mode().Perm()); err != nil {
		return err
	}
	if _, err = staged.Write(repaired); err != nil {
		return err
	}
	if err = staged.Sync(); err != nil {
		return err
	}
	if err = staged.Close(); err != nil {
		return err
	}
	if output == source {
		backup, e := os.CreateTemp("", "vacuum-"+filepath.Base(source)+"-*.bak")
		if e != nil {
			return e
		}
		_, writeErr := backup.Write(original)
		closeErr := backup.Close()
		if writeErr != nil {
			return writeErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	current, err := os.Lstat(source)
	if err != nil {
		return err
	}
	if !current.Mode().IsRegular() || !os.SameFile(info, current) {
		return fmt.Errorf("repair source changed since lint: %s", source)
	}
	latest, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	if !bytes.Equal(latest, original) {
		return fmt.Errorf("repair source changed since lint: %s", source)
	}
	if err := publishRepairFile(staged.Name(), output, createOnly); err != nil {
		return fmt.Errorf("publish repaired file: %w", err)
	}
	return nil
}

// Both publication paths expose complete bytes. Linking the staged file is an
// atomic create-only operation on its destination filesystem; rename may replace.
func publishRepairFile(staged, output string, createOnly bool) error {
	if createOnly {
		return os.Link(staged, output)
	}
	return os.Rename(staged, output)
}
