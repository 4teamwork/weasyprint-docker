using Microsoft.Extensions.FileProviders;
using System.IO;
using System.Net.Http;
using System.Net.Http.Headers;
using System.Threading;

namespace BccCode.PdfService.Client
{
    public class PdfServiceClient : IPdfServiceClient
    {
        private readonly PdfServiceOptions _options;
        private readonly AuthenticatedHttpRequestClient _client;
        private readonly IFileProvider? _fileProvider;

        public PdfServiceClient(PdfServiceOptions options, IHttpClientFactory clientFactory, IFileProvider fileProvider)
        {
            _options = options;
            _client = new AuthenticatedHttpRequestClient(options, clientFactory);
            _fileProvider = fileProvider;
        }

        public PdfServiceClient(PdfServiceOptions options, IFileProvider fileProvider)
        {
            _options = options;
            _client = new AuthenticatedHttpRequestClient(options);
            _fileProvider = fileProvider;
        }

        public PdfServiceClient(PdfServiceOptions options)
        {
            _options = options;
            _client = new AuthenticatedHttpRequestClient(options);
        }

        public Task<string> GeneratePdfToFileAsync(string outputFilename, string html, string css = "", CancellationToken cancellationToken = default)
        {
            return GeneratePdfToFileAsync(outputFilename, html, css, new string[] { }, cancellationToken);
        }

        public async Task<string> GeneratePdfToFileAsync(string outputFilename, string html, string css, IList<string>? attachmentFilenames, CancellationToken cancellationToken = default)
        {
            if (_fileProvider == null)
            {
                throw new Exception("IFileProvider service must be provided to PdfService client in order to save PDFs to file.");
            }
            else
            {

                if (outputFilename.Contains("/"))
                {
                    var directory = _fileProvider.GetFileInfo(Path.GetDirectoryName(outputFilename));
                    if (!directory.Exists)
                    {
                        Directory.CreateDirectory(directory.PhysicalPath);
                    }
                }

                var fileInfo = _fileProvider.GetFileInfo(outputFilename);

                using var inputStream = await GeneratePdfAsync(html, css, attachmentFilenames, cancellationToken);
                using (FileStream writer = File.Create(fileInfo.PhysicalPath))
                {
                    inputStream.Position = 0;
                    await inputStream.CopyToAsync(writer, cancellationToken);
                    return fileInfo.PhysicalPath;
                }
            }
        }

        public Task<Stream> GeneratePdfAsync(string html, string css = "", CancellationToken cancellationToken = default)
        {
            return GeneratePdfAsync(html, css, new string[] { }, cancellationToken);
        }

        public async Task<Stream> GeneratePdfAsync(string html, string css, IList<string>? attachmentFilenames, CancellationToken cancellationToken = default)
        {
            var attempts = 0;
        retry:
            try
            {
                using var request = new HttpRequestMessage(HttpMethod.Post, "");
                using var content = new MultipartFormDataContent();
                content.Add(new StreamContent(ReadStringToStream(html)), "html", "input.html");
                if (!string.IsNullOrEmpty(css))
                {
                    content.Add(new StreamContent(ReadStringToStream(css)), "css", "style.css");
                }
                if (attachmentFilenames?.Any() ?? false)
                {
                    if (_fileProvider == null)
                    {
                        throw new Exception("IFileProvider service must be provided to PdfService client in order to read attachments from file.");
                    }
                    else
                    {
                        foreach (var attachment in attachmentFilenames)
                        {
                            var file = _fileProvider.GetFileInfo(attachment);
                            if (file.Exists)
                            {
                                content.Add(new StreamContent(await ReadFileToStreamAsync(file)), $"attachment.{file.Name}", file.Name);
                            }
                            else
                            {
                                throw new Exception($"Attachment {file.Name} does not exist.");
                            }
                        }
                    }
                }

                request.Content = content;

                // Send request
                var result = await _client.SendAsync(request, cancellationToken);

                if (result.IsSuccessStatusCode)
                {
                    return await result.Content.ReadAsStreamAsync(cancellationToken);
                }
                else
                {
                    var errorResponse = await result.Content.ReadAsStringAsync(cancellationToken);
                    throw new Exception($"Failed to generate PDF. Service returned http status {result.StatusCode}. Content: {errorResponse ?? ""}");
                }
            }
            catch
            {
                attempts++;
                if (attempts < 5)
                {
                    await Task.Delay(1000, cancellationToken);
                    goto retry;
                }
                throw;
            }

        }

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