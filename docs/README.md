---
groupSection: Products
---

# PDF Service

A web service for generating PDF files from HTML.

The service is implemented as a single Go application that validates JWT bearer tokens and renders PDFs using [WeasyPrint](https://weasyprint.org/) in a `bubblewrap` sandbox.

See [architecture.md](./architecture.md) for system overview.

## Raw Usage

To generate a PDF make a `multipart/form-data` POST request to the service endpoint: https://api.bcc.no/pdf

The request should contain the following file fields:

- `html` - main HTML file
- `css` - (optional) stylesheet file
- `attachment.*` - (optional) PDF attachments embedded into the generated document
- `file.*` - (optional) treated as attachments for backwards compatibility
- `asset.*` - (optional) additional assets (such as images) available as local files to the HTML document

The request must have an `Authorization Header` containing a JWT bearer with the following claims:

- `issuer`: https://login.bcc.no
- `aud`: api.bcc.no
- `scope`: pdf#create

## Configuration

Runtime environment variables:

- `PORT` (optional, default `8080`)
- `AUTH_AUTHORITY` (required)
- `AUTH_AUDIENCE` (required)

For local development, create `.env` from `.env.example` and run:

```bash
bash scripts/run-local.sh
```

## .Net SDK

To generate PDFs in a .Net application, add the `BccCode.PdfService.Client` nuget package.

The `IPdfServiceClient` service can be added to the applications services during startup (startup.cs or program.cs) using the following code (.net 6):

```csharp
using BccCode.PdfService.Client;

\\ ...

builder.Services.AddPdfService(new PdfServiceOptions
{
    BaseUrl = "https://api.bcc.no/pdf",
    Authority = "https://login.bcc.no",
    ClientId = "[YOUR CLIENT ID]",
    ClientSecret = "[YOUR CLIENT SECRET]",
}, (services) => new PhysicalFileProvider(Directory.GetCurrentDirectory()));

```

Note that the file provider is only required if you intend to persist the generated PDFs or need to upload attachments from storage. Any provider that implements IFileProvider may be passed to the service.

In order to use the service to generate a PDF:

```csharp
using BccCode.PdfService.Client;

\\ ...

public class PdfGenerator
{
    // PDF Service provided by application services (DI)
    public PdfGenerator(IPdfServiceClient pdfServiceClient)
    {
        _client = pdfServiceClient;
    }

    private readonly IPdfServiceClient _client;

    // Save PDF to file
    public async Task GenerateMyPdfFileAsync()
    {
        var html = "<html><body><h1>Hello world!</h1></body></html>";
        var css = @"
            @page {
                size: A4;
                margin: 0cm
            }

            body {
                background: #333333;
                color: white;
                margin: 1cm;
            }
        ";

        var outputFilePath = await _client.GeneratePdfToFileAsync("mypdf.pdf", html, css, new[] { "assets/logo.png" });

        // ...

    }

    // Get bytes from stream
    public async Task<byte[]> GenerateMyPdfInMemoryAsync()
    {
        var html = "<html><body><h1>Hello world!</h1></body></html>";
        var css = @"...";

        using var stream = await _client.GeneratePdfAsync("mypdf.pdf", html, css, new[] { "assets/logo.png" });
        using (MemoryStream ms = new MemoryStream())
        {
            await stream.CopyToAsync(ms);
            return ms.ToArray();
        }
    }
}

```
