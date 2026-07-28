package importing

import "testing"

func TestNewImportBatchResponseIncludesFileMetadata(t *testing.T) {
	batch := ImportBatch{
		ID:               15,
		Code:             "IMPORT-20260728-8e51fa9a1c30",
		Profile:          ProfileOwnerOutlet,
		OriginalFilename: "sample.xlsx",
		FileSHA256:       "8e51fa9a1c30",
		Status:           BatchStatusValidated,
	}

	resp := NewImportBatchResponse(batch)

	if resp.File.OriginalFilename != "sample.xlsx" {
		t.Fatalf("original filename = %q", resp.File.OriginalFilename)
	}
	if resp.File.SHA256 != "8e51fa9a1c30" {
		t.Fatalf("sha256 = %q", resp.File.SHA256)
	}
	if resp.File.ViewPath != "/api/v1/imports/15/file" {
		t.Fatalf("view_path = %q", resp.File.ViewPath)
	}
	if resp.File.DownloadPath != "/api/v1/imports/15/file/download" {
		t.Fatalf("download_path = %q", resp.File.DownloadPath)
	}
}
