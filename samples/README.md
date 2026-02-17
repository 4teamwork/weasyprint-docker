# PDF sample generation

This folder contains a deterministic sample document for generating a PDF from the Go service.

## Files

- `sample.html`
- `sample.css`
- `logo.svg`
- `generate_sample_pdf.sh` (generates `new.pdf`)

## Usage

1. Start the Go service so it is reachable at `http://localhost:8080/` (or set `NEW_URL`)
2. Provide JWT for the service in `NEW_BEARER_TOKEN`

Run:

```bash
cd samples
NEW_BEARER_TOKEN="<token>" ./generate_sample_pdf.sh
```

Optional environment overrides:

- `NEW_URL`

Output is written to `samples/out/new.pdf`.
