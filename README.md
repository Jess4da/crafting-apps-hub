# Crafting Apps Hub

Desktop launcher that installs, updates, uninstalls and launches the Craft apps:

| App | Repo | Replaces |
|---|---|---|
| PhotoCraft | [storytold/photocraft](https://github.com/storytold/photocraft) | Photoshop |
| VectorCraft | [storytold/vectorcraft](https://github.com/storytold/vectorcraft) | Illustrator |
| FilmCraft | [storytold/filmcraft](https://github.com/storytold/filmcraft) | Premiere Pro |
| LightCraft | [storytold/lightcraft](https://github.com/storytold/lightcraft) | Lightroom |
| PdfCraft | [storytold/pdfcraft](https://github.com/storytold/pdfcraft) | Acrobat |
| EffectCraft | [storytold/effectcraft](https://github.com/storytold/effectcraft) | After Effects |
| DesignCraft | [storytold/designcraft](https://github.com/storytold/designcraft) | InDesign |

How it works:
- Finds each repo's latest release without GitHub's REST API, so there is no 60-requests/hour limit and no token is needed. The `github.com/storytold/<app>/releases/latest` redirect gives the version. The release's `SHA256SUMS.txt` lists the files and their checksums. The hub then picks the portable build for this OS/arch: Windows `*-portable.zip`, Linux/FreeBSD `*.tar.gz`, macOS `*-macos-universal.dmg`. The release-notes button opens the release page on GitHub.
- Every download is checked against the release's `SHA256SUMS.txt`. A release without a checksum is refused.
- Installs per user, so no admin rights are needed. Default locations:
  - Windows: `%LOCALAPPDATA%\Programs\CraftingApps`
  - macOS: `~/Applications/Crafting Apps`
  - Linux: `~/.local/share/crafting-apps`

  You can change the folder in Settings.
- Updates replace files atomically. If an update fails, the previous version is kept.
- All apps install from their **portable** builds. PhotoCraft's Windows zip also ships `portable.txt`, so it keeps its settings beside its exe in `PhotoCraftData` instead of `%APPDATA%`. The hub carries that folder over on every update and keeps it on a plain uninstall, so a reinstall gets the same settings back. A clean uninstall deletes it. The other apps always store settings in the user profile.
- Uninstall deletes the app's install folder. Tick **Also delete settings, recovery files and caches** for a clean uninstall: it also removes the folders the app writes for itself (for example `%APPDATA%\FilmCraft`, `%LOCALAPPDATA%\EffectCraft`). The dialog lists the exact folders first. Leftovers from an earlier uninstall can be cleared with the eraser button. User work is never touched: documents, the LightCraft library (`~/Pictures/LightCraft Library`), and EffectCraft presets (`~/Documents/EffectCraft`).
- Hub state is stored in `<user config dir>/CraftingAppsHub/state.json`.

## Develop

Requires Go 1.25+, [bun](https://bun.sh), [Task](https://taskfile.dev) and the `wails3` CLI (v3.0.0-beta.28).

```sh
wails3 dev          # live dev (Vite on :9245)
wails3 build        # bin/wails-crafting-apps-hub(.exe)
wails3 task package # installer for the current OS
go test ./...       # backend tests
cd frontend && bun run check
```

## Credits

- Crafting Apps Hub by [Jessada Duangpanyasawang](https://meatchopper.xyz/portfolio).
- PhotoCraft, VectorCraft, FilmCraft, LightCraft, PdfCraft, EffectCraft and DesignCraft, including their icons, by the ArtCraft Team and contributors at [storytold](https://github.com/storytold). Each app is licensed MIT OR Apache-2.0. This hub only downloads their official releases.
