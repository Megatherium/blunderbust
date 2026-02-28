# Dolt Internals Reference

**MANDATORY READ BEFORE:**
- Modifying `internal/data/dolt/`
- Working on ticket store implementations
- Debugging database connection issues

---

## internal/data/dolt/ - Beads Database Access

The `dolt` package implements `data.TicketStore` for reading tickets from Beads/Dolt databases.

**Key files:**
- `metadata.go` - Parses `.beads/metadata.json` to determine connection mode
- `store_embedded.go` - Embedded Dolt driver (build tag: `embedded`, requires CGO, single-connection)
- `server.go` - MySQL driver for Dolt server connections
- `store.go` - Main `Store` type implementing `TicketStore` (build tag: `!embedded`)
- `schema.go` - Schema verification utilities

**Connection modes:**
- **Embedded**: Requires `-tags=embedded` build, uses `github.com/dolthub/driver`, local `.beads/dolt/` directory
- **Server**: Available in all builds, activated by `dolt_mode: server` in metadata.json, uses MySQL protocol

**Build modes:**
- **Default** (no tags): Server-only build (~20-30MB)
- **Full** (`-tags=embedded`): Server + Embedded modes (~93MB)

**Usage:**
```go
store, err := dolt.NewStore(ctx, domain.AppOptions{BeadsDir: ".beads"})
if err != nil {
    // Handle with actionable error message
}
defer store.Close()

tickets, err := store.ListTickets(ctx, data.TicketFilter{
    Status: "open",
    Limit: 10,
})
```

**Error handling:** All errors include context. Common patterns:
- Missing metadata.json → "Is this a beads project? Run 'bd init'"
- Missing dolt directory → "The beads database may not be initialized"
- Connection failures → Check server running / database corrupted
- Schema failures → "Try running 'bd init' to repair"
