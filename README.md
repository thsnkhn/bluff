# Bluff

A private poker ledger, right in your terminal.

Bluff keeps player standings, games, and results together so the table always
adds up.

## Install

With Homebrew:

```bash
brew install thsnkhn/tap/bluff
```

Or install the latest version from source:

```bash
go install github.com/thsnkhn/bluff/cmd/bluff@latest
```

## Use

```bash
bluff
```

- `r` refresh
- `l` log out
- `q` quit

## Development

```bash
go run ./cmd/bluff
```

## License

Bluff is licensed under the [GNU General Public License v3.0](LICENSE).
