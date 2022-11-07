using Microsoft.Extensions.Configuration;
using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Threading.Tasks;

namespace BccCode.PdfService.Client.Tests
{
    public class ConfigHelper
    {
        public static IConfigurationRoot GetIConfigurationRoot(string outputPath)
        {
            return new ConfigurationBuilder()
                .SetBasePath(outputPath)
                .AddJsonFile("appsettings.json", optional: true)
                .AddUserSecrets("7631d5c1-8d69-4390-9569-4ed9124457f0")
                .Build();
        }

        public static PdfServiceOptions GetApplicationConfiguration(string outputPath)
        {
            var configuration = new PdfServiceOptions();

            var configRoot = GetIConfigurationRoot(outputPath);

            configRoot
                .GetSection("PdfService")
                .Bind(configuration);

            return configuration;
        }
    }
}
