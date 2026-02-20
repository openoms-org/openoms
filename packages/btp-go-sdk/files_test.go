package btp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFileAdd(t *testing.T) {
	srv := httptest.NewServer(withCSRF(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Gateway/ClientApi/FileAdd" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		ct := r.Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "multipart/form-data") {
			t.Fatalf("expected multipart/form-data, got %s", ct)
		}

		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("failed to parse multipart: %v", err)
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("expected file field: %v", err)
		}
		defer file.Close()
		if header.Filename != "test.pdf" {
			t.Fatalf("expected filename test.pdf, got %s", header.Filename)
		}
		data, _ := io.ReadAll(file)
		if string(data) != "pdf-content" {
			t.Fatalf("unexpected file content: %s", string(data))
		}
		if desc := r.FormValue("description"); desc != "Test file" {
			t.Fatalf("expected description 'Test file', got %s", desc)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(FileAddResponse{
			Result: APIResult{ResultType: "INFORMATION"},
			FileID: "file-abc-123",
		})
	}))
	defer srv.Close()

	c := NewClient("u", "p", WithBaseURL(srv.URL))
	resp, err := c.Files.Add(context.Background(), "test.pdf", strings.NewReader("pdf-content"), "Test file")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.FileID != "file-abc-123" {
		t.Fatalf("expected file-abc-123, got %s", resp.FileID)
	}
}

func TestFilesDelete(t *testing.T) {
	srv := httptest.NewServer(withCSRF(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Gateway/ClientApi/FilesDelete" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Fatalf("expected DELETE, got %s", r.Method)
		}

		var req FilesDeleteRequest
		json.NewDecoder(r.Body).Decode(&req)
		if len(req.FileIDs) != 2 {
			t.Fatalf("expected 2 file IDs, got %d", len(req.FileIDs))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(APIResult{ResultType: "INFORMATION"})
	}))
	defer srv.Close()

	c := NewClient("u", "p", WithBaseURL(srv.URL))
	result, err := c.Files.Delete(context.Background(), []string{"f1", "f2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ResultType != "INFORMATION" {
		t.Fatalf("expected INFORMATION, got %s", result.ResultType)
	}
}

func TestFilesClear(t *testing.T) {
	srv := httptest.NewServer(withCSRF(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Gateway/ClientApi/FilesClear" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Fatalf("expected DELETE, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(APIResult{ResultType: "INFORMATION"})
	}))
	defer srv.Close()

	c := NewClient("u", "p", WithBaseURL(srv.URL))
	result, err := c.Files.Clear(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ResultType != "INFORMATION" {
		t.Fatalf("expected INFORMATION, got %s", result.ResultType)
	}
}
