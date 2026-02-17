# PDF output comparison samples

This folder contains a deterministic sample document for comparing rendering output between:

- Old approach (Python `server.py` endpoint)
- New approach (Go service endpoint)

## Files

- `sample.html`
- `sample.css`
- `logo.svg`
- `generate_and_compare.sh`

## Usage

1. Start old service so it is reachable at `http://localhost:18080/` (or set `OLD_URL`)
2. Start new service so it is reachable at `http://localhost:8080/` (or set `NEW_URL`)
3. Provide JWT for the new service in `NEW_BEARER_TOKEN`

Run:

```bash
cd samples/compare
NEW_BEARER_TOKEN="<token>" ./generate_and_compare.sh
```

Optional environment overrides:

- `OLD_URL`
- `NEW_URL`

Output is written to `samples/compare/out/`:

- `old.pdf`
- `new.pdf`

If `pdftoppm` is installed, the script also rasterizes PDFs to PNG and compares the first page image for visual equality.
