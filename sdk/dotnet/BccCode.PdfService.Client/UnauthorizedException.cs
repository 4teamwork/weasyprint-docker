using System.Runtime.Serialization;

namespace BccCode.PdfService.Client
{
    [Serializable]
    public class UnauthorizedException : Exception
    {
        public UnauthorizedException(string message) : base(message)
        {
        }
    }
}