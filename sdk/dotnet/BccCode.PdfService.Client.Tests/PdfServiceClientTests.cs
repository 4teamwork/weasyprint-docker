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
                BaseUrl = ""
            };
            var client = new PdfServiceClient(options, new DummyHttpClientFactory(), new PhysicalFileProvider(Directory.GetCurrentDirectory()));
            await client.GeneratePdfToFileAsync("<html><body><h1>TEST</h1><p>Welcome</p></body></html>", "output.pdf");
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