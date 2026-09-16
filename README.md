# MODREPO

[English](README.md) | [日本語](README_JA.md)

MODREPO is a modern, standalone mod launcher and manager for the game **R.E.P.O.** (Semiwork Studios).
It allows players to easily discover, install, update, and manage mods directly from [Thunderstore](https://thunderstore.io/c/repo/) with isolated profile management.

## Key Features

- **Zero Game Directory Pollution**: Uses Unity Mono Doorstop with `windows.SetDllDirectory` and command-line arguments to load BepInEx from dedicated profile directories. The original R.E.P.O. game directory remains 100% untouched.
- **Thunderstore Integration**: Seamlessly fetches mods, dependencies, and updates from the official Thunderstore REPO community API (`https://thunderstore.io/c/repo/`) with ETag caching.
- **Profile Management**: Create and switch between isolated mod profiles. Export and share profiles via `.repopack` archives.
- **Automatic BepInEx Provisioning**: Automatically installs and configures `BepInEx-BepInExPack` for every profile.

## Install

### Latest Release

You can download the latest version of MODREPO from the [releases page](https://github.com/ikafly144/modrepo/releases/latest).
Windows releases are distributed as an MSI installer.

### Build from Source

To build MODREPO from source, ensure you have [Go](https://golang.org/dl/) (1.27+ recommended) and a C compiler (for CGo/Fyne) installed. Then, clone the repository and run:

```bash
git clone https://github.com/ikafly144/modrepo.git
cd modrepo
go build ./client
```

To run tests:

```bash
go test ./...
```

## Profile Archive Format

Profiles can be exported and imported using `.repopack` archive files (standard zip archives containing `modrepo.profile.json` and optional profile icons).

## Contributing

Contributions are welcome! Please submit issues or pull requests.

## License

MODREPO is licensed under the GNU General Public License v3.0. See the [LICENSE](LICENSE) file for more details.
