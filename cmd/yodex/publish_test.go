package main

import (
	"context"
	"os"
	"testing"
	"time"

	"yodex/internal/paths"
)

type fakeUploader struct {
	uploads []string
	copies  []string
}

func (f *fakeUploader) UploadFile(ctx context.Context, key, localPath, contentType, cacheControl string) error {
	f.uploads = append(f.uploads, key)
	return nil
}

func (f *fakeUploader) CopyToLatest(ctx context.Context, srcKey, filename, contentType, cacheControl string) error {
	f.copies = append(f.copies, filename)
	return nil
}

func (f *fakeUploader) KeyForDate(t time.Time, filename string) string {
	return "prefix/" + filename
}

func TestPublishUploadsMP3Only(t *testing.T) {
	orig := newUploader
	t.Cleanup(func() { newUploader = orig })

	fake := &fakeUploader{}
	newUploader = func(ctx context.Context, bucket, prefix, region string) (uploader, error) {
		return fake, nil
	}

	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWD) })

	date := time.Date(2025, 9, 30, 0, 0, 0, 0, time.UTC)
	builder := paths.New("")
	if err := builder.EnsureOutDir(date); err != nil {
		t.Fatalf("EnsureOutDir: %v", err)
	}
	if err := os.WriteFile(builder.EpisodeMP3(date), []byte("audio"), 0o644); err != nil {
		t.Fatalf("write mp3: %v", err)
	}

	if code := run([]string{"publish", "--date=2025-09-30", "--bucket=b", "--region=us-west-2"}); code != 0 {
		t.Fatalf("publish returned non-zero: %d", code)
	}
	if len(fake.uploads) != 1 {
		t.Fatalf("expected 1 upload, got %d", len(fake.uploads))
	}
	if len(fake.copies) != 1 {
		t.Fatalf("expected 1 copy, got %d", len(fake.copies))
	}
}

func TestPublishUploadsScriptAndMeta(t *testing.T) {
	orig := newUploader
	t.Cleanup(func() { newUploader = orig })

	fake := &fakeUploader{}
	newUploader = func(ctx context.Context, bucket, prefix, region string) (uploader, error) {
		return fake, nil
	}

	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWD) })

	date := time.Date(2025, 9, 30, 0, 0, 0, 0, time.UTC)
	builder := paths.New("")
	if err := builder.EnsureOutDir(date); err != nil {
		t.Fatalf("EnsureOutDir: %v", err)
	}
	if err := os.WriteFile(builder.EpisodeMP3(date), []byte("audio"), 0o644); err != nil {
		t.Fatalf("write mp3: %v", err)
	}
	if err := os.WriteFile(builder.EpisodeMarkdown(date), []byte("script"), 0o644); err != nil {
		t.Fatalf("write md: %v", err)
	}
	if err := os.WriteFile(builder.EpisodeMeta(date), []byte(`{"date":"2025-09-30"}`), 0o644); err != nil {
		t.Fatalf("write meta: %v", err)
	}

	if code := run([]string{"publish", "--date=2025-09-30", "--bucket=b", "--region=us-west-2", "--include-script"}); code != 0 {
		t.Fatalf("publish returned non-zero: %d", code)
	}
	if len(fake.uploads) != 3 {
		t.Fatalf("expected 3 uploads, got %d", len(fake.uploads))
	}
	if len(fake.copies) != 3 {
		t.Fatalf("expected 3 copies, got %d", len(fake.copies))
	}
}
