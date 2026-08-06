package netutil

import (
	"bytes"
	"compress/gzip"
	"crypto/sha1"
	"encoding/binary"
	"encoding/json"
	"fmt"
)

func GzipJSON[T any](v T, level int) ([]byte, error) {
	buf := bytes.Buffer{}
	gz, err := gzip.NewWriterLevel(&buf, level)
	if err != nil {
		return nil, err
	}
	if err := json.NewEncoder(gz).Encode(v); err != nil {
		gz.Close()
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Takes the items slice and applies keyFn to each element. The return values of all elements are
// written into a SHA-1 hash, which can in turn be used to identify whether items have changed.
//
// This hash is useful for purposes such as ETag creation, helping avoid unnecessary data transfers when the data has not changed.
func ComputeHash[T any](items []T, keyFn func(T) (id string, timestamp int64)) string {
	h := sha1.New()
	for _, item := range items {
		id, ts := keyFn(item)
		h.Write([]byte(id))
		var buf [8]byte
		binary.LittleEndian.PutUint64(buf[:], uint64(ts))
		h.Write(buf[:])
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}
