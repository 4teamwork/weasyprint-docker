using Microsoft.Extensions.FileProviders;
using System.Net.Http;
using System.Net.Http.Headers;

namespace BccCode.PdfService.Client
{
    public class PdfServiceClient
    {
        private readonly PdfServiceOptions _options;
        private readonly IHttpClientFactory _clientFactory;
        private readonly IFileProvider _fileProvider;

        public PdfServiceClient(PdfServiceOptions options, IHttpClientFactory clientFactory, IFileProvider fileProvider)
        {
            _options = options;
            _clientFactory = clientFactory;
            _fileProvider = fileProvider;
        }

        public async Task GeneratePdfToFileAsync(string outputFilename, string html, params string[] files)
        {
            var fileInfo = _fileProvider.GetFileInfo(outputFilename);
            using (FileStream writer = File.Create(fileInfo.PhysicalPath))
            {
                var inputStream = await GeneratePdfAsync(html);
                inputStream.Position = 0;
                await inputStream.CopyToAsync(writer);
                writer.Close();
                inputStream.Close();
            }
        }


        public async Task<Stream> GeneratePdfAsync(string html, params string[] files)
        {
            var client = new HttpClient
            {
                BaseAddress = new Uri(_options.BaseUrl)
            };

            using var stream = ReadStringToStream(html);
            using var request = new HttpRequestMessage(HttpMethod.Post, "");
            using var content = new MultipartFormDataContent
            {
                { new StreamContent(stream), "html", "input.pdf" }
            };

            request.Content = content;

            var result = await client.SendAsync(request);
            return await result.Content.ReadAsStreamAsync();
        }

        // ref: https://stackoverflow.com/questions/16906711/httpclient-how-to-upload-multiple-files-at-once

        private Stream ReadStringToStream(string str)
        {
            var stream = new MemoryStream();
            var sr = new StreamWriter(stream);
            sr.Write(str);
            sr.Flush();
            stream.Position = 0;
            return stream;
        }
    }
}