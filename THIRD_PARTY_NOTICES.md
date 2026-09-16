# Third-party notices for sushiro-cli

The root [MIT LICENSE](LICENSE), copyright (c) 2026 lmxx1234567, covers this
project's own contributions. It does not replace third-party licenses or claim
ownership of upstream code. This distribution contains components with different
licenses. Complete collected texts are in [THIRD_PARTY_LICENSES.txt](THIRD_PARTY_LICENSES.txt).

## Ryujoxys/sushiro-overdose — MIT

**sushiro-cli builds on the protocol research and implementation of
[Ryujoxys/sushiro-overdose](https://github.com/Ryujoxys/sushiro-overdose).**
Reference revision: `e273df046789773616c7851c0bea14d4546f47e5` (audited 2026-09-16).

The API client is a rewritten adaptation of upstream request construction,
query-versus-reservation authorization selection, business-error semantics and
nested reservation parsing. The `Slot` model is reproduced; other response
models are reduced/adapted. Store-list coordinates and result-count defaults,
and the built-in public-query token, derive from upstream `queue_live.go`.
Authentication import research also consulted upstream capture/import code.
See [CREDITS.md](CREDITS.md) and the [provenance audit](docs/provenance.md)
for specific files, changes and history. This is not a claim of completely
independent authorship or official Sushiro affiliation.

The full upstream notice below must be retained with distributed copies or
substantial portions. It is also included in the standalone license bundle.

MIT License

Copyright (c) 2026 Ryujoxys

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

## Pinned runtime dependencies

| Component | Version | License and attribution scope |
| --- | --- | --- |
| [modelcontextprotocol/go-sdk](https://github.com/modelcontextprotocol/go-sdk/blob/v1.4.0/LICENSE) | v1.4.0 | Apache-2.0 for new/relicensed contributions; original MIT remains for contributions without relicensing consent. SDK documentation excluding specifications is CC-BY-4.0. Preserve the complete transition statement and license texts. This is not an MIT-only component or a blanket choice between MIT and Apache. |
| [google/jsonschema-go](https://github.com/google/jsonschema-go/blob/v0.4.2/LICENSE) | v0.4.2 | MIT |
| [segmentio/asm](https://github.com/segmentio/asm/blob/v1.1.3/LICENSE) | v1.1.3 | MIT |
| [segmentio/encoding](https://github.com/segmentio/encoding/blob/v0.5.3/LICENSE) | v0.5.3 | Root MIT; Go-derived JSON code retains Go Authors BSD notices, covered by the included Go license. The separate Apache-2.0 `json/fuzz/LICENSE` belongs to the unlinked fuzz tool, not the runtime package. |
| [yosida95/uritemplate](https://github.com/yosida95/uritemplate/blob/v3.0.2/LICENSE) | v3.0.2 | BSD-3-Clause |
| [golang.org/x/oauth2](https://github.com/golang/oauth2/blob/v0.34.0/LICENSE) | v0.34.0 | BSD-3-Clause |
| [golang.org/x/sys](https://github.com/golang/sys/blob/v0.40.0/LICENSE) | v0.40.0 | BSD-3-Clause; additional `PATENTS` grant preserved |
| [Go runtime/standard library](https://github.com/golang/go/blob/go1.26.5/LICENSE) | go1.26.5 audited toolchain | BSD-3-Clause and `PATENTS`; bundled x/crypto, x/net, x/sys, x/text license/patent texts are included separately. The conditional boringcrypto notice is preserved with its original scope. |

The seven module versions and checksums are locked in `go.mod` / `go.sum`.
The build toolchain is separately recorded in the release build manifest; changing
it requires refreshing this inventory. No separate `NOTICE` file was found in
these seven downloaded module trees or the inspected Go dependency ancestors.
Do not remove or replace a future upstream `NOTICE`: applicable attribution
notices must travel with the distribution, including Apache-2.0 section 4(d).

## Distribution requirements

Ship `LICENSE`, `THIRD_PARTY_NOTICES.md`, `THIRD_PARTY_LICENSES.txt`, and
`CREDITS.md` with every standalone binary archive and npm package, including each
platform package and the wrapper. A link to an upstream license is not a substitute
for the required license text. Source distributions also retain the provenance
audit and existing source copyright/license notices. This four-file set is the
project's packaging rule; it includes both license obligations and our more
visible credit policy.

For Apache-covered code retain applicable source notices and mark modified
third-party files if any are changed. No SDK fork, patched module or vendored SDK
source is present in the audited baseline. SDK documentation is not copied into
our release package; importing its runtime does not relicense our own docs under
CC-BY-4.0. If SDK docs are later copied, review their attribution/license separately.

The public-query token is retained from published upstream source by explicit
project-owner choice. A software license does not establish that token's issuer,
ownership, lifetime, transferability, or permission to access Sushiro's backend.
Successful public read-only checks are operational evidence, not service
operator authorization. Upstream credit is not an endorsement by Ryujoxys,
Sushiro, the MCP project or any dependency author.
