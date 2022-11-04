using Microsoft.Extensions.FileProviders;

namespace BccCode.PdfService.Client.Tests
{
    public class PdfServiceClientTests
    {
        [Fact]
        public async Task GeneratePdfTest()
        {
            var options = new PdfServiceOptions
            {
                BaseUrl = "https://pdf-service-api.kindsea-6f2fe326.westeurope.azurecontainerapps.io"
            };
            var client = new PdfServiceClient(options, new DummyHttpClientFactory(), new PhysicalFileProvider(Directory.GetCurrentDirectory()));
            var css = await File.ReadAllTextAsync("assets/style.css");
            await client.GeneratePdfToFileAsync("output2.pdf", "<html><body><h1>TEST</h1><p>Welcome</p><img src=\"test.jpg\"></body></html>", css, "assets/test.jpg");
        }

        public class DummyHttpClientFactory : IHttpClientFactory
        {
            public HttpClient CreateClient(string name)
            {
                return new HttpClient();
            }
        }
    }
}