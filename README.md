# PDF Service

A web service for generating PDF files from HTML.  

The service is based on a fork from https://github.com/4teamwork/weasyprint-docker which uses [WeasyPrint](https://weasyprint.readthedocs.io/en/stable/index.html).

## Raw Usage

To generate a PDF make a `multipart/form-data` POST request to the service endpoint: https://pdf-service.kindsea-6f2fe326.westeurope.azurecontainerapps.io 

The request should contain the following parts:

* `html` - main HTML content
* `css` - (optional) stylesheet content
* `file.*` - (optional) additional attachments such as images. Name must start with `file.` or `attachment.`

The request must have an `Authorization Header` containing a JWT bearer with the following claims:

* `issuer`: https://login.bcc.no
* `aud`: api.bcc.no
* `scope`: pdf#read

## .Net SDK

To generate PDFs in a .Net application, add the `BccCode.PdfService.Client` nuget package.

The `IPdfServiceClient` service can be added to the applications services during startup (startup.cs or program.cs) using the following code (.net 6):

```csharp
using BccCode.PdfService.Client;

\\ ...

builder.Services.AddPdfService(new PdfServiceOptions
{
    BaseUrl = "https://pdf-service.kindsea-6f2fe326.westeurope.azurecontainerapps.io",
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

## Service Deployment

The service runs as an Azure Container App containing two containers:

1. Ingress [reverse proxy](/proxy) - .net application which handles authentication and forwards authenticated requests to the weasyprint service. This container is public on port 443.  

2. [Weasyprint server](server.py) - python web service which passes requests to the weasyprint library for processing. This container runs as a sidecar on port 8080
