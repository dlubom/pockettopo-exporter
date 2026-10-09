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
