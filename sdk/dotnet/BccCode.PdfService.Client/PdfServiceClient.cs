using Microsoft.Extensions.FileProviders;
using System.IO;
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

        public async Task GeneratePdfToFileAsync(string outputFilename, string html, string css, params string[] attachments)
        {
            var fileInfo = _fileProvider.GetFileInfo(outputFilename);
            using (FileStream writer = File.Create(fileInfo.PhysicalPath))
            {
                var inputStream = await GeneratePdfAsync(html, css, attachments);
                inputStream.Position = 0;
                await inputStream.CopyToAsync(writer);
                writer.Close();
                inputStream.Close();
            }
        }


        public async Task<Stream> GeneratePdfAsync(string html, string css, params string[] attachments)
        {
            var client = new HttpClient
            {
                BaseAddress = new Uri(_options.BaseUrl)
            };

            var streams = new List<Stream>();
            using var htmlStream = ReadStringToStream(html);
            using var request = new HttpRequestMessage(HttpMethod.Post, "");
            using var content = new MultipartFormDataContent()
            {
                { new StreamContent(htmlStream), "html", "input.html" }                    
            };
            if (!string.IsNullOrEmpty(css))
            {
                var cssStream = ReadStringToStream(css);
                streams.Add(cssStream);
                content.Add(new StreamContent(cssStream), "css", "style.css");
            }
            if (attachments != null)
            {
                foreach (var attachment in attachments)
                {
                    var file = _fileProvider.GetFileInfo(attachment);
                    if (file.Exists)
                    {
                        var fileStream = await ReadFileToStreamAsync(file);
                        streams.Add(fileStream);
                        content.Add(new StreamContent(fileStream), $"attachment.{file.Name}", file.Name);
                    }
                    else
                    {
                        throw new Exception($"Attachment {file.Name} does not exist.");
                    }
                }
            }

           

            request.Content = content;

            // Send request
            var result = await client.SendAsync(request);

            // Close streams
            streams.ForEach(s =>
            {
                s.Flush();
                s.Close();
            });

            if (result.IsSuccessStatusCode)
            {
                return await result.Content.ReadAsStreamAsync();
            }
            else
            {
                var errorResponse = await result.Content.ReadAsStringAsync();
                throw new Exception($"Failed to generate PDF. Service returned http status {result.StatusCode}. Content: {errorResponse ?? ""}");
            }
        }

        // ref: https://stackoverflow.com/questions/16906711/httpclient-how-to-upload-multiple-files-at-once

        private Stream ReadStringToStream(string str)
        {
            var ms = new MemoryStream();
            var sr = new StreamWriter(ms);
            sr.Write(str);
            sr.Flush();
            ms.Position = 0;
            return ms;
        }

        private async Task<Stream> ReadFileToStreamAsync(IFileInfo file)
        {
            var ms = new MemoryStream();
            var fileStream = File.OpenRead(file.PhysicalPath);
            await fileStream.CopyToAsync(ms);
            ms.Position = 0;
            return ms;
        }
    }
}