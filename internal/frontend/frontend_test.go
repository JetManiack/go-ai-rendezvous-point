package frontend_test

import (
	"io"
	"strings"
	"testing"

	"go-ai-rendezvous-point/internal/frontend"
)

func TestFS_ServesIndexHTML(t *testing.T) {
	fsys, err := frontend.FS(false)
	if err != nil {
		t.Fatalf("FS(false) error = %v", err)
	}

	f, err := fsys.Open("index.html")
	if err != nil {
		t.Fatalf("Open(index.html) error = %v", err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if !strings.Contains(string(data), `id="root"`) {
		t.Error("index.html does not contain the React mount point (id=\"root\")")
	}
	if !strings.Contains(string(data), `src="/js/app.bundle.js"`) {
		t.Error("index.html does not link /js/app.bundle.js — the page would render blank")
	}
	if !strings.Contains(string(data), "--bg:") {
		t.Error("index.html does not define the design system's CSS custom properties (--bg) — the stylesheet appears to be missing")
	}
}

func TestFS_ServesGeneratedBundle(t *testing.T) {
	fsys, err := frontend.FS(false)
	if err != nil {
		t.Fatalf("FS(false) error = %v", err)
	}

	f, err := fsys.Open("js/app.bundle.js")
	if err != nil {
		t.Fatalf("Open(js/app.bundle.js) error = %v — did you run `go generate ./internal/frontend/...`?", err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if len(data) == 0 {
		t.Error("js/app.bundle.js is empty")
	}
	if !strings.Contains(string(data), "/api/agents") {
		t.Error("app.bundle.js does not reference /api/agents — Agents.jsx was likely not actually bundled (check that index.jsx imports it)")
	}
}

func TestFS_BundleIncludesRouterAndNav(t *testing.T) {
	fsys, err := frontend.FS(false)
	if err != nil {
		t.Fatalf("FS(false) error = %v", err)
	}

	f, err := fsys.Open("js/app.bundle.js")
	if err != nil {
		t.Fatalf("Open(js/app.bundle.js) error = %v", err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if !strings.Contains(string(data), "hashchange") {
		t.Error("app.bundle.js does not reference hashchange — router.js was likely not bundled (check that App.jsx imports it)")
	}
	if !strings.Contains(string(data), "nav-tabs") {
		t.Error("app.bundle.js does not reference nav-tabs — TopNav.jsx was likely not bundled (check that Agents.jsx imports it)")
	}
}

func TestFS_BundleIncludesThreadList(t *testing.T) {
	fsys, err := frontend.FS(false)
	if err != nil {
		t.Fatalf("FS(false) error = %v", err)
	}

	f, err := fsys.Open("js/app.bundle.js")
	if err != nil {
		t.Fatalf("Open(js/app.bundle.js) error = %v", err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if !strings.Contains(string(data), "/api/threads") {
		t.Error("app.bundle.js does not reference /api/threads — ThreadList.jsx was likely not bundled")
	}
	if !strings.Contains(string(data), "Load more") {
		t.Error("app.bundle.js does not reference \"Load more\" — ThreadList.jsx's pagination control is missing from the bundle")
	}
}

func TestFS_BundleIncludesThreadDetail(t *testing.T) {
	fsys, err := frontend.FS(false)
	if err != nil {
		t.Fatalf("FS(false) error = %v", err)
	}

	f, err := fsys.Open("js/app.bundle.js")
	if err != nil {
		t.Fatalf("Open(js/app.bundle.js) error = %v", err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if !strings.Contains(string(data), "/api/actors") {
		t.Error("app.bundle.js does not reference /api/actors — ThreadDetail.jsx was likely not bundled")
	}
	if !strings.Contains(string(data), "Mark resolved") {
		t.Error("app.bundle.js does not reference \"Mark resolved\" — ThreadDetail.jsx's resolve/reopen control is missing from the bundle")
	}
}

func TestFS_BundleIncludesClickableTagFilter(t *testing.T) {
	fsys, err := frontend.FS(false)
	if err != nil {
		t.Fatalf("FS(false) error = %v", err)
	}

	f, err := fsys.Open("js/app.bundle.js")
	if err != nil {
		t.Fatalf("Open(js/app.bundle.js) error = %v", err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if !strings.Contains(string(data), "#/threads?tags=") {
		t.Error("app.bundle.js does not reference \"#/threads?tags=\" — ThreadDetail.jsx's clickable tag chips are missing from the bundle")
	}
	if !strings.Contains(string(data), "Filtered by tag") {
		t.Error("app.bundle.js does not reference \"Filtered by tag\" — ThreadList.jsx's active-tag-filter indicator is missing from the bundle")
	}
}

func TestFS_BundleIncludesMarkdownRendering(t *testing.T) {
	fsys, err := frontend.FS(false)
	if err != nil {
		t.Fatalf("FS(false) error = %v", err)
	}

	f, err := fsys.Open("js/app.bundle.js")
	if err != nil {
		t.Fatalf("Open(js/app.bundle.js) error = %v", err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if !strings.Contains(string(data), "markdown-body") {
		t.Error("app.bundle.js does not reference \"markdown-body\" — Markdown.jsx was likely not bundled (check that ThreadDetail.jsx imports it)")
	}
}

func TestFS_BundleIncludesAuthGating(t *testing.T) {
	fsys, err := frontend.FS(false)
	if err != nil {
		t.Fatalf("FS(false) error = %v", err)
	}

	f, err := fsys.Open("js/app.bundle.js")
	if err != nil {
		t.Fatalf("Open(js/app.bundle.js) error = %v", err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if !strings.Contains(string(data), "/auth/login") {
		t.Error("app.bundle.js does not reference /auth/login — App.jsx's login gate was likely not bundled")
	}
	if !strings.Contains(string(data), "/auth/logout") {
		t.Error("app.bundle.js does not reference /auth/logout — TopNav.jsx's logout control was likely not bundled")
	}
}
