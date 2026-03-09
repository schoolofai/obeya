package domain_test

import (
	"testing"

	"github.com/niladribose/obeya/internal/domain"
)

func TestLinkedProject_Fields(t *testing.T) {
	lp := domain.LinkedProject{
		Name:      "api-server",
		LocalPath: "/home/user/code/api-server",
		GitRemote: "git@github.com:user/api-server.git",
		LinkedAt:  "2026-03-09T10:00:00Z",
	}
	if lp.Name != "api-server" {
		t.Errorf("expected Name 'api-server', got %q", lp.Name)
	}
	if lp.GitRemote != "git@github.com:user/api-server.git" {
		t.Errorf("expected GitRemote set, got %q", lp.GitRemote)
	}
}

func TestItem_ProjectField(t *testing.T) {
	item := domain.Item{
		ID:      "test-1",
		Title:   "Test task",
		Project: "api-server",
	}
	if item.Project != "api-server" {
		t.Errorf("expected Project 'api-server', got %q", item.Project)
	}
}

func TestBoard_ProjectsMap(t *testing.T) {
	b := domain.NewBoard("test")
	if b.Projects == nil {
		t.Fatal("expected Projects map to be initialized")
	}
	if len(b.Projects) != 0 {
		t.Errorf("expected empty Projects map, got %d entries", len(b.Projects))
	}
}
