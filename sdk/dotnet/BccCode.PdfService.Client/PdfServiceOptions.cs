using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Threading.Tasks;

namespace BccCode.PdfService.Client
{
    public class PdfServiceOptions
    {
        public string BaseUrl { get; set; } = "";

        public string Authority { get; set; } = "";

        public string Scope { get; set; } = PdfServiceScope.Create;

        public string TokenEndpoint { get; set; } = "/oauth/token";

        public string Audience { get; set; } = "api.bcc.no";
        
        public string ClientId { get; set; } = ""; 

        public string ClientSecret { get; set; } = "";

    }
}
