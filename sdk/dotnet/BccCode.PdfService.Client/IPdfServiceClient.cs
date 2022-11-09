namespace BccCode.PdfService.Client
{
    public interface IPdfServiceClient
    {
        Task<Stream> GeneratePdfAsync(string html, string css = "", CancellationToken cancellationToken = default);
        Task<Stream> GeneratePdfAsync(string html, string css, IList<string>? attachmentFilenames, CancellationToken cancellationToken = default);
        Task<string> GeneratePdfToFileAsync(string outputFilename, string html, string css = "", CancellationToken cancellationToken = default);
        Task<string> GeneratePdfToFileAsync(string outputFilename, string html, string css, IList<string>? attachmentFilenames, CancellationToken cancellationToken = default);
    }
}