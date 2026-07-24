package services

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"

	"novelhub/pkg/jsonx"
)

func TestUploadServiceRecoversSessionsAndCleansInvalidOnes(t *testing.T) {
	root := filepath.Join(t.TempDir(), "uploads")
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}

	validID := uuid.NewString()
	writeUploadTestSession(t, root, validID, uploadManifest{
		OwnerID: "owner", TotalBytes: 6, TotalChunks: 2, ExpiresAt: time.Now().Add(time.Hour),
		UploadChunkBytes: 4, UploadChunks: 2, UploadSessions: 3, UploadBytes: 10, SessionTTLSeconds: 60,
	}, map[string]string{"chunk_0": "abcd", "chunk_1": "ef"})
	expiredID := uuid.NewString()
	writeUploadTestSession(t, root, expiredID, uploadManifest{
		OwnerID: "owner", TotalBytes: 1, TotalChunks: 1, ExpiresAt: time.Now().Add(-time.Minute),
		UploadChunkBytes: 1, UploadChunks: 1, UploadSessions: 3, UploadBytes: 1, SessionTTLSeconds: 60,
	}, nil)
	invalidID := uuid.NewString()
	if err := os.Mkdir(filepath.Join(root, invalidID), 0700); err != nil {
		t.Fatal(err)
	}

	s := &uploadService{root: root, sessions: map[string]*uploadSession{}, ownerCounts: map[string]int{}}
	s.recoverSessions()

	session := s.sessions[validID]
	if session == nil || session.storedBytes != 6 || session.storedChunks != 2 || s.ownerCounts["owner"] != 1 {
		t.Fatalf("unexpected recovered state: session=%+v owner count=%d", session, s.ownerCounts["owner"])
	}
	for _, id := range []string{expiredID, invalidID} {
		if _, err := os.Stat(filepath.Join(root, id)); !os.IsNotExist(err) {
			t.Fatalf("session %s was not cleaned up: %v", id, err)
		}
	}
}

func writeUploadTestSession(t *testing.T, root, id string, manifest uploadManifest, chunks map[string]string) {
	t.Helper()
	dir := filepath.Join(root, id)
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	data, err := jsonx.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "session.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	for name, contents := range chunks {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(contents), 0600); err != nil {
			t.Fatal(err)
		}
	}
}
