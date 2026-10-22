# Contract fixtures

Keep service contracts under `go/cmd/<service>/testdata/`, independently extracted from the official API reference. Use `contracts.json` for one source, or source-named files for regional variants.

| Field | Purpose |
| --- | --- |
| `schema_version` | Fixture format version |
| `source` | Official URL, title/version, retrieval date, and source-byte SHA-256 |
| `region`, `base_url`, `authentication` | Published endpoint and authentication scope |
| `operations[].command`, `operation_id` | CLI mapping and official operation identifier |
| `method`, `path`, `parameters`, `request`, `response` | Request/response facts used by tests |
| `mutation`, `destructive` | Reviewed CLI safety classification |

Record provenance once per source. Schemas may add source-specific facts; preserve API names, requiredness, types, bounds, and transport details used by tests. Do not infer defaults or constraints absent from the reference.

Tests validate fixture metadata and inventory, then compare commands and local-server requests against fixture facts—not against expectations derived from implementation descriptors. Request/response examples use synthetic `fixture-*` identifiers and credentials. These examples test serialization and handling, not provider-side business acceptance.

Never include real accounts, resource IDs, tokens, private catalogs, or workstation paths. Review official source changes before updating fixtures. Mocked tests do not establish live provider support.

`go/cmd/testdata/upstream_commands.json` separately records pre-integration command paths and flags. Its additive compatibility test protects existing vServer, AgentBase, login, logout, and configuration entry points; it does not establish wire or output compatibility.
