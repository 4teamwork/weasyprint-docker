using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Threading.Tasks;

namespace BccCode.PdfService.Client
{
    public struct PdfServiceScope
    {
        public static readonly PdfServiceScope Create = "pdf#create";

        public string Value { get; set; }

        public override string ToString()
        {
            return this.Value;
        }

        public static implicit operator string(PdfServiceScope scope) => scope.Value;

        public static implicit operator PdfServiceScope(string value) => new PdfServiceScope { Value = value };
    }
}
