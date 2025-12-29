package storage

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type fakeS3 struct {
	lastPut  *s3.PutObjectInput
	lastCopy *s3.CopyObjectInput
}

func (f *fakeS3) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	f.lastPut = params
	if params.Body != nil {
		_, _ = io.ReadAll(params.Body)
	}
	return &s3.PutObjectOutput{}, nil
}

func (f *fakeS3) CopyObject(ctx context.Context, params *s3.CopyObjectInput, optFns ...func(*s3.Options)) (*s3.CopyObjectOutput, error) {
	f.lastCopy = params
	return &s3.CopyObjectOutput{}, nil
}

func TestKeyConstruction(t *testing.T) {
	u := NewWithClient("bucket", "yodex", &fakeS3{})
	date := time.Date(2025, 9, 30, 0, 0, 0, 0, time.UTC)
	if got := u.KeyForDate(date, "episode.mp3"); got != "yodex/2025/09/30/episode.mp3" {
		t.Fatalf("KeyForDate mismatch: %s", got)
	}
	if got := u.KeyForLatest("episode.mp3"); got != "yodex/latest/episode.mp3" {
		t.Fatalf("KeyForLatest mismatch: %s", got)
	}
}

func TestUploadAndCopy(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "episode.mp3")
	if err := os.WriteFile(path, []byte("audio"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	fake := &fakeS3{}
	u := NewWithClient("bucket", "yodex", fake)
	ctx := context.Background()

	key := u.KeyForDate(time.Date(2025, 9, 30, 0, 0, 0, 0, time.UTC), "episode.mp3")
	if err := u.UploadFile(ctx, key, path, "audio/mpeg", "public, max-age=86400"); err != nil {
		t.Fatalf("UploadFile error: %v", err)
	}
	if fake.lastPut == nil || fake.lastPut.Key == nil || *fake.lastPut.Key != key {
		t.Fatalf("expected PutObject with key %q", key)
	}

	if err := u.CopyToLatest(ctx, key, "episode.mp3", "audio/mpeg", "public, max-age=300"); err != nil {
		t.Fatalf("CopyToLatest error: %v", err)
	}
	if fake.lastCopy == nil || fake.lastCopy.Key == nil || *fake.lastCopy.Key != "yodex/latest/episode.mp3" {
		t.Fatalf("expected CopyObject to latest key")
	}
}
