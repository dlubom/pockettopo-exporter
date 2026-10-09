# Native fixture and bounded reader evidence

`api-trips-ids.top` is an unchanged 542-byte native PocketTopo 1.372 fixture
from the JKTZ project, originally created by its native model/serializer through
reflection and also accepted in the GUI. These are synthetic test inputs,
not field observations. Attribution: dlubom/Jaskiniowy-Kataster-Tatr-Zachodnich,
CC BY-SA 4.0; this copied fixture retains those terms.

Pinned evidence revision: `3e3daa4156c6e6e79dce5203d8bb8e36122d571f`:

- [Native procedure and provenance](https://github.com/dlubom/Jaskiniowy-Kataster-Tatr-Zachodnich/blob/3e3daa4156c6e6e79dce5203d8bb8e36122d571f/doc/pockettopo/evidence/p01/README.md).
- [Original fixture](https://github.com/dlubom/Jaskiniowy-Kataster-Tatr-Zachodnich/blob/3e3daa4156c6e6e79dce5203d8bb8e36122d571f/doc/pockettopo/evidence/p01/cases/api-trips-ids/api-trips-ids.top).
- [Independent expected fields](https://github.com/dlubom/Jaskiniowy-Kataster-Tatr-Zachodnich/blob/3e3daa4156c6e6e79dce5203d8bb8e36122d571f/doc/pockettopo/evidence/p01/cases/api-trips-ids/expected.json).
- [Literal native API helper](https://github.com/dlubom/Jaskiniowy-Kataster-Tatr-Zachodnich/blob/3e3daa4156c6e6e79dce5203d8bb8e36122d571f/doc/pockettopo/helpers/pockettopo_fixtures.cs), `TripsAndIds`, with three explicit ticks/comments/Auto inputs.
- [Original application/runtime evidence](https://github.com/dlubom/Jaskiniowy-Kataster-Tatr-Zachodnich/blob/3e3daa4156c6e6e79dce5203d8bb8e36122d571f/doc/pockettopo/evidence/p01/native-api-evidence.json).

Copied from the local JKTZ regression path
`tests/fixtures/pockettopo/p01/cases/api-trips-ids/api-trips-ids.top`.
Git blob `bc3613b8bb92166f1ae25be205942f440dbe3132`; SHA-256
`adb83280b6d5a70f383b5542727f675b3ff8a5740058b601d3d49ed27b14895b`.
The frozen expected JSON SHA-256 is
`df6e0ca6ba56ca426e7ef44a62802f4af899b5b8fad97c7b9ab10253446cc2a6`.
P03b reproduces only header/trip values as literal test expectations;
measurements, references and drawings are outside that slice. P03c1 adds the
independent measurement expectations below; P03c2 adds references below.
Drawings remain unparsed.

| Trip | Record span | Ticks | Comment bytes | Declination raw | Auto |
| --- | --- | --- | --- | --- | --- |
| 0 | `[8,67)` | `630822816000000007` | 48 | 910 | false |
| 1 | `[67,118)` | `639259776000000009` | 40 | -637 | false |
| 2 | `[118,171)` | `639260640000000001` | 42 | -32768 | true |

Offsets follow the independently established fields and their one-byte encoded
lengths, not a Go reader oracle. Tail size is `542 - 171 = 371` bytes.

## Fresh native probe, 2026-10-09

`native-trip-read.txt` is the exact CRLF stdout of the project-authored
`scripts/reference-trip-probe.cs`, protected from Git newline normalization.
The probe calls original native `Trip.ReadList`/`Trip.Read` through reflection
on in-memory bytes, without reading measurements or writing TOP files. Native
stream positions start after the four-byte TOP header. All three fixture trips
were read; native Auto's derived zero is reported separately from raw -32768.
Comments are UTF-8 re-encoded to base64 for unambiguous console evidence.
Malformed strings are deliberate test inputs, never altered originals.

Environment: macOS arm64, Wine `wine-11.7 (Staging)`, native Microsoft .NET
Framework 2.0 x86, reported runtime `2.0.50727.42`, assembly `1.3.7.0`.

| Evidence | SHA-256 |
| --- | --- |
| Original PocketTopo EXE | `048cf57b874238ac142f1e4d3446af639ce3760b4a91724ea00bcc320eb5c371` |
| Runtime `Framework/v2.0.50727/mscorlib.dll` | `12bc52d0e0271a586f5740a9eddcd867b6f21b68274d3f65ed7a65f43e996b8d` |
| `scripts/reference-trip-probe.cs` | `fdef343e695128fc694ebc9efe0c83de06b303894f63f033df21201871e0a425` |
| `native-trip-read.txt` | `68c11c2848e1843cd4198538273ba3b53b4fe149d0557f0cdd4a34e70a3ad17e` |

The original .NET 2.0 probe confirms ticks min/max and out-of-range rejection.
It accepts a negative trip count and nonminimal zero length. Its bad UTF-8
probes omit the invalid byte/incomplete sequence. Its fifth-byte value 16 is
discarded by native Int32 decoding, yielding zero; negative length and a sixth
length byte fail. Go accepts and preserves nonminimal zero, but rejects negative
counts, malformed UTF-8 and overflowing/negative/six-byte lengths.
These observations are bounded to the tested original runtime; newer .NET
decoder behavior must not be substituted for them.

Actual optional commands run from this repository on the recorded host:

```sh
env WINEPREFIX=/Users/dariuszlubomski/.local/share/pockettopo/wineprefix WINEDEBUG=-all \
  wine /Users/dariuszlubomski/.local/share/pockettopo/wineprefix/drive_c/windows/Microsoft.NET/Framework/v2.0.50727/csc.exe \
  /nologo /out:Z:\\Users\\dariuszlubomski\\proj\\pockettopo-exporter\\build\\reference-trip-probe.exe \
  Z:\\Users\\dariuszlubomski\\proj\\pockettopo-exporter\\scripts\\reference-trip-probe.cs
env WINEPREFIX=/Users/dariuszlubomski/.local/share/pockettopo/wineprefix WINEDEBUG=-all \
  wine build/reference-trip-probe.exe \
  Z:\\Users\\dariuszlubomski\\.local\\share\\pockettopo\\app\\PocketTopoV1372\\PocketTopo.exe \
  Z:\\Users\\dariuszlubomski\\proj\\pockettopo-exporter\\internal\\top\\testdata\\api-trips-ids.top \
  >build/native-trip-read.txt
```

Compare a fresh output to the frozen stdout; do not overwrite this evidence
to match the Go implementation. Wine/.NET/PocketTopo are optional research
tools, not dependencies of ordinary tests, the CLI, or CI.

## P03c1 original measurement reader probe, 2026-10-09

`native-measurement-read.txt` is the exact 291-line CRLF stdout of the
project-authored `scripts/reference-measurement-probe.cs`, protected from Git
newline normalization. The original `Station.Read` runs on in-memory literal
bytes, retaining its flags before `Survey.Read` modifies `readOnly`. Native
`Trip.ReadList` identifies the fixture's measurement-count position, then only
the four measurement records are read. No native writer, references, drawings,
GUI or Go reader participates. Fixture positions include its four header bytes.

Environment and original assembly/runtime hashes are identical to the P03b
probe above. The new probe exited 0. The native compiler produced the executable,
but its Wine compiler/start processes remained open after compilation; this pass
stopped only those two newly created processes after the probe succeeded. The
compiler launch below is an optional evidence command, not a clean-exit CI gate.

| Evidence | SHA-256 |
| --- | --- |
| `Station.cs` | `44726d9b81881474147865a629365b9e02f0cc3e42dcec0b921613e36826c1cb` |
| `scripts/reference-measurement-probe.cs` | `8061205cf2aeeb2e43260d3f44f56f5554092a6cd568556307025300051c9f36` |
| `native-measurement-read.txt` | `a9898be3ef1191c97b87427ce9c641d4d99ec39770981feae04524dde6e1bb98` |

The probe confirms the declared flag values 1, 2, 4, 8, 16, 32 and 128;
all 256 raw flag bytes are accepted unchanged, including undeclared bit 64.
Only bit 2 consumes a comment, with empty and absent strings distinct. All
five tested Int32 distance boundaries (min, -1, 0, 1, max) remain unchanged.
Seven raw trip indices (-32768, -2, -1, 0, 1, 2, 32767) are accepted without
list-membership checking. At native trip offset 3, only nonnegative indices
change; 32767 wraps to -32766. The Go source reader never applies that context
offset. Native invalid `A FF B` reads as `AB`, nonminimal zero succeeds and a
fifth encoded-length byte 16 becomes zero. Strict Go rules reject malformed
UTF-8 and that length overflow, preserving valid nonminimal zero encodings.

The independently frozen expected JSON/helper inputs linked above establish
these measurement values. Fresh native reflection readback confirms them:

| Index | Record span | From/To raw hex | Distance mm | Azimuth/inclination raw | Flags/roll | Trip raw | Comment UTF-8 bytes |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 0 | `[175,226)` | `80000001` / `00000000` | 1000 | 0 / 0 | 2 / 0 | 0 | 30 |
| 1 | `[226,419)` | `00000000` / `000cffff` | 2000 | 16384 / 1820 | 3 / 0 | 1 | 171 |
| 2 | `[419,460)` | `000cffff` / `80000000` | 3000 | -32768 / -1820 | 2 / 0 | 2 | 20 |
| 3 | `[460,496)` | `000cffff` / `000d0000` | 4000 | -16384 / 0 | 2 / 0 | -1 | 15 |

Count span `[171,175)`; comment length spans `[195,196)`, `[246,248)`,
`[439,440)`, `[480,481)`. Go tests use literal expectations from the pinned
JSON, including 70 repeated `ą` characters in the second comment. Prefix
consumption is 496 bytes and unparsed tail size is `542 - 496 = 46`. This is
neither full-file validation nor exporter compatibility evidence.

Actual optional commands run from this repository on the recorded host:

```sh
env WINEPREFIX=/Users/dariuszlubomski/.local/share/pockettopo/wineprefix WINEDEBUG=-all \
  wine /Users/dariuszlubomski/.local/share/pockettopo/wineprefix/drive_c/windows/Microsoft.NET/Framework/v2.0.50727/csc.exe \
  /nologo '/out:Z:\Users\dariuszlubomski\proj\pockettopo-exporter\build\reference-measurement-probe.exe' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\scripts\reference-measurement-probe.cs'
env WINEPREFIX=/Users/dariuszlubomski/.local/share/pockettopo/wineprefix WINEDEBUG=-all \
  wine build/reference-measurement-probe.exe \
  'Z:\Users\dariuszlubomski\.local\share\pockettopo\app\PocketTopoV1372\PocketTopo.exe' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\internal\top\testdata\api-trips-ids.top' \
  >build/native-measurement-read.txt
```

Compare fresh output to the frozen stdout; do not regenerate expectations from
Go output. Original TOP bytes and decompilation artifacts remain unchanged.

## P03c2 native references and original reader probe, 2026-10-09

`api-references.top` is the unchanged 248-byte native fixture copied from
JKTZ `tests/fixtures/pockettopo/p01/cases/api-references/api-references.top`.
Attribution and CC BY-SA 4.0 terms above apply to this copied fixture too.
Pinned evidence revision remains `3e3daa4156c6e6e79dce5203d8bb8e36122d571f`:

- [Original native fixture](https://github.com/dlubom/Jaskiniowy-Kataster-Tatr-Zachodnich/blob/3e3daa4156c6e6e79dce5203d8bb8e36122d571f/doc/pockettopo/evidence/p01/cases/api-references/api-references.top).
- [Independent expectations](https://github.com/dlubom/Jaskiniowy-Kataster-Tatr-Zachodnich/blob/3e3daa4156c6e6e79dce5203d8bb8e36122d571f/doc/pockettopo/evidence/p01/cases/api-references/expected.json).
- [Native helper](https://github.com/dlubom/Jaskiniowy-Kataster-Tatr-Zachodnich/blob/3e3daa4156c6e6e79dce5203d8bb8e36122d571f/doc/pockettopo/helpers/pockettopo_fixtures.cs), `References`: original `DataSet.Write`, then successful native readback and exports.

Git blob `ce1c957f50f6f3e25d1471f159436c5e258e7205` was also checked through
GitHub's pinned contents API. The fixture and expected JSON SHA-256 values are
`9034cf5e52f92a8713c7823bd9be52bdb50b294966a9e9713c538b937b7feb5f` and
`93d8bb70baf630b2be35d6ec0268435c999cc4882b8ea1f143b228b2cc0c7d9f`.
Source inputs are synthetic and preserve two references at the same raw station
`0x80000001`. This does not establish a CRS, entrance or geographic tie.

| Index | Record span | East mm | North mm | Altitude mm | Comment |
| --- | --- | --- | --- | --- | --- |
| 0 | `[87,150)` | -4000000001 | -5000000002 | -1250 | `ujemne E/N/Z; żadnego przypisania CRS` (38 UTF-8 bytes) |
| 1 | `[150,206)` | 6000000003 | 7000000004 | 1500250 | `positive int64 E/N beyond int32` (31 UTF-8 bytes) |

Count span `[83,87)`; fixed fields are ID 4 / east 8 / north 8 / altitude 4
bytes. Comment-length spans `[111,112)` and `[174,175)`; comments `[112,150)`
and `[175,206)`. These offsets follow independent helper fields/encoded lengths
and were confirmed by fresh native stream positions, not inferred from Go.
Prefix consumption is 206, leaving 42 bytes uninterpreted. The existing
`api-trips-ids.top` has zero references at `[496,500)` and also leaves 42 bytes.
No drawing or complete-file validation was added.

`native-reference-read.txt` is the exact 31-line CRLF stdout of
`scripts/reference-reference-probe.cs`, protected from Git newline normalization.
Original `Reference.Read` runs on literal in-memory bytes; native `Trip.ReadList`
and `Station.Read` establish each fixture's reference-table boundary. There is
no Go reader, native writer, GUI or drawing read in this probe. Environment,
original assembly hash and .NET runtime hash are identical to the verified
P03b/P03c1 evidence above. The probe exited 0. Native compilation produced a
usable executable, but its two compiler/start processes remained open and were
stopped after successful readback; compiler completion is not a CI gate.

| New evidence | SHA-256 |
| --- | --- |
| `scripts/reference-reference-probe.cs` | `55ebd0dd9bbc06a6c8942fe1f1ce31c9a9e0e968d846ded20aa066e190fc8f1e` |
| `native-reference-read.txt` | `dc313a5a82f0987b1ce1dcaa47a19206dd94bd1d8838807510821e13985f178f` |

The original reader retains seven tested Int64 cases (min, -2^53-1, -1, 0,
1, 2^53+1, max), five Int32 boundaries and ID aliases/reserved values.
The literal endian case retains east -8644934341102468607, north
72623859790382856, altitude -2 and Unicode `Aą`. Native empty comments become
null, including nonminimal zero encodings. Nonminimal one yields `A`.
Native malformed `A FF B` becomes `AB`; fifth length byte 16 yields null;
negative/six-byte lengths fail. Go retains empty encodings/spans and keeps
strict P03b/P03c1 rejection of malformed UTF-8 and fifth bytes above 7.
The negative-reference-count loop behavior is C#/IL evidence, not a runtime
probe of `Survey.Read`. Raw sentinel units/blank display behavior are documented
from `MetricLocation`/`MetricGrid` C#/IL; the probe verifies raw numeric reading.

Actual optional commands run from this repository on the recorded host:

```sh
env WINEPREFIX=/Users/dariuszlubomski/.local/share/pockettopo/wineprefix WINEDEBUG=-all \
  wine /Users/dariuszlubomski/.local/share/pockettopo/wineprefix/drive_c/windows/Microsoft.NET/Framework/v2.0.50727/csc.exe \
  /nologo '/out:Z:\Users\dariuszlubomski\proj\pockettopo-exporter\build\reference-reference-probe.exe' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\scripts\reference-reference-probe.cs'
env WINEPREFIX=/Users/dariuszlubomski/.local/share/pockettopo/wineprefix WINEDEBUG=-all \
  wine build/reference-reference-probe.exe \
  'Z:\Users\dariuszlubomski\.local\share\pockettopo\app\PocketTopoV1372\PocketTopo.exe' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\internal\top\testdata\api-trips-ids.top' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\internal\top\testdata\api-references.top' \
  >build/native-reference-read.txt
```

Compare fresh stdout against frozen evidence; do not overwrite expected fields
to match Go results. Both source TOP hashes were unchanged after native probing.
Wine/.NET/PocketTopo remain optional evidence tools, never runtime or CI dependencies.
