package main

import (
	"context"

	"craftinghub/internal/hub"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// HubService exposes hub.Hub to the frontend plus the bits that need the
// Wails app (native dialogs, file manager).
type HubService struct {
	*hub.Hub
}

// Version is the hub's own SemVer, from build/config.yml.
func (s *HubService) Version() string { return version }

// CheckHubUpdate reports a newer hub release on GitHub, or nil when current.
func (s *HubService) CheckHubUpdate(ctx context.Context) (*hub.HubUpdate, error) {
	return hub.CheckHubUpdate(ctx, version)
}

// ChooseInstallRoot opens a folder picker; returns current settings unchanged
// when the user cancels.
func (s *HubService) ChooseInstallRoot() (hub.Settings, error) {
	dir, err := application.Get().Dialog.OpenFile().
		CanChooseDirectories(true).
		CanChooseFiles(false).
		CanCreateDirectories(true).
		SetTitle("Choose install folder").
		SetDirectory(s.GetSettings().InstallRoot).
		PromptForSingleSelection()
	if err != nil || dir == "" {
		return s.GetSettings(), err
	}
	return s.SetInstallRoot(dir)
}

// OpenInstallDir reveals an installed app's folder in the OS file manager.
func (s *HubService) OpenInstallDir(id string) error {
	dir := s.InstallDir(id)
	if dir == "" {
		dir = s.GetSettings().InstallRoot
	}
	return application.Get().Env.OpenFileManager(dir, false)
}
