package update

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
)

func extractBinary(ctx context.Context, archivePath, staged, binary string, zipped bool) error {
	if zipped {
		return extractZip(ctx, archivePath, staged, binary)
	}
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gz.Close()
	expanded := &io.LimitedReader{R: contextReader{ctx, gz}, N: maxBinarySize + maxMetadataSize + 1}
	reader := tar.NewReader(expanded)
	found := false
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if header.Name != binary && header.Name != "./"+binary {
			return fmt.Errorf("unexpected archive entry %q", header.Name)
		}
		if found || header.Typeflag != tar.TypeReg || header.Size <= 0 || header.Size > maxBinarySize {
			return fmt.Errorf("archive must contain exactly one regular %s executable", binary)
		}
		if err := writeBinary(ctx, staged, reader); err != nil {
			return err
		}
		found = true
	}
	if !found {
		return fmt.Errorf("archive does not contain %s", binary)
	}
	// Read the gzip trailer so a corrupt stream cannot be installed.
	_, err = io.Copy(io.Discard, expanded)
	if expanded.N == 0 {
		return fmt.Errorf("expanded release archive is too large")
	}
	return err
}

func extractZip(ctx context.Context, archivePath, staged, binary string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer reader.Close()
	if len(reader.File) != 1 {
		return fmt.Errorf("archive must contain exactly one regular %s executable", binary)
	}
	file := reader.File[0]
	if (file.Name != binary && file.Name != "./"+binary) || !file.Mode().IsRegular() || file.UncompressedSize64 == 0 || file.UncompressedSize64 > maxBinarySize {
		return fmt.Errorf("archive must contain exactly one regular %s executable", binary)
	}
	content, err := file.Open()
	if err != nil {
		return err
	}
	defer content.Close()
	return writeBinary(ctx, staged, content)
}

func writeBinary(ctx context.Context, path string, reader io.Reader) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0700)
	if err != nil {
		return err
	}
	defer file.Close()
	n, err := io.Copy(file, io.LimitReader(contextReader{ctx, reader}, maxBinarySize+1))
	if err != nil {
		return err
	}
	if n <= 0 || n > maxBinarySize {
		return fmt.Errorf("executable size must be between 1 and %d bytes", maxBinarySize)
	}
	if err := file.Sync(); err != nil {
		return err
	}
	return file.Close()
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}
