using BccCode.PdfService.Client;
using Microsoft.Extensions.FileProviders;
using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Threading.Tasks;

namespace Microsoft.Extensions.DependencyInjection
{
    public static class PdfServiceInstaller
    {
        public static IServiceCollection AddPdfService(this IServiceCollection services, PdfServiceOptions options)
        {
            return services.AddPdfService(options, null);
        }
        public static IServiceCollection AddPdfService(this IServiceCollection services, PdfServiceOptions options, IFileProvider fileProvider)
        {
            if (!services.Any(x => x.ServiceType == typeof(IHttpClientFactory)))
            {
                services.AddHttpClient();
            }

            return services.AddSingleton<IPdfServiceClient>(x =>
            {
                var clientFactory = x.GetRequiredService<IHttpClientFactory>();
                var fp = fileProvider ?? x.GetService<IFileProvider>();
                return fp != null ? new PdfServiceClient(options, fp) : new PdfServiceClient(options);
            });
        }

    }
}
