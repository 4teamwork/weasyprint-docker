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
            var css = await File.ReadAllTextAsync("assets/style.css");
            var tasks = new List<Task>();
            for (int i = 0; i < 50; i++)
            {
                tasks.Add( client.GeneratePdfToFileAsync($"output{i}.pdf", $"<html><body><h1>TEST {i}</h1><p>Welcome</p><img src=\"test.jpg\"><img src=\"https://bcc.no/wp-content/themes/bcc-forbund/logo.svg\"></body></html>", css, "assets/test.jpg"));
            }
            await Task.WhenAll(tasks);
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