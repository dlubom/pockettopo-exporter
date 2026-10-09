# P03b native fixture and reader evidence

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
Only header/trip values are reproduced as literal test expectations in this
project; measurements, references and drawings are outside P03b.

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
