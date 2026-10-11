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

## P04a overview mapping and original reader probe, 2026-10-09

`api-drawings.top` is the unchanged 680-byte native fixture copied from JKTZ
revision `3e3daa4156c6e6e79dce5203d8bb8e36122d571f`. Attribution and CC BY-SA
4.0 terms above apply to this copied fixture too. The pinned bytes were read
directly from the original Git object and compared with the local regression
fixture; no native writer was run to recreate the TOP file.

- [Original fixture](https://github.com/dlubom/Jaskiniowy-Kataster-Tatr-Zachodnich/blob/3e3daa4156c6e6e79dce5203d8bb8e36122d571f/doc/pockettopo/evidence/p01/cases/api-drawings/api-drawings.top).
- [Independent expected fields](https://github.com/dlubom/Jaskiniowy-Kataster-Tatr-Zachodnich/blob/3e3daa4156c6e6e79dce5203d8bb8e36122d571f/doc/pockettopo/evidence/p01/cases/api-drawings/expected.json).
- [Native helper](https://github.com/dlubom/Jaskiniowy-Kataster-Tatr-Zachodnich/blob/3e3daa4156c6e6e79dce5203d8bb8e36122d571f/doc/pockettopo/helpers/pockettopo_fixtures.cs), `Drawings`: literal overview origin -1234 / 5678 and native default stored scale 500. Original `DataSet.Write` produced this preserved fixture.

| Pinned evidence | Git blob | SHA-256 |
| --- | --- | --- |
| `api-drawings.top` | `07f805f0438dd2f48e9bd18d6771bf125713e8fe` | `4a494ead03cade750f670d50aa38661abda9b3e1f131d67f27a252982752c2a5` |
| Drawing `expected.json` | `5dab73dbb55a4597902973b15592705686e69efc` | `93e3c3de6f8c899d76e9e0f284df5ae3e0a2b51995ecfbc9619cafe4f5c968d3` |
| Native `pockettopo_fixtures.cs` helper | `4967b25e04a58a431a8626bd5f5b0a9acff31664` | `ae1f32a1034bbd95a59e97db47e887fabb30b06eabe55a960c9c5ebf9090d005` |

Independent trip fields give span `[8,49)` (30-byte comment), measurement count
`[49,53)`, records `[53,73)` and `[73,118)` (24-byte second comment), and zero
reference count `[118,122)`. The overview is `[122,134)`: x0 `[122,126)` = -1234,
y0 `[126,130)` = 5678, stored scale `[130,134)` = 500. Consumed 134, unparsed
tail `680 - 134 = 546`. These follow native helper fields/encoded lengths and
fresh native stream positions, independently of the Go parser.

The existing `api-references.top` overview is `[206,218)`: x0 `[206,210)` = 0,
y0 `[210,214)` = 0 and stored scale `[214,218)` = 500. Consumed 218, tail 30.
Both tails remain uninterpreted. No plan/side mapping or drawing element is read.

`native-mapping-read.txt` is exact 61-line CRLF stdout from
`scripts/reference-mapping-probe.cs`, protected from Git newline normalization.
It calls original `Mapping.Read` on literal bytes, plus both fixtures after
native `Trip.ReadList`, `Station.Read` and `Reference.Read` establish the
overview boundary. An uninitialized native Mapping avoids its UI constructor;
`Read`/`Write` access only scalar fields. Native `Mapping.Write` targets a
separate memory stream containing one record, never a TOP file or source archive.
No Go reader, drawing read or GUI interaction participates.

The original assembly/runtime and their hashes are the same as the verified
earlier evidence: Wine `wine-11.7 (Staging)`, .NET `2.0.50727.42` x86, assembly
`1.3.7.0` on macOS arm64. The final probe exited 0 with empty stderr.
An initial helper lookup hit the inherited `Reference.Read` overload; explicit
method signatures fixed it. Incomplete runs were not used as passed evidence.
Compiler/start processes remained open after producing usable executables;
only processes verified as created by these compilations were stopped.
The compilation launch is an optional evidence command, not a clean-exit CI gate.

| New evidence | SHA-256 |
| --- | --- |
| `scripts/reference-mapping-probe.cs` | `b6dd9a25dd330fa9003c6127c285b7eb69d31c5bf6de87abf53b0878c3b785ab` |
| `native-mapping-read.txt` | `ee9b96c71d2422c130d45cd86008c7aaee7a432d3e1db09d697dd0a7b252d4da` |

For each native pixel mode the probe tests 20 scales (Int32 min/max, -501,
-11, -10, -6, -5, -4, -1, 0, 1, 4, 5, 6, 9, 10, 11, 499, 500, 501), five
origin boundaries (min, -1, 0, 1, max), asymmetric signed/endian bytes and
both fixtures. Default `PixPerMm=5`; original `SetVga` sets 10. Stored -501
reads as derived -100 / -50 and rewrites as -500 in either mode. Stored -1
reads/rewrites as zero. Int32 min/max and negative/zero scales all read
successfully; no positivity check or JKTZ parser's 10..50000 bound is applied.

The literal endian record is x0 -2080177663, y0 84281096 and stored scale
-67305986. Native derived scale is -13461197 / -6730598 and rewrites as
-67305985 / -67305980. Go assertions preserve all three original Int32 values;
no derived native quotient or rewritten value is substituted into the source.

Actual optional commands run from this repository on the recorded host:

```sh
env WINEPREFIX=/Users/dariuszlubomski/.local/share/pockettopo/wineprefix WINEDEBUG=-all MVK_CONFIG_LOG_LEVEL=0 \
  wine /Users/dariuszlubomski/.local/share/pockettopo/wineprefix/drive_c/windows/Microsoft.NET/Framework/v2.0.50727/csc.exe \
  /nologo '/out:Z:\Users\dariuszlubomski\proj\pockettopo-exporter\build\reference-mapping-probe.exe' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\scripts\reference-mapping-probe.cs'
env WINEPREFIX=/Users/dariuszlubomski/.local/share/pockettopo/wineprefix WINEDEBUG=-all MVK_CONFIG_LOG_LEVEL=0 \
  wine build/reference-mapping-probe.exe \
  'Z:\Users\dariuszlubomski\.local\share\pockettopo\app\PocketTopoV1372\PocketTopo.exe' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\internal\top\testdata\api-references.top' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\internal\top\testdata\api-drawings.top' \
  >build/native-mapping-read.txt 2>build/native-mapping-read.err
```

Compare fresh stdout to the frozen evidence; do not overwrite expectations to
match Go. Both source TOP hashes remained unchanged after native probing.
The C#/IL reference files remain unchanged. Drawing/export compatibility is
outside P04a; Wine/.NET/PocketTopo remain optional evidence tools only.

## P04b plan mapping and original reader probe, 2026-10-09

The same unchanged native fixtures and pinned JKTZ helper establish the plan
mapping immediately after the overview. `Drawings` sets the plan origin to
-100 / 200 separately from overview -1234 / 5678 and side 300 / -400. The
`References` case retains native default origins 0 / 0. Stored scale is 500
in both. Original `DataSet.Read` and the v3 `Drawing.Read` branch confirm the
order; the side mapping is after variable-length plan elements, not adjacent.

| Fixture | Plan record / x0 / y0 / scale spans | Raw values | Consumed / tail |
| --- | --- | --- | --- |
| `api-references.top` | `[218,230)` / `[218,222)` / `[222,226)` / `[226,230)` | `0 / 0 / 500` | `230 / 18` |
| `api-drawings.top` | `[134,146)` / `[134,138)` / `[138,142)` / `[142,146)` | `-100 / 200 / 500` | `146 / 534` |

Fresh `native-plan-mapping-read.txt` is the exact 25-line CRLF stdout of the
separate `scripts/reference-plan-mapping-probe.cs`. Native `Trip.ReadList`,
`Station.Read` and `Reference.Read` establish the overview boundary; two
original `Mapping.Read` calls read overview then plan. The probe asserts the
independently pinned plan start/end, without Go-generated expectations.
No drawing reader, element marker/payload, side mapping, GUI or writer is used.

Both full fixtures and in-memory copies ending at the plan end succeed, in
default `PixPerMm=5` and native `SetVga` mode 10. Origins remain unchanged;
native derived scale is 100 / 50 from stored 500. Prefix-only tails are zero;
full-fixture tails are 18 / 534. Go retains the stored scale. The previous
literal signed/endian/scale evidence in P04a remains unchanged and applies to
the same original Mapping method; P04b adds plan-position/separation evidence.

Environment: macOS arm64, Wine `wine-11.7 (Staging)`, Microsoft .NET 2.0 x86,
reported runtime `2.0.50727.42`, assembly `1.3.7.0`. The original EXE, runtime,
C#/IL and all three TOP fixtures were hash-verified against the earlier values.
Final native probe exit was 0 with empty stderr. The sandboxed compiler launch
could not connect to the existing Wine server; the permitted host launch
produced the executable. Its two verified compiler/start processes remained
open and were stopped after successful readback. Compilation process completion
is not a CI gate. No original/archive/decompilation bytes were changed.

| New evidence | SHA-256 |
| --- | --- |
| `Drawing.cs` | `c07a9f255cf372cbcc2a3ecd2a31b0f1c959325749fd9155d8fd9a5649f817e7` |
| `scripts/reference-plan-mapping-probe.cs` | `c146071808f42d61e5974bd69a40e058679e14dd90e3278ee0aa4e9818f2ab2e` |
| `native-plan-mapping-read.txt` | `65f167bd0252be67f5cc925c7a57d28f99e79302cb86555a6eda025ea5f0e472` |

Actual optional commands run from this repository on the recorded host:

```sh
env WINEPREFIX=/Users/dariuszlubomski/.local/share/pockettopo/wineprefix WINEDEBUG=-all MVK_CONFIG_LOG_LEVEL=0 \
  wine /Users/dariuszlubomski/.local/share/pockettopo/wineprefix/drive_c/windows/Microsoft.NET/Framework/v2.0.50727/csc.exe \
  /nologo '/out:Z:\Users\dariuszlubomski\proj\pockettopo-exporter\build\reference-plan-mapping-probe.exe' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\scripts\reference-plan-mapping-probe.cs'
env WINEPREFIX=/Users/dariuszlubomski/.local/share/pockettopo/wineprefix WINEDEBUG=-all MVK_CONFIG_LOG_LEVEL=0 \
  wine build/reference-plan-mapping-probe.exe \
  'Z:\Users\dariuszlubomski\.local\share\pockettopo\app\PocketTopoV1372\PocketTopo.exe' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\internal\top\testdata\api-references.top' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\internal\top\testdata\api-drawings.top' \
  >build/native-plan-mapping-read.txt 2>build/native-plan-mapping-read.err
```

Compare fresh stdout to the frozen evidence; do not change expectations to match
Go. Neither prefix acceptance nor native scalar readback validates elements,
complete TOP files or exporters. Wine/.NET/PocketTopo remain evidence tools,
not application or CI dependencies.


## P04c1 first plan marker and original reader probe, 2026-10-09

The unchanged fixtures and pinned native helper above establish the first
plan marker independently of Go. `References` adds no drawing elements;
`Drawings` appends a three-point Polygon first. Native `Drawing.Write` emits
marker 0 for the former and marker 1 for the latter; `Drawing.Read` reads one
UInt8 after its v3 mapping. Fresh readback confirms:

| Fixture | Marker span / raw UInt8 | Consumed / full-file tail |
| --- | --- | --- |
| `api-references.top` | `[230,231)` / 0 | `231 / 17` |
| `api-drawings.top` | `[146,147)` / 1 | `147 / 533` |

`native-plan-marker-read.txt` is exact 15-line CRLF stdout from the separate
`scripts/reference-plan-marker-probe.cs`, protected from Git normalization.
Original `Trip.ReadList`, `Station.Read` and `Reference.Read` locate overview;
two original `Mapping.Read` calls reach the marker. The probe reads exactly
one byte using the .NET `BinaryReader` held by the original `FileReader`.
It asserts literal helper/format marker positions and values. It never calls
`Drawing.Read`, parses a payload/side mapping, writes a TOP file or uses Go.

Both full fixtures and in-memory copies ending just after the marker agree
in `PixPerMm=5` and native `SetVga` mode 10. Exact-prefix tails are zero.
Copies ending before the marker throw `EndOfStreamException` with the stream
position unchanged. No fixture is rewritten to create these bounded streams.
C#/IL dispatch (0 terminates, 1 Polygon, 3 XSection, other nonzero bytes cause
`ReadBytes(0)` then the next marker read) is documented in the main README;
this probe deliberately does not execute that dispatch.

Environment and original EXE/runtime hashes remain the same as earlier evidence:
macOS arm64, Wine `wine-11.7 (Staging)`, .NET `2.0.50727.42` x86, assembly
`1.3.7.0`. The final probe exited 0 with empty stderr. The permitted host
compilation produced a usable executable; its two verified lingering
compiler/start processes were stopped after native readback. Compilation
process completion is not a CI gate. The EXE, runtime, C#/IL and all three
TOP fixture hashes were verified before and after; no originals were edited.

| New evidence | SHA-256 |
| --- | --- |
| `scripts/reference-plan-marker-probe.cs` | `29547ea71f4e7e23e09ffdc88f370faf3f335dac0fc5e2ae7988b29a259a719b` |
| `native-plan-marker-read.txt` | `949ec2251c088881547feb092aae29a5c0f4c7ff687d324d769457447e62636a` |

The pinned JKTZ helper was fetched into ignored `build/pinned-pockettopo-fixtures.cs`
with `gh api` at revision `3e3daa4156c6e6e79dce5203d8bb8e36122d571f`,
SHA-256 `ae1f32a1034bbd95a59e97db47e887fabb30b06eabe55a960c9c5ebf9090d005`.
Its `References`/`Drawings` methods and ordered `Append` were reviewed against
C#/IL; no JKTZ parser policy was imported.

Actual optional commands run from this repository on the recorded host:

```sh
env WINEPREFIX=/Users/dariuszlubomski/.local/share/pockettopo/wineprefix WINEDEBUG=-all MVK_CONFIG_LOG_LEVEL=0 \
  wine /Users/dariuszlubomski/.local/share/pockettopo/wineprefix/drive_c/windows/Microsoft.NET/Framework/v2.0.50727/csc.exe \
  /nologo '/out:Z:\Users\dariuszlubomski\proj\pockettopo-exporter\build\reference-plan-marker-probe.exe' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\scripts\reference-plan-marker-probe.cs'
env WINEPREFIX=/Users/dariuszlubomski/.local/share/pockettopo/wineprefix WINEDEBUG=-all MVK_CONFIG_LOG_LEVEL=0 \
  wine build/reference-plan-marker-probe.exe \
  'Z:\Users\dariuszlubomski\.local\share\pockettopo\app\PocketTopoV1372\PocketTopo.exe' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\internal\top\testdata\api-references.top' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\internal\top\testdata\api-drawings.top' \
  >build/native-plan-marker-read.txt 2>build/native-plan-marker-read.err
```

Compare a fresh stdout to frozen evidence; do not adjust it to match Go.
One-byte prefix acceptance does not validate payloads, complete TOP files or
exporters. Wine/.NET/PocketTopo remain optional evidence tools only.

## P04c2 first Polygon point count and original reader probe, 2026-10-09

The unchanged `api-drawings.top` and pinned helper's `Drawings`/`Append` methods
establish a first three-point Polygon independently of Go. Fresh original
reader readback confirms marker `[146,147)` = 1, signed little-endian count
`[147,151)` = 3, consumed 151 and full-file tail 529. Full fixture and exact
count-only memory streams agree in native `PixPerMm=5` and `SetVga` mode 10.
No TOP fixture is rewritten to produce bounded streams.

The separate `scripts/reference-plan-polygon-count-probe.cs` uses original
`Trip.ReadList`, `Station.Read`, `Reference.Read` and two `Mapping.Read` calls
before reading the marker and one `ReadInt32` through the .NET `BinaryReader`
held by original `FileReader`. It never invokes `Drawing.Read`/`Polygon.Read`,
allocates points, reads coordinates/color, runs a GUI, writes a TOP, or uses
Go-generated expectations. Its literal four-byte cases confirm 0, 1, 3, 66051,
1,000,000, 1,000,001, Int32.MaxValue, -1 and Int32.MinValue, with exact-prefix
and arbitrary-tail streams. These are scalar-reader observations, not acceptance
by the complete native Polygon reader. No large native arrays were allocated.

Truncations with 0/1/2/3 count bytes throw `EndOfStreamException`. Native stream
positions become 147/148/149/150 respectively; the failed scalar read can consume
available bytes. The Go API instead returns an empty result with `truncated`
at the stable count start 147 and leaves caller bytes unchanged.

C#/IL `Polygon.Read` (RVA `0xc4b8`) reads signed Int32 at `IL_0007`, then
`newarr System.Drawing.Point` at `IL_000f`, followed by signed X/Y pairs and
a color byte even for zero points. Negative allocation behavior and the absence
of a native operational ceiling are documented from this control flow; the
probe deliberately stops before it. Go rejects negatives and counts above its
explicit bound, and stops before all point/color bytes including for zero.
The earlier JKTZ parser's signed count/resource checks were reviewed against
C#/IL; its minimum-point-byte preflight and full Polygon/color reading are
outside this count-only contract. GitHub master parser blob remains
`74965043ebb9600ad71089bb079e76eaad1268f3`.

Environment: macOS arm64, Wine `wine-11.7 (Staging)`, Microsoft .NET x86 runtime
`2.0.50727.42`, original assembly `1.3.7.0`. Final native probe exit was 0 with
empty stderr. Its 33-line CRLF stdout is frozen as
`native-plan-polygon-count-read.txt`, protected from Git text normalization.
The host compiler produced a usable executable; its two verified lingering
compiler/start processes were stopped afterwards. Sandbox Wine server binding
required a permitted host invocation. Compilation completion is not a CI gate.
Original EXE/runtime, C#/IL, all earlier native fixtures/probes/stdout and the
pinned helper remain unchanged; hashes were verified before and after.

| New evidence | SHA-256 |
| --- | --- |
| `Polygon.cs` | `63d57a5200116e9a4acaadddecec5a1381ca8173544eddafdc14d07e29237fe2` |
| `scripts/reference-plan-polygon-count-probe.cs` | `5c8ab4d740cb080fafdc7ecff5ea07006f44a53d252e37d485fc8e185d7d7cb7` |
| `native-plan-polygon-count-read.txt` | `d6d76594d0958707672909c3ab2aa7a407f83f647887a8699ea8b6a9d78d9d6b` |

Actual optional commands run from this repository on the recorded host:

```sh
env WINEPREFIX=/Users/dariuszlubomski/.local/share/pockettopo/wineprefix WINEDEBUG=-all MVK_CONFIG_LOG_LEVEL=0 \
  wine /Users/dariuszlubomski/.local/share/pockettopo/wineprefix/drive_c/windows/Microsoft.NET/Framework/v2.0.50727/csc.exe \
  /nologo '/out:Z:\Users\dariuszlubomski\proj\pockettopo-exporter\build\reference-plan-polygon-count-probe.exe' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\scripts\reference-plan-polygon-count-probe.cs'
env WINEPREFIX=/Users/dariuszlubomski/.local/share/pockettopo/wineprefix WINEDEBUG=-all MVK_CONFIG_LOG_LEVEL=0 \
  wine build/reference-plan-polygon-count-probe.exe \
  'Z:\Users\dariuszlubomski\.local\share\pockettopo\app\PocketTopoV1372\PocketTopo.exe' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\internal\top\testdata\api-drawings.top' \
  >build/native-plan-polygon-count-read.txt 2>build/native-plan-polygon-count-read.err
```

Compare fresh stdout to the frozen evidence; do not overwrite it to match Go.
Only the first Polygon count is verified. Point/color payloads, complete files,
side mappings and native exports remain outside P04c2. Wine/.NET/PocketTopo are
optional evidence tools, not application or CI dependencies.

## P04c3 first Polygon vertices and original scalar probe, 2026-10-09

The separate [points probe](../../../scripts/reference-plan-polygon-points-probe.cs)
extends the P04c2 approach without modifying earlier probes or captured outputs.
Original `Trip.ReadList`, `Station.Read`, `Reference.Read` and two `Mapping.Read`
calls locate the first marker. The `BinaryReader` held by original `FileReader`
then reads marker 1, count 3, and three signed Int32 X/Y pairs. It stops before
color and never invokes `Polygon.Read`, allocates a native point table, writes
TOP bytes, runs the GUI or derives expectations from Go.

The pinned helper remains SHA-256
`ae1f32a1034bbd95a59e97db47e887fabb30b06eabe55a960c9c5ebf9090d005`.
Its literal `Drawings` inputs and ordered `Append` independently establish
`(-6000,-2000)`, `(-5500,-1000)`, `(-4500,-1500)`. Fresh readback confirms
point records `[151,159)`, `[159,167)`, `[167,175)`, X first and Y second.
Full-file tail is 505 bytes; the exact point-only stream has no tail. Both
native pixel modes 5/10 agree. All 24 shorter point tables throw
`EndOfStreamException`, consuming their available bytes. Go instead preflights
the whole point table and returns an empty result with `truncated` at
`plan.elements[0].points`, offset 151.

Six additional literal streams cover zero/one/three points, exact prefixes and
arbitrary two-byte tails, asymmetric endian values +/-66051, Int32 min/max,
-1/0 and +/-16777217. Zero succeeds immediately at count end (position 5 in
these marker-only scalar streams), without reading a color byte. These are
bounded scalar-reader observations, not full native Polygon acceptance.

C#/IL `Polygon.Read` at RVA `0xc4b8` reads X at `IL_0041`, Y at `IL_004e`,
assigns them at `IL_0064`/`IL_0079`, and advances in stored order. It then
calculates a bounding rectangle and reads color at `IL_00f0`, even for zero
points. `Polygon.Write` emits X then Y as Int32 at `IL_003e`/`IL_005b`.
P04c3 preserves only the stored pairs and omits derived bounds and color.
The unchanged Polygon C# hash is
`63d57a5200116e9a4acaadddecec5a1381ca8173544eddafdc14d07e29237fe2`;
IL remains `ed465cef8fb81c61b37845ce7936105d70670cc8075500ac25abf9b706351a76`.
Earlier JKTZ parser blob `74965043ebb9600ad71089bb079e76eaad1268f3` was verified
on GitHub and its local point/count methods inspected: signed pairs and
minimum-byte preflight are useful, while its color bounds, whole drawing
loop and aggregate 2,000,000-point budget are not this prefix contract.

Environment: macOS arm64, installed Wine Staging 11.7, original assembly
1.3.7.0 and Microsoft .NET x86 runtime 2.0.50727.42. Final probe exit was 0,
stderr empty, stdout 61 exact CRLF lines. The first sandbox invocation could
not bind Wine's server; the permitted host invocation compiled and ran the
probe. The two identified lingering compiler/start processes were stopped
after successful execution. These optional native commands are not CI gates.
Original EXE/runtime, C#/IL, TOP fixtures, pinned helper and all earlier probe
sources/stdout were hash-checked before and after and remain unchanged.

| New evidence | SHA-256 |
| --- | --- |
| `scripts/reference-plan-polygon-points-probe.cs` | `e6782e81cb8fceeb0f31768ebd8bbe34b20393c17499897ed3dff9a3afe0394a` |
| `native-plan-polygon-points-read.txt` | `c87da2871b7252c7d66271ae248ade9a857aaa61db8ea6ecb585e7e9c00359c9` |

Actual optional commands run from the repository root:

```sh
env WINEPREFIX=/Users/dariuszlubomski/.local/share/pockettopo/wineprefix WINEDEBUG=-all MVK_CONFIG_LOG_LEVEL=0 \
  wine /Users/dariuszlubomski/.local/share/pockettopo/wineprefix/drive_c/windows/Microsoft.NET/Framework/v2.0.50727/csc.exe \
  /nologo '/out:Z:\Users\dariuszlubomski\proj\pockettopo-exporter\build\reference-plan-polygon-points-probe.exe' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\scripts\reference-plan-polygon-points-probe.cs'
env WINEPREFIX=/Users/dariuszlubomski/.local/share/pockettopo/wineprefix WINEDEBUG=-all MVK_CONFIG_LOG_LEVEL=0 \
  wine build/reference-plan-polygon-points-probe.exe \
  'Z:\Users\dariuszlubomski\.local\share\pockettopo\app\PocketTopoV1372\PocketTopo.exe' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\internal\top\testdata\api-drawings.top' \
  >build/native-plan-polygon-points-read.txt 2>build/native-plan-polygon-points-read.err
```

The frozen output is protected from text normalization. Compare future native
stdout with this evidence; never regenerate its expected values using Go.

## P04c4 first Polygon color and original scalar probe, 2026-10-09

The separate [color probe](../../../scripts/reference-plan-polygon-color-probe.cs)
extends the scalar approach by exactly one byte after the points. Original
`Trip.ReadList`, `Station.Read`, `Reference.Read` and two `Mapping.Read` calls
locate marker 1, count 3 and the three pinned raw X/Y pairs. The `BinaryReader`
held by original `FileReader` then reads the color. It never invokes
`Drawing.Read`/`Polygon.Read`, derives a pen or rectangle, opens the GUI, writes
TOP bytes or uses Go output. Every earlier probe and frozen stdout is unchanged.

The unchanged pinned helper's first plan Polygon uses `MainForm.black`.
`Polygon.Write` writes byte 1 for that pen after the ordered points. Fresh
original scalar readback confirms color 1 at `[175,176)`, consumed 176 and
full-file tail 504. Exact color-only prefixes agree in native pixel modes 5/10.
Missing color throws `EndOfStreamException` at position 175 without advancing.

Literal scalar streams with zero, one and three points verify all 256 colors,
each with no tail and arbitrary `FF 80` tails. They consume exactly one byte
at positions 5, 13 and 29 respectively; missing color at each position fails
without advancing. Point bytes are the independent signed/endian literals from
the earlier probe. These 1,536 successful literal cases, three missing-color
cases, six fixture cases and three environment lines give **1,548 exact CRLF
stdout lines**. They establish scalar reads/stops, not full native Polygon
acceptance or rendering. All rows and line endings were independently checked.

C#/IL `Polygon.Read` (RVA `0xc4b8`) reads UInt8 color at `IL_00f0`, after
points and derived bounds, even for zero points. Its branches select 1 black,
2 gray, 3 brown, 4 blue, 5 red, 7 orange, and default green (including 6 and
all unknown codes). `Polygon.Write` (RVA `0xc350`) writes those named codes
and default 6. That native rendering/write normalization is documented from
C#/IL; this probe does not execute it, and the Go source preserves every byte.
Earlier JKTZ parser blob `74965043ebb9600ad71089bb079e76eaad1268f3` was fetched
and hash-verified; its `1..7` color restriction was checked against native
branches and deliberately not adopted.

Environment: macOS arm64, Wine `wine-11.7 (Staging)`, original assembly
`1.3.7.0`, Microsoft .NET x86 runtime `2.0.50727.42`. Original EXE/runtime,
C#/IL, pinned helper, TOP fixtures and all earlier probes/stdout were
hash-checked before and after and remain unchanged. Final native probe exit
was 0 with empty stderr. The sandbox attempt could not bind Wine's server;
the permitted host invocation compiled and ran the probe. Only the two
identified lingering compiler/start processes created for this compilation
were stopped. These optional evidence commands are not CI gates.

| New evidence | SHA-256 |
| --- | --- |
| `scripts/reference-plan-polygon-color-probe.cs` | `d5cf8ab377feea0f136ed8bc5bd02c5dcb10c8898844437e5d0e8cf2ae98d39e` |
| `native-plan-polygon-color-read.txt` | `405c76eb6d14025a2110e259d8db53e2b16cf0c9a190261a2c53cc3ae424e693` |
| Optional compiled probe (`build/reference-plan-polygon-color-probe.exe`) | `2bbb441248f59bc1f6fff89302b2e21c5387b0d18fae85383105b9ca07a61b8c` |

The unchanged Polygon C# hash is
`63d57a5200116e9a4acaadddecec5a1381ca8173544eddafdc14d07e29237fe2`;
IL remains `ed465cef8fb81c61b37845ce7936105d70670cc8075500ac25abf9b706351a76`.
The helper remains `ae1f32a1034bbd95a59e97db47e887fabb30b06eabe55a960c9c5ebf9090d005`.
Original EXE/runtime and copied fixture hashes are recorded above.

Actual optional commands run from the repository root:

```sh
env WINEPREFIX=/Users/dariuszlubomski/.local/share/pockettopo/wineprefix WINEDEBUG=-all MVK_CONFIG_LOG_LEVEL=0 \
  wine /Users/dariuszlubomski/.local/share/pockettopo/wineprefix/drive_c/windows/Microsoft.NET/Framework/v2.0.50727/csc.exe \
  /nologo '/out:Z:\Users\dariuszlubomski\proj\pockettopo-exporter\build\reference-plan-polygon-color-probe.exe' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\scripts\reference-plan-polygon-color-probe.cs'
env WINEPREFIX=/Users/dariuszlubomski/.local/share/pockettopo/wineprefix WINEDEBUG=-all MVK_CONFIG_LOG_LEVEL=0 \
  wine build/reference-plan-polygon-color-probe.exe \
  'Z:\Users\dariuszlubomski\.local\share\pockettopo\app\PocketTopoV1372\PocketTopo.exe' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\internal\top\testdata\api-drawings.top' \
  >build/native-plan-polygon-color-read.txt 2>build/native-plan-polygon-color-read.err
```

The frozen stdout is protected from text normalization. Compare future native
output with this evidence; never regenerate expected values using Go.

## P04c5 next plan marker and original scalar probe, 2026-10-09

The separate [next-marker probe](../../../scripts/reference-plan-next-marker-probe.cs)
extends the scalar approach through exactly one byte after the first Polygon
color. Original table/Mapping.Read methods locate the first Polygon; the
BinaryReader held by original FileReader reads its pinned count, signed points,
color 1, and next marker 1 at `[176,177)`. Full-file tail is 503; exact prefix
has no tail. Missing marker throws EndOfStreamException at 176 without advancing.
Fixture streams agree in native PixPerMm 5/10. No Drawing.Read/Polygon.Read,
GUI, TOP write or Go-generated expectations are used.

The hash-verified pinned helper appends the gray Polygon immediately after the
black Polygon. The earlier helper/probe/stdout and TOP bytes are unchanged.
Independent zero/one/three-point literal streams each verify all 256 marker bytes
with complementary colors `255-marker`, exact prefixes and arbitrary `FF 80`
tails. Missing marker is checked for each point-count case. The 1,536 successful
literal cases, three missing cases, six fixture cases and three environment
lines produce **1,548 exact CRLF rows**. Every row and line ending was checked
independently. They establish scalar preservation and stops, not native dispatch,
complete drawing acceptance or exporter compatibility.

C#/IL Drawing.Read (RVA `0x10a30`) calls next ReadByte at IL_00c3 after the
element read at IL_007f. The loop terminates on 0; 1 selects Polygon, 3 XSection,
and unknown nonzero markers skip ReadBytes(0) then read the next marker.
The Go prefix preserves every byte without executing that dispatch. The verified
JKTZ parser blob `74965043ebb9600ad71089bb079e76eaad1268f3` rejects unknown
markers and reads full drawings; those policies were not adopted.

Environment: macOS arm64, installed Wine Staging 11.7, original assembly 1.3.7.0,
Microsoft .NET x86 runtime 2.0.50727.42. Final probe exited 0, stderr empty.
The permitted host compilation produced a usable executable; only its two
identified lingering compiler/start processes were stopped afterwards. Native
commands are optional evidence tools, not application/CI dependencies. All
original EXE/runtime, C#/IL, helper, TOP and earlier probe/output hashes were
checked before and after and remain unchanged.

| New evidence | SHA-256 |
| --- | --- |
| `scripts/reference-plan-next-marker-probe.cs` | `bfe6bcdfb917adb8b00df390db135059d0806bd9e60f629f7c4b0d4ceecfe9f1` |
| `native-plan-next-marker-read.txt` | `aecf642cfcf85405e2855d827bff67a5b79ead82e0dcc3d71f8a72b3944a15d3` |
| Optional compiled probe (`build/reference-plan-next-marker-probe.exe`) | `f1583022973fc8167400a8fadd570c005a92aca35111e9f1f096a2b013c968fc` |

Drawing C# remains `c07a9f255cf372cbcc2a3ecd2a31b0f1c959325749fd9155d8fd9a5649f817e7`;
IL remains `ed465cef8fb81c61b37845ce7936105d70670cc8075500ac25abf9b706351a76`.
The helper remains `ae1f32a1034bbd95a59e97db47e887fabb30b06eabe55a960c9c5ebf9090d005`.
Original EXE/runtime and fixture hashes remain as recorded above.

Actual optional commands run from the repository root:

```sh
env WINEPREFIX=/Users/dariuszlubomski/.local/share/pockettopo/wineprefix WINEDEBUG=-all MVK_CONFIG_LOG_LEVEL=0 \
  wine /Users/dariuszlubomski/.local/share/pockettopo/wineprefix/drive_c/windows/Microsoft.NET/Framework/v2.0.50727/csc.exe \
  /nologo '/out:Z:\Users\dariuszlubomski\proj\pockettopo-exporter\build\reference-plan-next-marker-probe.exe' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\scripts\reference-plan-next-marker-probe.cs'
env WINEPREFIX=/Users/dariuszlubomski/.local/share/pockettopo/wineprefix WINEDEBUG=-all MVK_CONFIG_LOG_LEVEL=0 \
  wine build/reference-plan-next-marker-probe.exe \
  'Z:\Users\dariuszlubomski\.local\share\pockettopo\app\PocketTopoV1372\PocketTopo.exe' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\internal\top\testdata\api-drawings.top' \
  >build/native-plan-next-marker-read.txt 2>build/native-plan-next-marker-read.err
```

Frozen stdout is protected from text normalization. Compare future native
stdout to this evidence; never regenerate expectations from Go.

## P04c6 second Polygon count and original scalar probe, 2026-10-10

The separate [second-count probe](../../../scripts/reference-plan-second-polygon-count-probe.cs)
uses original table/Mapping.Read methods and the BinaryReader held by original
FileReader. It reads the first Polygon's pinned scalars, next marker 1, and one
signed Int32, stopping before second points/color. No Drawing.Read/Polygon.Read,
GUI, TOP writer, large point allocation or Go oracle participates.

The unchanged hash-verified pinned helper's Drawings/Append methods establish
three points in the second gray Polygon. Format layout puts its count at
`[177,181)`; fresh native full/exact streams confirm count 3, consumed 181 and
full-file tail 499 in pixel modes 5/10. Every 0–3-byte incomplete count throws
EndOfStreamException after consuming available bytes. Go returns an empty result
and its stable count-field start, preserving inherited strict error policy.

Independent zero/one/three-first-point literal streams confirm second counts
0, 1, 3, 66051, 1,000,000, 1,000,001, Int32.MaxValue, -1, Int32.MinValue and
-66051, with exact prefixes and arbitrary FF 80 tails. They consume exactly four
bytes at starts 7/15/31, without second-point allocation or color. Each literal
first-point case also tests all four count truncations. Scalar native ReadInt32
accepts all signed values; Go rejects negative/above-bound counts. These are
scalar observations, not full native Polygon/drawing/export acceptance.

The 60 successful literals, 12 truncated literals, 12 fixture calls and three
environment lines produce **87 exact CRLF stdout rows**, all independently
checked against literal expected values and positions. Final probe exit was 0
and stderr empty. Environment: macOS arm64, Wine Staging 11.7, original assembly
1.3.7.0, Microsoft .NET x86 2.0.50727.42. Compilation produced the usable
executable without diagnostics; only its two identified lingering compiler/start
processes were stopped after successful readback. Native commands are optional
evidence tools, not application or CI dependencies.

Inspected the analysis map, Drawing.Read (RVA 0x10a30), Polygon.Read (0xc4b8)
and Polygon.Write (0xc350), with matching IL. Next ReadByte is IL_00c3;
marker 1 dispatches Polygon.Read, whose signed ReadInt32 at IL_0007 precedes
newarr at IL_000f. Write emits marker 1 then Int32 length at IL_001c. Second
allocation/payload behavior is documented from C#/IL and was not executed.

Original EXE/runtime, C#/IL, helper, source TOP fixtures and every earlier
probe/stdout remain byte-for-byte unchanged, checked before and after. Polygon
C# remains 63d57a5200116e9a4acaadddecec5a1381ca8173544eddafdc14d07e29237fe2;
Drawing remains c07a9f255cf372cbcc2a3ecd2a31b0f1c959325749fd9155d8fd9a5649f817e7;
IL remains ed465cef8fb81c61b37845ce7936105d70670cc8075500ac25abf9b706351a76;
helper remains ae1f32a1034bbd95a59e97db47e887fabb30b06eabe55a960c9c5ebf9090d005.
Earlier JKTZ point-byte preflight/full drawing reads and aggregate budgets remain
outside this count-only prefix; this increment reuses no additional parser code.

| New evidence | SHA-256 |
| --- | --- |
| `scripts/reference-plan-second-polygon-count-probe.cs` | `a89deebc9013990defa93fa867f42ae4f7f2c6c89fe23f0f4c2d2cfac943e22d` |
| `native-plan-second-polygon-count-read.txt` | `ec79176c505408103c1043763cfa18b144ae4b903edc454630a3dcc5a94eb64b` |
| `Optional compiled probe (build/reference-plan-second-polygon-count-probe.exe)` | `5b19e07f02a37649c4c54dc0f6815d5718ff93567d046c89a992e080674e37a0` |

Actual optional commands run from the repository root:

```sh
env WINEPREFIX=/Users/dariuszlubomski/.local/share/pockettopo/wineprefix WINEDEBUG=-all MVK_CONFIG_LOG_LEVEL=0 \
  wine /Users/dariuszlubomski/.local/share/pockettopo/wineprefix/drive_c/windows/Microsoft.NET/Framework/v2.0.50727/csc.exe \
  /nologo '/out:Z:\Users\dariuszlubomski\proj\pockettopo-exporter\build\reference-plan-second-polygon-count-probe.exe' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\scripts\reference-plan-second-polygon-count-probe.cs'
env WINEPREFIX=/Users/dariuszlubomski/.local/share/pockettopo/wineprefix WINEDEBUG=-all MVK_CONFIG_LOG_LEVEL=0 \
  wine build/reference-plan-second-polygon-count-probe.exe \
  'Z:\Users\dariuszlubomski\.local\share\pockettopo\app\PocketTopoV1372\PocketTopo.exe' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\internal\top\testdata\api-drawings.top' \
  >build/native-plan-second-polygon-count-read.txt 2>build/native-plan-second-polygon-count-read.err
```

Frozen stdout is protected from text normalization. Compare future native
stdout with this evidence; never regenerate expected values using Go.

## P04c7 second Polygon vertices and original scalar probe, 2026-10-10

The separate [second-points probe](../../../scripts/reference-plan-second-polygon-points-probe.cs)
uses original table/Mapping.Read methods and FileReader's BinaryReader. It reads
first Polygon scalars, then second marker/count/ordered X/Y, stopping before
second color, including zero points. No Drawing.Read/Polygon.Read, GUI, TOP
writer, large native allocation or Go-generated oracle participates.

The unchanged hash-verified pinned helper's Drawings/Append methods establish
second gray points `(-4000,-2000)`, `(-3500,-1000)`, `(-2500,-1500)`.
Fresh full/exact streams in pixel modes 5/10 confirm table `[181,205)`, consumed
205 and full-file tail 475. Every 0–23-byte incomplete point table throws
EndOfStreamException after consuming available bytes. Go preflights before
allocation and returns an empty result with the stable table start.

Nine combinations of zero/one/three first and second points, each with exact
and arbitrary FF 80 tails, confirm signed/endian boundaries, order and second
starts 11/19/35. Zero second points read no additional byte. The 18 successful
literals, 52 fixture calls and three environment lines produce **73 exact CRLF
stdout rows**, independently checked against literal values and positions.
Final native exit was 0 and stderr empty. These are scalar observations;
full native Polygon/drawing/export acceptance is unproven.

Environment: macOS arm64, Wine Staging 11.7, original assembly 1.3.7.0,
Microsoft .NET x86 runtime 2.0.50727.42. Compilation produced the usable
executable without diagnostics; only its two identified lingering compiler/start
processes were stopped after successful readback. Native commands are optional
evidence tools, not application or CI dependencies.

Inspected the analysis map, Drawing.Read/Polygon.Read/Polygon.Write and matching
IL. Signed X/Y reads are IL_0041/IL_004e; writes IL_003e/IL_005b. Native derived
rectangle and color at IL_00f0 follow all vertices, including zero points.
This probe executes neither operation. Original EXE/runtime, C#/IL, helper,
source TOP and every prior probe/stdout remain byte-for-byte intact, checked
before and after. Their hashes remain as recorded above.

| New evidence | SHA-256 |
| --- | --- |
| `scripts/reference-plan-second-polygon-points-probe.cs` | `7389f14338f854c6287f5eb308c301d902543ef4d62e9b4764590300377e0498` |
| `native-plan-second-polygon-points-read.txt` | `eec3d7adc5f30a8280de92769223b16cfe4f2ee6073e5c3c3d68b873b7891e67` |
| Optional compiled probe (`build/reference-plan-second-polygon-points-probe.exe`) | `c1dfa587c6c6b74489ce6e2b58787c53c3f81d20916c2706f05452585483cdda` |

Actual optional commands run from the repository root:

```sh
env WINEPREFIX=/Users/dariuszlubomski/.local/share/pockettopo/wineprefix WINEDEBUG=-all MVK_CONFIG_LOG_LEVEL=0 \
  wine /Users/dariuszlubomski/.local/share/pockettopo/wineprefix/drive_c/windows/Microsoft.NET/Framework/v2.0.50727/csc.exe \
  /nologo '/out:Z:\Users\dariuszlubomski\proj\pockettopo-exporter\build\reference-plan-second-polygon-points-probe.exe' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\scripts\reference-plan-second-polygon-points-probe.cs'
env WINEPREFIX=/Users/dariuszlubomski/.local/share/pockettopo/wineprefix WINEDEBUG=-all MVK_CONFIG_LOG_LEVEL=0 \
  wine build/reference-plan-second-polygon-points-probe.exe \
  'Z:\Users\dariuszlubomski\.local\share\pockettopo\app\PocketTopoV1372\PocketTopo.exe' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\internal\top\testdata\api-drawings.top' \
  >build/native-plan-second-polygon-points-read.txt 2>build/native-plan-second-polygon-points-read.err
```

Frozen stdout is protected from text normalization. Compare future native
stdout to this evidence; never regenerate expected values using Go.

## P04c8 second Polygon color and original scalar probe, 2026-10-10

The separate [second-color probe](../../../scripts/reference-plan-second-polygon-color-probe.cs)
uses original table/Mapping.Read methods and FileReader's BinaryReader. It reads
first Polygon scalars, second marker/count/ordered X/Y, then exactly one raw
UInt8 color, including zero points. No Drawing.Read/Polygon.Read, derived
rectangle, pen selection, GUI, TOP write or Go-generated oracle participates.

The unchanged pinned helper's second plan Polygon is gray. Polygon.Write emits
code 2 after its three X/Y pairs; independent layout puts color at `[205,206)`.
Fresh full/exact streams in pixel modes 5/10 confirm raw 2, consumed 206 and
full-file tail 474. Missing color throws EndOfStreamException at 205 without
advancing. The probe stops before the following marker.

Independent zero/one/three-first/second-point literal streams cover all 256
second color bytes with exact prefixes and arbitrary FF 80 tails. First color
stays 129; signed/endian point boundaries and ordered pairs are asserted before
the second color read. Missing color fails in all nine point-table combinations,
including zero second points. Starts are `11 + 8 * (first_count + second_count)`;
success consumes exactly one further byte. The 4,608 successful literals, nine
missing literals, six fixture calls and three environment lines produce
**4,626 exact CRLF rows**, independently checked row by row against literal
values and positions. Final native exit was 0 and stderr empty.

Environment: macOS arm64, Wine Staging 11.7, original assembly 1.3.7.0,
Microsoft .NET x86 runtime 2.0.50727.42. Compilation produced the usable
executable without diagnostics; only its two identified lingering compiler/start
processes were stopped after successful readback. These optional commands are
evidence tools, not application or CI dependencies.

After the analysis map, inspected Polygon.Read (RVA 0xc4b8), Write (0xc350)
and matching IL. ReadByte at IL_00f0 follows vertices and the derived rectangle
even for zero points. Native color mapping is 1 black, 2 gray, 3 brown, 4 blue,
5 red, 7 orange and default green (including 6/unknown). Write emits these codes
and default 6. Mapping/rendering/write behavior is C#/IL evidence; it is not
executed by the scalar probe or implemented by the Go source reader.
The reviewed JKTZ parser's 1..7 restriction and aggregate point budget remain
outside this contract. Original EXE/runtime, C#/IL, pinned helper, source TOP
and every earlier probe/stdout were checked before and after and are unchanged.
These observations do not prove full native Polygon/drawing/export acceptance.

| New evidence | SHA-256 |
| --- | --- |
| `scripts/reference-plan-second-polygon-color-probe.cs` | `0a6e3b1dc0a1eb3a6c7622e53b692b7f2a0d8d505c8e3d2425ddfefb87c62f8b` |
| `native-plan-second-polygon-color-read.txt` | `03f350dd7456bdf1aaa2b0695a758a29c108d053698d0b87fee0f3116233e1c5` |
| Optional compiled probe (`build/reference-plan-second-polygon-color-probe.exe`) | `4013e9e03aa69e5ce3091a64199ce9032bdec3c77798a3f353e7aad6bce4c836` |

Actual optional commands run from the repository root:

```sh
env WINEPREFIX=/Users/dariuszlubomski/.local/share/pockettopo/wineprefix WINEDEBUG=-all MVK_CONFIG_LOG_LEVEL=0 \
  wine /Users/dariuszlubomski/.local/share/pockettopo/wineprefix/drive_c/windows/Microsoft.NET/Framework/v2.0.50727/csc.exe \
  /nologo '/out:Z:\Users\dariuszlubomski\proj\pockettopo-exporter\build\reference-plan-second-polygon-color-probe.exe' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\scripts\reference-plan-second-polygon-color-probe.cs'
env WINEPREFIX=/Users/dariuszlubomski/.local/share/pockettopo/wineprefix WINEDEBUG=-all MVK_CONFIG_LOG_LEVEL=0 \
  wine build/reference-plan-second-polygon-color-probe.exe \
  'Z:\Users\dariuszlubomski\.local\share\pockettopo\app\PocketTopoV1372\PocketTopo.exe' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\internal\top\testdata\api-drawings.top' \
  >build/native-plan-second-polygon-color-read.txt 2>build/native-plan-second-polygon-color-read.err
```

Frozen stdout is protected from text normalization. Compare future native
stdout to this evidence; never regenerate expected values using Go.

## P04c9 following marker and original scalar probe, 2026-10-10

The separate [following-marker probe](../../../scripts/reference-plan-following-marker-probe.cs)
uses original table/Mapping.Read methods and FileReader's BinaryReader, then
first and second Polygon scalars and exactly one following raw byte. It invokes
no Drawing.Read/Polygon.Read, derived geometry/pen selection, GUI or TOP writer.
The unchanged pinned helper's third plan Polygon is brown; its marker 1 follows
second gray color 2 at `[206,207)`. Full/exact streams in pixel modes 5/10 confirm
raw 1, consumed 207 and full-file tail 473. Missing marker throws
EndOfStreamException at 206 without advancing.

Zero/one/three points in both tables cover all 256 following marker bytes with
complementary second colors (`255 - marker`), exact prefixes and FF 80 tails.
First color stays 129; signed/endian boundary coordinates and order are asserted
before the marker. Missing markers fail in all nine table combinations, including
zero points. Literal starts are `12 + 8 * (first_count + second_count)`; success
consumes exactly one further byte. All **4,626 exact CRLF rows** (4,608 successful
literals, nine missing literals, six fixture calls, three environment lines)
were checked independently against literal expectations, without Go output.
Native exit 0, empty stderr.

Environment: macOS arm64, Wine Staging 11.7, original assembly 1.3.7.0,
Microsoft .NET x86 runtime 2.0.50727.42. Compilation emitted the usable EXE
without diagnostics; only its two identified lingering compiler/start processes
were stopped after successful readback. Wine's server socket required execution
outside the sandbox; these optional tools remain separate from application and CI dependencies.
After the analysis map, inspected Drawing.Read (RVA 0x10a30) ReadByte at IL_00c3,
zero comparison/loop at IL_00ca, 1/3 dispatch, unknown ReadBytes(0), and
Polygon.Read/Write/matching IL. Dispatch/termination are source inspection
claims; the probe stops before dispatch, even for zero. This is bounded scalar
evidence, not full native drawing/export acceptance.

| New evidence | SHA-256 |
| --- | --- |
| `scripts/reference-plan-following-marker-probe.cs` | `0aaaac33810928ffaa6e9720adc81cd0c370204337580d933f695956df654133` |
| `native-plan-following-marker-read.txt` | `19596fb11816d3132532a723727c9f2be82c8053172271bae5a112542d2f0cf4` |
| Optional compiled probe (`build/reference-plan-following-marker-probe.exe`) | `df333a816f3ed37747e73c32c6366390c7d645b39a9d4e83babcaea24d5d963c` |

Drawing.cs SHA-256 is `c07a9f255cf372cbcc2a3ecd2a31b0f1c959325749fd9155d8fd9a5649f817e7`,
Polygon.cs `63d57a5200116e9a4acaadddecec5a1381ca8173544eddafdc14d07e29237fe2`,
complete IL `ed465cef8fb81c61b37845ce7936105d70670cc8075500ac25abf9b706351a76`.
The pinned helper was read from its original Git object and matched the earlier
`ae1f32a1034bbd95a59e97db47e887fabb30b06eabe55a960c9c5ebf9090d005` hash.
Original EXE/runtime, C#/IL, TOP inputs and all earlier probes/stdout remain
byte-for-byte unchanged; frozen stdout is protected from text normalization.

Actual optional commands run from the repository root:

```sh
env WINEPREFIX=/Users/dariuszlubomski/.local/share/pockettopo/wineprefix WINEDEBUG=-all MVK_CONFIG_LOG_LEVEL=0 \
  wine /Users/dariuszlubomski/.local/share/pockettopo/wineprefix/drive_c/windows/Microsoft.NET/Framework/v2.0.50727/csc.exe \
  /nologo '/out:Z:\Users\dariuszlubomski\proj\pockettopo-exporter\build\reference-plan-following-marker-probe.exe' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\scripts\reference-plan-following-marker-probe.cs'
env WINEPREFIX=/Users/dariuszlubomski/.local/share/pockettopo/wineprefix WINEDEBUG=-all MVK_CONFIG_LOG_LEVEL=0 \
  wine build/reference-plan-following-marker-probe.exe \
  'Z:\Users\dariuszlubomski\.local\share\pockettopo\app\PocketTopoV1372\PocketTopo.exe' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\internal\top\testdata\api-drawings.top' \
  >build/native-plan-following-marker-read.txt 2>build/native-plan-following-marker-read.err
```

## P04c10 third Polygon count and original scalar probe, 2026-10-10

The separate [third-count probe](../../../scripts/reference-plan-third-polygon-count-probe.cs)
uses original table/Mapping.Read methods and FileReader's BinaryReader, then
first/second Polygon scalars, following marker 1 and exactly one signed Int32.
No Drawing.Read/Polygon.Read, third vertices/color, geometry, GUI or TOP write.
The unchanged pinned helper's third brown plan Polygon has three points; the
marker lies at `[206,207)` and count at `[207,211)`. Full/exact streams in pixel
modes 5/10 confirm count 3, consumed 211, full-file tail 469. Four truncated
prefixes throw EndOfStreamException after consuming available count bytes.

Independent literal streams have zero/one/three points in each earlier table,
first color 129 and second color 255. Counts from literal bytes are 0, 1, 3,
66051, 1000000, 1000001, Int32.MaxValue, -1, Int32.MinValue and -66051. Each
case has an exact prefix and FF 80 tail; each table combination tests all four
partial-count lengths. Literal count starts are `13 + 8 * (first + second)`.
All **231 exact CRLF rows** (180 successful literals, 36 partial literals,
12 fixture reads and three environment lines) match independently constructed
literal expectations; native exit 0 and stderr empty. No Go-generated oracle.

After the analysis map, inspected Drawing.Read marker-1 dispatch at IL_0057,
Polygon.Read ReadInt32 at IL_0007, newarr at IL_000f and Polygon.Write count
Write(int32) at IL_001c. Go preserves scalar values but rejects negative/over-limit
counts explicitly and performs no third allocation/preflight. Native scalar
reads accept negative/large counts; this probe does not invoke the native point
allocation. Full native drawing/export compatibility remains unvalidated.
JKTZ's aggregate point budget/preflight and color restriction were reviewed and
not adopted. Original EXE/runtime, C#/IL, pinned helper, source TOP and every
prior probe/stdout remain unchanged, verified against the preservation snapshot.

Environment: macOS arm64, Wine Staging 11.7, original assembly 1.3.7.0,
Microsoft .NET x86 runtime 2.0.50727.42. Compilation produced the usable EXE
without diagnostics; its two identified lingering compiler/start processes were
stopped after successful readback. Wine's existing server socket required execution outside
the sandbox. These optional evidence commands are not application/CI dependencies.

| New evidence | SHA-256 |
| --- | --- |
| `scripts/reference-plan-third-polygon-count-probe.cs` | `30f547a6c3ed43d78ba03f2a640d9ec446e0fb37faf489cbc57a290c059694f9` |
| `native-plan-third-polygon-count-read.txt` | `e3d431ee54000b28a7517058b04b943b34ff2e460813bd25cafc4bd915650179` |
| Optional compiled probe (`build/reference-plan-third-polygon-count-probe.exe`) | `c07b6417ed796e0806e96b98e1d2b017a54433b7f91f4425a0808423e820d7d4` |

Drawing.cs SHA-256 remains `c07a9f255cf372cbcc2a3ecd2a31b0f1c959325749fd9155d8fd9a5649f817e7`,
Polygon.cs `63d57a5200116e9a4acaadddecec5a1381ca8173544eddafdc14d07e29237fe2`,
complete IL `ed465cef8fb81c61b37845ce7936105d70670cc8075500ac25abf9b706351a76`.
The helper read from its original Git object matches the recorded
`ae1f32a1034bbd95a59e97db47e887fabb30b06eabe55a960c9c5ebf9090d005` hash.

Actual optional commands run from the repository root:

```sh
env WINEPREFIX=/Users/dariuszlubomski/.local/share/pockettopo/wineprefix WINEDEBUG=-all MVK_CONFIG_LOG_LEVEL=0 \
  wine /Users/dariuszlubomski/.local/share/pockettopo/wineprefix/drive_c/windows/Microsoft.NET/Framework/v2.0.50727/csc.exe \
  /nologo '/out:Z:\Users\dariuszlubomski\proj\pockettopo-exporter\build\reference-plan-third-polygon-count-probe.exe' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\scripts\reference-plan-third-polygon-count-probe.cs'
env WINEPREFIX=/Users/dariuszlubomski/.local/share/pockettopo/wineprefix WINEDEBUG=-all MVK_CONFIG_LOG_LEVEL=0 \
  wine build/reference-plan-third-polygon-count-probe.exe \
  'Z:\Users\dariuszlubomski\.local\share\pockettopo\app\PocketTopoV1372\PocketTopo.exe' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\internal\top\testdata\api-drawings.top' \
  >build/native-plan-third-polygon-count-read.txt 2>build/native-plan-third-polygon-count-read.err
```

Frozen stdout is protected from text normalization; never regenerate expected values using Go.

## P04c11 third Polygon points and original scalar probe, 2026-10-10

The separate [third-points probe](../../../scripts/reference-plan-third-polygon-points-probe.cs)
uses original table/Mapping.Read methods and FileReader's BinaryReader, then
first/second Polygon scalar fields, following marker/count and third signed X/Y
pairs only. No Drawing.Read/Polygon.Read, third color, geometry, GUI or TOP write.
The unchanged pinned helper's third brown Polygon has the ordered points
`(-2000,-2000)`, `(-1500,-1000)`, `(-500,-1500)` at `[211,235)`.
Full/exact streams in pixel modes 5/10 confirm consumed 235, full-file tail 445.
All 24 partial-point lengths throw EndOfStreamException after consuming the
available bytes. Go instead returns an empty result and the point-table start.

Independent literal streams cover zero/one/three points in each of all three
tables; first color 129 and second color 255. Literal point values are
`(66051,-66051)`, `(Int32.MinValue,Int32.MaxValue)`, `(-1,0)` and
`(16777217,-16777217)`. Every complete stream is read with exact and FF 80 tails;
every partial third-point length is also read. Literal third-point starts are
`17 + 8 * (first + second)`. All **397 exact CRLF rows** (54 successful literals,
288 partial literals, 52 fixture reads and three environment lines) match an
independently constructed literal oracle byte-for-byte; native exit 0, empty
stderr. No Go-generated oracle or regenerated earlier evidence.

After the analysis map, inspected Polygon.Read's X/Y ReadInt32 at IL_0041/IL_004e
and assignments at IL_0064/IL_0079, and Polygon.Write's X/Y Write(int32) at
IL_003e/IL_005b. Allocation, derived rectangle and color remain outside this
reader. JKTZ's signed point decoding was reviewed; aggregate budgets and color
restriction remain outside this contract. Scalar evidence does not establish
full native drawing/export compatibility.

Environment: macOS arm64, Wine Staging 11.7, original assembly 1.3.7.0,
Microsoft .NET x86 runtime 2.0.50727.42. Compilation produced the usable EXE
without diagnostics; only its two identified lingering compiler/start processes
were stopped after successful readback. Wine's server socket required execution
outside the sandbox. Optional tools remain separate from application/CI dependencies.

| New evidence | SHA-256 |
| --- | --- |
| `scripts/reference-plan-third-polygon-points-probe.cs` | `1bcfd5ae8f59ff7c4dc5dcfd39dafe1b5a6de7e50e22482bbf7b17cea80e58e9` |
| `native-plan-third-polygon-points-read.txt` | `345ffb288789b2f2c7acd561518475504b9ee2f5b531a28509608de5697b7518` |
| Optional compiled probe (`build/reference-plan-third-polygon-points-probe.exe`) | `0d8c53f8103553b75788345f2602d58d07a96a98d7a96331141c12a85bb72efb` |

Drawing.cs SHA-256 remains `c07a9f255cf372cbcc2a3ecd2a31b0f1c959325749fd9155d8fd9a5649f817e7`,
Polygon.cs `63d57a5200116e9a4acaadddecec5a1381ca8173544eddafdc14d07e29237fe2`,
complete IL `ed465cef8fb81c61b37845ce7936105d70670cc8075500ac25abf9b706351a76`.
The helper read from its pinned Git object matches
`ae1f32a1034bbd95a59e97db47e887fabb30b06eabe55a960c9c5ebf9090d005`.
Original EXE/runtime, C#/IL, TOP inputs and all earlier probes/stdout remain
byte-for-byte unchanged against `build/p04c11-preservation.json`.

Actual optional commands run from the repository root:

```sh
env WINEPREFIX=/Users/dariuszlubomski/.local/share/pockettopo/wineprefix WINEDEBUG=-all MVK_CONFIG_LOG_LEVEL=0 \
  wine /Users/dariuszlubomski/.local/share/pockettopo/wineprefix/drive_c/windows/Microsoft.NET/Framework/v2.0.50727/csc.exe \
  /nologo '/out:Z:\Users\dariuszlubomski\proj\pockettopo-exporter\build\reference-plan-third-polygon-points-probe.exe' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\scripts\reference-plan-third-polygon-points-probe.cs'
env WINEPREFIX=/Users/dariuszlubomski/.local/share/pockettopo/wineprefix WINEDEBUG=-all MVK_CONFIG_LOG_LEVEL=0 \
  wine build/reference-plan-third-polygon-points-probe.exe \
  'Z:\Users\dariuszlubomski\.local\share\pockettopo\app\PocketTopoV1372\PocketTopo.exe' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\internal\top\testdata\api-drawings.top' \
  >build/native-plan-third-polygon-points-read.txt 2>build/native-plan-third-polygon-points-read.err
```

Frozen stdout is protected from text normalization; never regenerate expectations using Go.

## P04c12 third Polygon color and original scalar probe, 2026-10-10

The separate [third-color probe](../../../scripts/reference-plan-third-polygon-color-probe.cs)
uses original table/Mapping.Read methods and FileReader's BinaryReader, then
first/second Polygon scalars, following marker/count, third X/Y and one color
byte. No Drawing.Read/Polygon.Read, later marker, geometry, GUI or TOP write.
The unchanged pinned helper's third brown Polygon has raw color 3 at `[235,236)`.
Full/exact streams in pixel modes 5/10 confirm consumed 236, full-file tail 444.
Missing color throws EndOfStreamException without advancing from 235.

Independent literal streams cover zero/one/three points in each of all three
tables, with first color 129 and second color 255. Literal coordinates are
`(66051,-66051)`, `(Int32.MinValue,Int32.MaxValue)`, `(-1,0)` and
`(16777217,-16777217)`. Every third color 0..255 is read with exact and FF 80
tails; missing color is checked for every combination, including zero points.
Literal color starts are `17 + 8 * (first + second + third)`. All **13,860 exact
CRLF rows** (13,824 successful literals, 27 missing literals, six fixture reads
and three environment lines) match an independently constructed literal oracle
byte-for-byte; native exit 0, empty stderr. No Go-generated oracle or regenerated
earlier evidence.

After the analysis map, inspected Polygon.Read's ReadByte at IL_00f0 after its
point loop and matching Write color branches, including brown byte 3. Pen
selection and rendering are outside this raw reader. JKTZ's 1..7 restriction is
not adopted. Scalar evidence does not establish full native drawing/export compatibility.

Environment: macOS arm64, Wine Staging 11.7, original assembly 1.3.7.0,
Microsoft .NET x86 runtime 2.0.50727.42. Compilation produced the usable EXE
without diagnostics; only its two identified lingering compiler/start processes
were stopped after successful readback. Wine's server socket required execution
outside the sandbox. Optional tools remain separate from application/CI dependencies.

| New evidence | SHA-256 |
| --- | --- |
| `scripts/reference-plan-third-polygon-color-probe.cs` | `3632bf7584a32815a6b6ed8f66004a6313f8b072ddccf648b58b8bc03ea0de9b` |
| `native-plan-third-polygon-color-read.txt` | `713fbcc2c066fd624771065e99d9f41b9cc768b7ed147c38c9d56ca2f5325777` |
| Optional compiled probe (`build/reference-plan-third-polygon-color-probe.exe`) | `a83938a1cfaec53fa570ef9d96a5d0f095124cd412ea6517cdea2afee0221d84` |

Drawing.cs SHA-256 remains `c07a9f255cf372cbcc2a3ecd2a31b0f1c959325749fd9155d8fd9a5649f817e7`,
Polygon.cs `63d57a5200116e9a4acaadddecec5a1381ca8173544eddafdc14d07e29237fe2`,
complete IL `ed465cef8fb81c61b37845ce7936105d70670cc8075500ac25abf9b706351a76`.
The helper read from its pinned Git object matches
`ae1f32a1034bbd95a59e97db47e887fabb30b06eabe55a960c9c5ebf9090d005`.
Original EXE/runtime, C#/IL, TOP inputs and all earlier probes/stdout remain
byte-for-byte unchanged against `build/p04c12-preservation.json`.

Actual optional commands run from the repository root:

```sh
env WINEPREFIX=/Users/dariuszlubomski/.local/share/pockettopo/wineprefix WINEDEBUG=-all MVK_CONFIG_LOG_LEVEL=0 \
  wine /Users/dariuszlubomski/.local/share/pockettopo/wineprefix/drive_c/windows/Microsoft.NET/Framework/v2.0.50727/csc.exe \
  /nologo '/out:Z:\Users\dariuszlubomski\proj\pockettopo-exporter\build\reference-plan-third-polygon-color-probe.exe' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\scripts\reference-plan-third-polygon-color-probe.cs'
env WINEPREFIX=/Users/dariuszlubomski/.local/share/pockettopo/wineprefix WINEDEBUG=-all MVK_CONFIG_LOG_LEVEL=0 \
  wine build/reference-plan-third-polygon-color-probe.exe \
  'Z:\Users\dariuszlubomski\.local\share\pockettopo\app\PocketTopoV1372\PocketTopo.exe' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\internal\top\testdata\api-drawings.top' \
  >build/native-plan-third-polygon-color-read.txt 2>build/native-plan-third-polygon-color-read.err
```

Frozen stdout is protected from text normalization; never regenerate expectations using Go.

## P04c13 next plan marker scalar evidence (2026-10-10)

Original PocketTopo 1.372, existing Wine 11.7 Staging/.NET x86 2.0.50727.42.
The bounded probe reads exactly one marker after third color. Native marker 1
is at `[236,237)`; full tail 443, exact tail 0, missing marker throws at 236.
All 13,860 exact CRLF rows match independent literal expectations for all 256
markers, independent empty/one/three-point tables and exact/FF 80 tails.
Exit 0 and empty stderr. This is scalar evidence, not full drawing compatibility.

| Artifact | SHA-256 |
| --- | --- |
| `scripts/reference-plan-third-next-marker-probe.cs` | `2795a664e9853b2b21331dd3f8b55d7769e09b05dab1a83aa5deea49140dfe3c` |
| `internal/top/testdata/native-plan-third-next-marker-read.txt` | `f4f074c4dc8994b4b72d996943f452f1a3976ccab6a9c868d91ff1092c34bf59` |
| `build/reference-plan-third-next-marker-probe.exe` | `00bf2d2325b748eae72d975d71a73bfd32518f462efc3d0423975f4bc989164b` |
| `internal/top/testdata/api-drawings.top` | `4a494ead03cade750f670d50aa38661abda9b3e1f131d67f27a252982752c2a5` |
| `../pockettopo-decompilation/decompiled/csharp/PocketTopo/Drawing.cs` | `c07a9f255cf372cbcc2a3ecd2a31b0f1c959325749fd9155d8fd9a5649f817e7` |
| `../pockettopo-decompilation/decompiled/PocketTopo.il` | `ed465cef8fb81c61b37845ce7936105d70670cc8075500ac25abf9b706351a76` |

Actual optional commands run from the repository root:

```sh
env WINEPREFIX=/Users/dariuszlubomski/.local/share/pockettopo/wineprefix WINEDEBUG=-all MVK_CONFIG_LOG_LEVEL=0 \
  wine /Users/dariuszlubomski/.local/share/pockettopo/wineprefix/drive_c/windows/Microsoft.NET/Framework/v2.0.50727/csc.exe \
  /nologo '/out:Z:\Users\dariuszlubomski\proj\pockettopo-exporter\build\reference-plan-third-next-marker-probe.exe' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\scripts\reference-plan-third-next-marker-probe.cs'
env WINEPREFIX=/Users/dariuszlubomski/.local/share/pockettopo/wineprefix WINEDEBUG=-all MVK_CONFIG_LOG_LEVEL=0 \
  wine build/reference-plan-third-next-marker-probe.exe \
  'Z:\Users\dariuszlubomski\.local\share\pockettopo\app\PocketTopoV1372\PocketTopo.exe' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\internal\top\testdata\api-drawings.top' \
  >build/native-plan-third-next-marker-read.txt 2>build/native-plan-third-next-marker-read.err
```

Frozen stdout is protected from text normalization; expectations are independent of Go.

## P04c14 plan Polygon loop and original-method probe, 2026-10-10

The new [probe](../../../scripts/reference-plan-polygons-probe.cs) loads the
unchanged original PocketTopo 1.372 assembly in the same Wine/.NET environment
as earlier evidence. It executes **original Drawing.Read and Polygon.Read** on
bounded memory streams, then traverses original linked elements and reads their
stored point arrays by reflection. No GUI instance, exports or TOP files are
written. Original methods may compute their own internal bounding rectangles;
Go implements no geometry.

The pinned helper's `api-drawings.top` contains **eight Polygons / 22 points**
starting at 146. Its first XSection marker is 3 at `[370,371)`; the new Go prefix
consumes 371, leaving 309 bytes. The probe's separately labeled `bounded` harness
uses original FileReader/BinaryReader scalars and applies the requested stop
policy before unsupported payload. Full and exact inputs agree; removing that
marker throws EndOfStreamException at 370. To test original Drawing.Read's
terminal boundary, the probe copies the plan bytes and replaces marker 3 with 0;
original methods consume local offset 237 (global 371), with tails 309 / 0.
The original fixture itself remains byte-for-byte unchanged.

Independent literal streams exercise 0/1/4/17/257 Polygons, zero point counts,
signed and endian-asymmetric points, all 256 colors, all 255 stopping markers
(0 and 2..255), exact/FF 80 tails and every cut of a five-Polygon plan.
All **945 CRLF rows** match the independent literal expectations in
`TestPlanPolygonsNativeProbeTranscript` byte-for-byte; native exit 0 and empty
stderr. Expectations use pinned-helper coordinates and literal formulas, never
Go reader results. Native truncations consume available scalar bytes; Go's
preflight instead reports the established whole-points field start and returns
an empty result. Unknown markers are not passed to original Drawing.Read:
its C#/IL dispatches 3 and skips zero bytes for unknown kinds before continuing.
The safer Go unsupported stop is a deliberate reader contract, not a claim of
identical native unknown-kind handling. Color bytes remain raw in Go, while
native Polygon.Read maps unknown colors to the green pen.

| Artifact | SHA-256 |
| --- | --- |
| `scripts/reference-plan-polygons-probe.cs` | `cdb040b84ae08e844657e2799b49ecfeac68d97c8e0b8a988b81ad035feaba0d` |
| `internal/top/testdata/native-plan-polygons-read.txt` | `eabcea5717319d160f05cbdfdef9566dee965675c85047bb3c11aa6be3883949` |
| `build/reference-plan-polygons-probe.exe` | `3c2fa507bf3da4ba490908da3e07389abf765b3f5d5fb81c920f29a7caf7262a` |
| `internal/top/testdata/api-drawings.top` | `4a494ead03cade750f670d50aa38661abda9b3e1f131d67f27a252982752c2a5` |
| `../pockettopo-decompilation/decompiled/csharp/PocketTopo/Drawing.cs` | `c07a9f255cf372cbcc2a3ecd2a31b0f1c959325749fd9155d8fd9a5649f817e7` |
| `../pockettopo-decompilation/decompiled/csharp/PocketTopo/Polygon.cs` | `63d57a5200116e9a4acaadddecec5a1381ca8173544eddafdc14d07e29237fe2` |
| `../pockettopo-decompilation/decompiled/PocketTopo.il` | `ed465cef8fb81c61b37845ce7936105d70670cc8075500ac25abf9b706351a76` |

Actual optional commands (Wine needed socket access outside the sandbox):

```sh
env WINEPREFIX=/Users/dariuszlubomski/.local/share/pockettopo/wineprefix WINEDEBUG=-all MVK_CONFIG_LOG_LEVEL=0 \
  wine /Users/dariuszlubomski/.local/share/pockettopo/wineprefix/drive_c/windows/Microsoft.NET/Framework/v2.0.50727/csc.exe \
  /nologo '/out:Z:\Users\dariuszlubomski\proj\pockettopo-exporter\build\reference-plan-polygons-probe.exe' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\scripts\reference-plan-polygons-probe.cs'
env WINEPREFIX=/Users/dariuszlubomski/.local/share/pockettopo/wineprefix WINEDEBUG=-all MVK_CONFIG_LOG_LEVEL=0 \
  wine build/reference-plan-polygons-probe.exe \
  'Z:\Users\dariuszlubomski\.local\share\pockettopo\app\PocketTopoV1372\PocketTopo.exe' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\internal\top\testdata\api-drawings.top' \
  >build/native-plan-polygons-read.txt 2>build/native-plan-polygons-read.err
```

Frozen stdout is protected from text normalization. Earlier evidence is retained.
This confirms the measured loop/boundary cases, not whole-file/export compatibility.

## P04c15 complete plan XSection native evidence (2026-10-11)

After the analysis map, inspected XSection.Read/Write, Drawing.Read and ID.Read
in C# and matching IL. XSection.Read calls signed ReadInt32 at IL_0007/IL_0013,
ID.Read at IL_0028 and signed ReadInt32 at IL_0040, without direction bounds or
station lookup. Write emits marker 3, center X/Y, ID and direction. The payload
is 16 bytes; the complete source record is 17 bytes. ID.Read normalizes the
internal identity; Go retains the original UInt32 bits separately through StationID.
JKTZ's local parser confirms field order but imposes direction >= -1 and rejects
unknown markers; neither policy is adopted here.

The pinned helper's Drawings method independently authors plan sections at
(-5000,-6000), station "1", direction -1 and (5000,-6000), station "2", direction
16384. Fixture raw IDs are 0x80000002 / 0x80000003; native internal IDs are
-1048575 / -1048574. Records occupy [370,387) and [387,404); marker 0 is [404,405).
The new API reads eight Polygons / 22 points and these two XSections, consumed
405 with 275 tail bytes. P04c14 continues to stop after marker 3 at 371.

The fresh original-method probe invokes original Drawing.Read, which dispatches
original Polygon.Read and XSection.Read, on a fixture copy starting at mapping
134 and independent literal streams. It inspects original linked-list order,
center values, normalized internal IDs and signed directions. No TOP writes,
GUI, exports or station resolution. Full and exact fixture streams consume 271
relative bytes, with 275/0 tail. Missing terminator raises EndOfStreamException
at 270. Literal streams exercise 0/1/2/3/4/17/257 mixed records, exact/arbitrary
tails, signed/endian extremes, alias/reserved station IDs, negative directions
below -1, directions above 65535 and every byte cut after mapping. The original
native unknown-marker rows document its zero-payload continuation; Go's explicit
unsupported stop is independently tested and intentionally differs.

All **238 exact CRLF rows** match independently constructed literal expectations
and pinned helper fields, including unchecked C# Int32 arithmetic in the 257-record
literal. Native execution exited 0 with empty stderr. Transcript tests do not use
the Go parser or probe output to generate expected values. All earlier native
fixtures/probes are preserved byte-for-byte; no full export compatibility claim.

| Evidence | SHA-256 |
| --- | --- |
| `scripts/reference-plan-elements-probe.cs` | `a3146dc03f3af6d8e154a632364479180180a336af87e10fcb58351220033770` |
| `internal/top/testdata/native-plan-elements-read.txt` | `654b9209da1b6e9d49218c7365e08b74fc4f9547eb06f90da51da663e33dc319` |
| `internal/top/testdata/api-drawings.top` | `4a494ead03cade750f670d50aa38661abda9b3e1f131d67f27a252982752c2a5` |
| `build/pinned-pockettopo-fixtures.cs` | `ae1f32a1034bbd95a59e97db47e887fabb30b06eabe55a960c9c5ebf9090d005` |
| `../pockettopo-decompilation/decompiled/csharp/PocketTopo/XSection.cs` | `8c5a99138193290b5433c7d7470e3b4329067e7bfbe5b0ef81dffa669719d1ac` |
| `../pockettopo-decompilation/decompiled/csharp/PocketTopo/Drawing.cs` | `c07a9f255cf372cbcc2a3ecd2a31b0f1c959325749fd9155d8fd9a5649f817e7` |
| `../pockettopo-decompilation/decompiled/csharp/PocketTopo/ID.cs` | `8737d23529d5c27bc7d7be7a9563b73cd76ed8738d95639dde64eef113790fe6` |
| `../pockettopo-decompilation/decompiled/PocketTopo.il` | `ed465cef8fb81c61b37845ce7936105d70670cc8075500ac25abf9b706351a76` |
| `/Users/dariuszlubomski/.local/share/pockettopo/app/PocketTopoV1372/PocketTopo.exe` | `048cf57b874238ac142f1e4d3446af639ce3760b4a91724ea00bcc320eb5c371` |

Actual optional commands (existing Wine/runtime, repository root):

```sh
env WINEPREFIX=/Users/dariuszlubomski/.local/share/pockettopo/wineprefix WINEDEBUG=-all MVK_CONFIG_LOG_LEVEL=0 \
  wine /Users/dariuszlubomski/.local/share/pockettopo/wineprefix/drive_c/windows/Microsoft.NET/Framework/v2.0.50727/csc.exe \
  /nologo '/out:Z:\Users\dariuszlubomski\proj\pockettopo-exporter\build\reference-plan-elements-probe.exe' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\scripts\reference-plan-elements-probe.cs'
env WINEPREFIX=/Users/dariuszlubomski/.local/share/pockettopo/wineprefix WINEDEBUG=-all MVK_CONFIG_LOG_LEVEL=0 \
  wine build/reference-plan-elements-probe.exe \
  'Z:\Users\dariuszlubomski\.local\share\pockettopo\app\PocketTopoV1372\PocketTopo.exe' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\internal\top\testdata\api-drawings.top' \
  >build/native-plan-elements-read.txt 2>build/native-plan-elements-read.err
```

The first compile began while the disk was full and stalled. After stopping only
that task's compiler and recovering abandoned cache space, recompiled the probe;
its resulting assembly executed successfully with the exact frozen transcript.
Wine's local socket required sandbox escalation. Native execution is optional
for CI; the frozen independently asserted transcript is checked on every platform.

## P04c16 complete side mapping native evidence (2026-10-11)

After the analysis map, inspected DataSet.Read, Drawing.Read and Mapping.Read/Write
in C# and IL. DataSet reads overview, outline, then sideview; Drawing.Read begins
with Mapping.Read for v3. Mapping.Read calls signed ReadInt32 at IL_0008, IL_0019
and IL_002a, then divides only the stored scale by PixPerMm at IL_0034.
Go retains stored scale bits without division. JKTZ's parser confirms the layout
but restricts scale to 10..50000; this restriction is not adopted.

The new probe invokes original Drawing.Read (including mixed Polygon/XSection
records) to establish the plan terminator, then original Mapping.Read only.
It uses full/exact fixture copies and independently authored literal streams.
Both normal and VGA pixel modes (5/10) are exercised. The pinned fixtures have
side mappings [231,243) = (0,0,500) and [405,417) = (300,-400,500), leaving 5/263
bytes unparsed. Literal plans have 0/1/4/17 mixed records, distinct origins,
signed Int32 extremes, zero/negative scale and every missing side-field byte.

All **210 exact CRLF rows** match independent expectations in
`native_side_mapping_evidence_test.go`. Native execution exited 0 with empty
stderr. Native partial Int32 reads consume the available bytes before raising
EndOfStreamException; the Go reader instead reports the missing field's start
and exposes no partial result. No native unknown-marker policy is adopted:
P04c16 preserves P04c15's explicit unsupported stop without side reads.
Earlier evidence remains unchanged. No geometry or native export claim.

| Evidence | SHA-256 |
| --- | --- |
| `scripts/reference-side-mapping-probe.cs` | `b17cb37abf85ce9ec739dc38e139f10ff0a902c934f6ac3271a24a87b708ba68` |
| `internal/top/testdata/native-side-mapping-read.txt` | `50795958fb0af4a597e4eb627fa9416cf07b6627a9b04a3904882128525ec4a9` |
| `internal/top/testdata/api-references.top` | `9034cf5e52f92a8713c7823bd9be52bdb50b294966a9e9713c538b937b7feb5f` |
| `internal/top/testdata/api-drawings.top` | `4a494ead03cade750f670d50aa38661abda9b3e1f131d67f27a252982752c2a5` |
| `../pockettopo-decompilation/decompiled/csharp/PocketTopo/Mapping.cs` | `364d6d34cab642863ea92be7210afe652147ff509d9eba0cd35e319837d04e65` |
| `../pockettopo-decompilation/decompiled/csharp/PocketTopo/Drawing.cs` | `c07a9f255cf372cbcc2a3ecd2a31b0f1c959325749fd9155d8fd9a5649f817e7` |
| `../pockettopo-decompilation/decompiled/csharp/PocketTopo/DataSet.cs` | `18dc45a572a7fcd53c18eef93d5195ab3778a755485674237c07d7923a5d9940` |
| `../pockettopo-decompilation/decompiled/PocketTopo.il` | `ed465cef8fb81c61b37845ce7936105d70670cc8075500ac25abf9b706351a76` |
| `/Users/dariuszlubomski/.local/share/pockettopo/app/PocketTopoV1372/PocketTopo.exe` | `048cf57b874238ac142f1e4d3446af639ce3760b4a91724ea00bcc320eb5c371` |

Actual optional commands (repository root; existing Wine/.NET runtime):

```sh
env WINEPREFIX=/Users/dariuszlubomski/.local/share/pockettopo/wineprefix WINEDEBUG=-all MVK_CONFIG_LOG_LEVEL=0 \
  wine /Users/dariuszlubomski/.local/share/pockettopo/wineprefix/drive_c/windows/Microsoft.NET/Framework/v2.0.50727/csc.exe \
  /nologo '/out:Z:\Users\dariuszlubomski\proj\pockettopo-exporter\build\reference-side-mapping-probe.exe' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\scripts\reference-side-mapping-probe.cs'
env WINEPREFIX=/Users/dariuszlubomski/.local/share/pockettopo/wineprefix WINEDEBUG=-all MVK_CONFIG_LOG_LEVEL=0 \
  wine build/reference-side-mapping-probe.exe \
  'Z:\Users\dariuszlubomski\.local\share\pockettopo\app\PocketTopoV1372\PocketTopo.exe' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\internal\top\testdata\api-references.top' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\internal\top\testdata\api-drawings.top' \
  >build/native-side-mapping-read.txt 2>build/native-side-mapping-read.err
```

Wine execution required sandbox escalation for its local socket. The compiler
produced the executable but its Wine process remained running; stopped only that
task's compiler processes after the generated probe executed successfully. Native
execution is optional for CI; the independently asserted frozen transcript runs
everywhere.

## P04c17 first side marker native evidence (2026-10-11)

The separate `reference-side-marker-probe.cs` executes original plan
`Drawing.Read` and side `Mapping.Read`, then `BinaryReader.ReadByte`, the exact
marker operation in original `Drawing.Read`. It intentionally does not execute
side payload dispatch. C# `Drawing.cs:171-183` and IL `Drawing.Read` at
`IL_0017`/`IL_003a` establish mapping-before-marker; `DataSet.cs:138-139` reads
plan then side. The earlier JKTZ parser's `drawing` method also reads one byte
immediately after mapping, but its complete-file unknown-marker rejection is
not adopted by this prefix API.

All 775 exact CRLF rows are independently asserted in
`side_marker_native_test.go`: literal mixed plans with 0/4/17 elements, all 256
marker values, missing marker EOF, and full/exact existing fixtures. Literal
marker offset is `25 + 14*ceil(n/2) + 17*floor(n/2)`; fixtures independently place
markers at `[243,244)` (zero; tail 4) and `[417,418)` (one; tail 262).
Native probe exited 0 with empty stderr. Earlier native artifacts are unchanged.
This is method/field evidence; no geometry or native exporter claim is made.

| Artifact | SHA-256 |
| --- | --- |
| `scripts/reference-side-marker-probe.cs` | `ac896b16efe95a6f19ba5d477020a142467598a1774dfbe00e355d323a0003b1` |
| `internal/top/testdata/native-side-marker-read.txt` | `89c006cba519d00a0be06774596d4019264746b967016bdd42d93ce13bf8de21` |
| `Drawing.cs` | `c07a9f255cf372cbcc2a3ecd2a31b0f1c959325749fd9155d8fd9a5649f817e7` |
| `PocketTopo.il` | `ed465cef8fb81c61b37845ce7936105d70670cc8075500ac25abf9b706351a76` |

Original EXE/fixture hashes and Wine/.NET runtime are unchanged from P04c16.
Actual commands (repository root; existing runtime):

```sh
env WINEPREFIX=/Users/dariuszlubomski/.local/share/pockettopo/wineprefix WINEDEBUG=-all MVK_CONFIG_LOG_LEVEL=0 \
  wine /Users/dariuszlubomski/.local/share/pockettopo/wineprefix/drive_c/windows/Microsoft.NET/Framework/v2.0.50727/csc.exe \
  /nologo '/out:Z:\Users\dariuszlubomski\proj\pockettopo-exporter\build\reference-side-marker-probe.exe' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\scripts\reference-side-marker-probe.cs'
env WINEPREFIX=/Users/dariuszlubomski/.local/share/pockettopo/wineprefix WINEDEBUG=-all MVK_CONFIG_LOG_LEVEL=0 \
  wine build/reference-side-marker-probe.exe \
  'Z:\Users\dariuszlubomski\.local\share\pockettopo\app\PocketTopoV1372\PocketTopo.exe' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\internal\top\testdata\api-references.top' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\internal\top\testdata\api-drawings.top' \
  >build/native-side-marker-read.txt 2>build/native-side-marker-read.err
```

Wine required escalation for its local socket. The compiler emitted the probe
but its host process remained running; probe execution itself completed.
CI verifies the frozen transcript without Wine.

## P04d complete document

The full reader/inspect evidence adds `scripts/reference-document-probe.cs` and
`native-document-read.txt`; all earlier fixtures and native logs are unchanged.
The probe invokes **original `DataSet.Read`/`Write` and side `Drawing.Read`** via
reflection, with fresh Survey/BindingSource/Trip state per disposable input. It
never calls Load or scans template neighbors. It writes only under ignored
`build/document-probe`. The 347 CRLF stdout rows are preserved byte-for-byte;
execution returned 0 and stderr was empty. `TestDocumentNativeEvidence` constructs
all expected rows independently from the C# literal recipe and known fixture
records, then checks exact bytes; it also reads literal mixed records in Go.

| Artifact | SHA-256 |
| --- | --- |
| Original `PocketTopo.exe` | `048cf57b874238ac142f1e4d3446af639ce3760b4a91724ea00bcc320eb5c371` |
| `scripts/reference-document-probe.cs` | `5dd061d8d0b8984b48d9cb2ec436020932e11acfb88b3a1f1e493ace2762c742` |
| `native-document-read.txt` | `21bf34d971a8e2dc4f2d28aba3d27463445c0112039a3767af9cdd1e0982acb7` |
| Existing `api-references.top` | `9034cf5e52f92a8713c7823bd9be52bdb50b294966a9e9713c538b937b7feb5f` |
| Existing `api-drawings.top` | `4a494ead03cade750f670d50aa38661abda9b3e1f131d67f27a252982752c2a5` |

Actual native commands (Wine/.NET 2.0, existing reference installation):

```sh
env WINEPREFIX=/Users/dariuszlubomski/.local/share/pockettopo/wineprefix WINEDEBUG=-all MVK_CONFIG_LOG_LEVEL=0 \
  wine /Users/dariuszlubomski/.local/share/pockettopo/wineprefix/drive_c/windows/Microsoft.NET/Framework/v2.0.50727/csc.exe \
  /nologo /r:System.Windows.Forms.dll \
  '/out:Z:\Users\dariuszlubomski\proj\pockettopo-exporter\build\reference-document-probe.exe' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\scripts\reference-document-probe.cs'
env WINEPREFIX=/Users/dariuszlubomski/.local/share/pockettopo/wineprefix WINEDEBUG=-all MVK_CONFIG_LOG_LEVEL=0 \
  wine build/reference-document-probe.exe \
  'Z:\Users\dariuszlubomski\.local\share\pockettopo\app\PocketTopoV1372\PocketTopo.exe' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\build\document-probe' \
  'Z:\Users\dariuszlubomski\proj\pockettopo-exporter\internal\top\testdata' \
  > build/native-document-read.txt 2> build/native-document-read.err
```

Literal cases contain 0/1/4/17 ordered side elements, alternating empty/one/two
point Polygons and XSections with signed Int32 extremes, station raw `0x800fffff`
(native internal identity -2), direction -2, and all 256 raw color bytes. Every
side-byte cut [41,116) is rejected by original DataSet.Read; native partial field
reads consume available bytes, unlike Go's stable field-start diagnostics.
The real drawing fixture has eight Polygons (22 vertices) and two XSections in
each drawing, body end 676 and a four-byte zero trailer. The references fixture
has empty drawings, body end 244 and the same trailer.

C#/IL reviewed: DataSet.Read ends with outline/sideview Read (IL_0087–IL_00a3),
then reports true without a trailer/EOF check. DataSet.Write calls sideview.Write
and BinaryWriter.Write(Int32 zero), IL_0063–IL_0072. Drawing.Read dispatches markers
1/3, terminates on 0, and skips unknown markers with ReadBytes(0), IL_0046–IL_00bb.
Polygon.Read uses signed count, signed X/Y pairs and raw ReadByte color (native
pens interpret/normalize that byte); XSection.Read uses signed center/direction
and ID.Read. Mapping.Read divides stored scale by PixPerMm only for native state.
Go preserves stored values and makes no geometry/pen inference. Prior-art JKTZ
`parser.py` was reviewed locally: its EOF/zero trailer policy is similar, but its
validation/unknown errors are not substituted for the existing Go contracts.

Native DataSet.Read accepted absent, 1/2/3-byte zero, exact zero Int32, nonzero
Int32 and zero-plus-extra tails. The drawing stream ends at 116 in all seven
cases; tail bytes are ignored. The original writer's empty source output is 58
bytes, including final Int32 zero. Go therefore supports both EOF and exactly
four zero bytes, but preserves/classifies partial/nonzero/extra tails as
unsupported. This is an explicit policy stricter than native-reader acceptance,
not a claim that the native format requires the writer trailer during reading.
Native unknown markers 2/255 followed by zero succeed; Go preserves the marker
and stops without guessing a payload or crossing into the next drawing.

The source API, not inspection JSON, exposes original raw fields and spans.
The JSON deliberately summarizes counts and accounting without dumping vertices.
No text/geometry/export compatibility is claimed by these readback probes.
