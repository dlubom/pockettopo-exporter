# PocketTopo Exporter 1.0.0

Convert PocketTopo **TOP v3** files offline. Your original TOP file is preserved.
No Go, Python, Wine or PocketTopo installation is needed.

## Download and unpack

Download a program ZIP from the [latest release](https://github.com/dlubom/pockettopo-exporter/releases/latest).
Choose a program archive rather than GitHub's source-code download:

| Computer | Archive suffix |
| --- | --- |
| Windows with Intel/AMD processor | `windows-amd64.zip` |
| Windows with ARM64 processor | `windows-arm64.zip` |
| Mac with Intel processor | `darwin-amd64.zip` |
| Mac with Apple Silicon (M series) | `darwin-arm64.zip` |
| Linux on Intel/AMD 64-bit | `linux-amd64.zip` |
| Linux on ARM64 | `linux-arm64.zip` |

Unpack the entire ZIP to a folder you can keep. On Windows, use **Extract All**.
On macOS/Linux, preserve the executable permission when extracting; if needed,
run `chmod +x ./pockettopo-exporter` in that folder.

The Go 1.26 runtime requires Windows 10/Server 2016 or later, macOS 12 or later,
and a supported Linux kernel (3.2 or later; architecture requirements also apply).
CI validates current runner versions, separately from these runtime minimums.
See [Go minimum requirements](https://go.dev/wiki/MinimumRequirements).

## First export

Open a terminal in the unpacked folder. Replace the input path with your file
and choose an output folder that does **not already exist**.

Windows PowerShell:

```powershell
.\pockettopo-exporter.exe --version
.\pockettopo-exporter.exe export "C:\Surveys\My cave.top" --format svx --output "C:\Surveys\My cave-survex"
```

Windows Command Prompt (cmd):

```bat
pockettopo-exporter.exe export "C:\Surveys\My cave.top" --format srv --output "C:\Surveys\My cave-walls"
```

macOS/Linux:

```sh
./pockettopo-exporter --version
./pockettopo-exporter export "$HOME/Surveys/My cave.top" --format svx --output "$HOME/Surveys/My cave-survex"
```

The new folder contains `survey.svx` (or the selected format) and `report.json`.
The report records source identity, transformations and omissions. Keep it with
the export. Exit 0 means success; errors appear on stderr and return exit 1.

| `--format` | Export |
| --- | --- |
| `svx` | Survex individual readings |
| `srv` | Walls, averaged consecutive repeated/reversed legs |
| `text` | Native PocketTopo text |
| `therion` | Native Therion interchange, `survey.txt` |
| `visualtopo` | VisualTopo, `survey.tro` |
| `toporobot` | Toporobot, `survey.text` |
| `dxf` | DXF plan; add `--view side` for extended elevation |
| `dxf3d` | Native 3D DXF |

Therion interchange is not a standalone `.th`/`.th2` project. Drawings and fixes
are omitted by the measurement-only SRV/SVX profiles. See `docs/CAPABILITIES.md`
for supported settings, limits and other losses.

## Optional Windows right-click menu

From the unpacked folder, run:

```powershell
.\pockettopo-exporter.exe windows-menu install
```

This installs **PocketTopo Exporter** for `.top` files for your Windows account.
Administrator rights are not needed. Right-click one TOP file and select an
export format. In Windows 11, first choose **Show more options**. Each action
uses the normal export defaults and writes a new folder beside the TOP file;
existing exports get a numeric suffix. The console shows success or error and
waits for Enter when opened interactively.

Use a program folder whose full path contains no `%` character; Explorer uses
percent tokens in command templates, so menu installation rejects such a program
path explicitly. TOP input names may contain `%`; normal CLI paths are unrestricted.
Keep the unpacked program in its installed location. After moving it, run the
install command again from its new folder. To remove the menu:

```powershell
.\pockettopo-exporter.exe windows-menu uninstall
```

The menu does not change your default application for opening TOP files.
Advanced settings, template context and loop closure use the command line.

## Troubleshooting

- **Output folder exists:** choose a new folder. The CLI does not overwrite one.
- **Unsupported or malformed TOP:** run `inspect "input.top" --json` and retain
  the diagnostic. Older TOP versions and unknown records are not guessed.
- **Command not found:** use the full executable path, or `./` on Unix and `.\`
  in PowerShell when running from its folder.
- **Permission denied on Unix:** restore the executable bit with `chmod +x`.
- **Windows menu missing:** use **Show more options**, and re-run install from
  the current program folder. Installation/removal errors remain explicit.

1.0.0 packages are unsigned: no Windows publisher certificate or Apple Developer
ID/notarization is supplied. A downloaded program may need explicit OS approval
before it can run. Verify the release ZIP against `SHA256SUMS`; use the OS's
per-program approval controls if blocked. Do not disable system-wide protection.
See [Apple's opening guidance](https://support.apple.com/en-us/102445).

## Agents and scripts

Use the normal CLI; it has no prompts. `inspect`, `process`, `geometry` and
`context` support `--json`; export produces a deterministic provenance report
for the chosen settings. Toporobot includes an export timestamp; pass
`--export-time` explicitly when reproducible bytes are needed. Use `--help` and
`docs/SPEC.md` for the full argument and schema contracts.

The optional `shell-export` command is for interactive Explorer use. Automations
should use `export` with an explicit new output directory.

Project code/documentation uses MIT; Go runtime notices are in
`THIRD_PARTY_NOTICES.md`. Source fixtures and original-program evidence retain
their own provenance and are not bundled as executable dependencies.
