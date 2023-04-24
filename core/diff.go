package core

import (
	"bytes"
	"os"
	"os/exec"
)

// MergeDiff merges the diff lines between source and dest
func MergeDiff(sourceFile, destFile string) error {
	PrintVerbose("MergeDiff: merging %s -> %s", sourceFile, destFile)

	// get the diff lines
	diffLines, err := DiffFiles(sourceFile, destFile)
	if err != nil {
		PrintVerbose("MergeDiff:error: %s", err)
		return err
	}

	// write the diff to the destination
	err = WriteDiff(destFile, diffLines)
	if err != nil {
		PrintVerbose("MergeDiff:error: %s", err)
		return err
	}

	PrintVerbose("MergeDiff: merge completed")
	return nil
}

// DiffFiles returns the diff lines between source and dest files.
func DiffFiles(sourceFile, destFile string) ([]byte, error) {
	PrintVerbose("DiffFiles: diffing %s -> %s", sourceFile, destFile)

	cmd := exec.Command("diff", "-u", sourceFile, destFile)
	var out bytes.Buffer
	cmd.Stdout = &out
	errCode := 0
	err := cmd.Run()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			errCode = exitError.ExitCode()
		}
	}

	if errCode == 1 {
		PrintVerbose("DiffFiles: diff found")
		return out.Bytes(), nil
	}

	PrintVerbose("DiffFiles: no diff found")
	return nil, nil
}

// WriteDiff applies the diff lines to the destination file.
func WriteDiff(destFile string, diffLines []byte) error {
	PrintVerbose("WriteDiff: applying diff to %s", destFile)
	if len(diffLines) == 0 {
		PrintVerbose("WriteDiff: no changes to apply")
		return nil // no changes to apply
	}

	cmd := exec.Command("patch", destFile)
	cmd.Stdin = bytes.NewReader(diffLines)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		PrintVerbose("WriteDiff:error: %s", err)
		return err
	}

	PrintVerbose("WriteDiff: diff applied")
	return nil
}
