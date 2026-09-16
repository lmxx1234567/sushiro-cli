# Credits

## Foundational upstream: Ryujoxys/sushiro-overdose

**Thank you to [Ryujoxys](https://github.com/Ryujoxys) and the contributors to
[sushiro-overdose](https://github.com/Ryujoxys/sushiro-overdose). Their published
Sushiro China API implementation and protocol research made this smaller CLI
and MCP client possible.**

We used revision `e273df046789773616c7851c0bea14d4546f47e5`, specifically:

- [`internal/api/api.go`](https://github.com/Ryujoxys/sushiro-overdose/blob/e273df046789773616c7851c0bea14d4546f47e5/internal/api/api.go): store and timeslot requests, reservation request fields, separate query/reservation authorization, E044/E052 meaning, and nested ticket-response parsing. Our HTTP client rewrites these flows with its own errors, limits and validation.
- [`internal/core/slot.go`](https://github.com/Ryujoxys/sushiro-overdose/blob/e273df046789773616c7851c0bea14d4546f47e5/internal/core/slot.go): the reproduced `Slot` definition and adapted store/reservation models.
- [`internal/app/queue_live.go`](https://github.com/Ryujoxys/sushiro-overdose/blob/e273df046789773616c7851c0bea14d4546f47e5/internal/app/queue_live.go): store discovery, non-location coordinates, result-count default and the public-query default token. Token content is deliberately omitted from this credit document.
- [`internal/core/capture.go`](https://github.com/Ryujoxys/sushiro-overdose/blob/e273df046789773616c7851c0bea14d4546f47e5/internal/core/capture.go), [`internal/app/auth_import.go`](https://github.com/Ryujoxys/sushiro-overdose/blob/e273df046789773616c7851c0bea14d4546f47e5/internal/app/auth_import.go) and [`internal/app/mobile_auth_capture.go`](https://github.com/Ryujoxys/sushiro-overdose/blob/e273df046789773616c7851c0bea14d4546f47e5/internal/app/mobile_auth_capture.go): evidence for credential/header interpretation and the distinction between local capture guidance and native WeChat login. Our strict credential importer and bounded observer are separate implementations informed by this research.

sushiro-cli has a separate CLI/service/auth architecture, official Go SDK MCP
integration and npm launcher. It does not include upstream's desktop app,
Web UI, prediction/sampling system or Python FastMCP server. These differences
do not erase the adaptation above. Please credit and support the upstream work
when discussing this project's origins.

Upstream is MIT, copyright (c) 2026 Ryujoxys. Its complete notice is preserved in
[THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md) and
[THIRD_PARTY_LICENSES.txt](THIRD_PARTY_LICENSES.txt).

## Runtime and packaging foundations

Thanks to the Model Context Protocol Go SDK contributors, JSON Schema Go Project
Authors, Segment, Kohei YOSHIDA, and the Go Authors for the runtime dependencies
listed in [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md). The Node.js/npm tools
are used to package and launch the Go executable; Node.js itself is not bundled.

This community client is not an official Sushiro product, and no upstream author
or dependency maintainer is represented as endorsing it. The root MIT license
covers this project's own contributions by lmxx1234567, while third-party
copyright and license notices remain applicable.
