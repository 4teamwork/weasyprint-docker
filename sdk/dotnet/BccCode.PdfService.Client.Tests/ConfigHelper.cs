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
        public static PdfServiceOptions GetApplicationConfiguration()
        {
            var configuration = new PdfServiceOptions();

            var dir = Directory.GetCurrentDirectory() + "../../../../";
            var configRoot = new ConfigurationBuilder()
                .SetBasePath(dir)
                .AddJsonFile("appsettings.json", optional: false)
                .AddUserSecrets("7631d5c1-8d69-4390-9569-4ed9124457f0")
                .Build();

            configRoot
                .GetSection("PdfService")
                .Bind(configuration);

            return configuration;
        }
    }
}
