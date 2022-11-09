using Newtonsoft.Json;
using Newtonsoft.Json.Converters;
using Newtonsoft.Json.Serialization;
using System;
using System.Collections.Concurrent;
using System.Collections.Generic;
using System.Linq;
using System.Net.Http.Headers;
using System.Text;
using System.Threading.Tasks;

namespace BccCode.PdfService.Client
{
    internal class AuthenticatedHttpRequestClient
    {
        internal readonly IHttpClientFactory _clientFactory;
        internal readonly PdfServiceOptions _options;

        internal static JsonSerializerSettings _serializerSettings;

        static AuthenticatedHttpRequestClient()
        {
            var contractResolver = new DefaultContractResolver
            {
                NamingStrategy = new CamelCaseNamingStrategy()
            };

            _serializerSettings = new JsonSerializerSettings
            {
                ContractResolver = contractResolver
            };

            _serializerSettings.Converters.Add(new StringEnumConverter
            {
                NamingStrategy = new DefaultNamingStrategy()
            });

        }
        public AuthenticatedHttpRequestClient(PdfServiceOptions options, IHttpClientFactory clientFactory)
        {
            this._clientFactory = clientFactory;
            this._options = options;
        }

        internal AuthenticatedHttpRequestClient(PdfServiceOptions options)
        {
            this._clientFactory = new TransientHttpClientFactory();
            this._options = options;
        }


        protected static ConcurrentDictionary<string, (DateTimeOffset expiry, string token)> _tokens = new ConcurrentDictionary<string, (DateTimeOffset expiry, string token)>();
        protected static ConcurrentDictionary<string, SemaphoreSlim> _semaphores = new ConcurrentDictionary<string, SemaphoreSlim>();
        protected async Task<string> GetToken()
        {
            // Retreive cached token
            var tokenKey = $"{_options.Authority}|{_options.Scope}|{_options.ClientId}|{_options.Audience}";
            if (_tokens.TryGetValue(tokenKey, out (DateTimeOffset expiry, string token) token) && token.expiry > DateTimeOffset.Now)
            {
                return token.token;
            }

            // Ensure only one token request is made at a time
            var requestLock = _semaphores.GetOrAdd(tokenKey, new SemaphoreSlim(1));
            await requestLock.WaitAsync();
            try
            {
                // Check if token has already been retreived by another thread
                if (_tokens.TryGetValue(tokenKey, out token) && token.expiry > DateTimeOffset.Now)
                {
                    return token.token;
                }

                var client = _clientFactory.CreateClient("oauth");
                var request = new HttpRequestMessage
                {
                    RequestUri = new Uri($"{_options.Authority.TrimEnd('/')}/{_options.TokenEndpoint.TrimStart('/')}"),
                    Content = new FormUrlEncodedContent(new Dictionary<string, string>
               {
                    {"grant_type", "client_credentials"},
                    {"client_id", _options.ClientId },
                    {"client_secret", _options.ClientSecret },
                    {"audience", _options.Audience },
                    {"scope", _options.Scope },
                }.ToList()),
                    Method = HttpMethod.Post
                };
                var response = await client.SendAsync(request);
                var content = await response.Content.ReadAsStringAsync();
                if (response.IsSuccessStatusCode)
                {
                    var result = JsonConvert.DeserializeAnonymousType(content, new { access_token = "", token_type = "" });
                    if (!string.IsNullOrEmpty(result?.access_token) && result.access_token.Contains('.'))
                    {
                        // Determine token expiry
                        var base64Payload = result.access_token.Split('.')[1];
                        // Add padding to base64 encoded string
                        for (var i = 0; base64Payload.Length % 4 != 0; i++)
                        {
                            base64Payload += "=";
                        }

                        var payload = JsonConvert.DeserializeAnonymousType(
                                        Encoding.UTF8.GetString(Convert.FromBase64String(base64Payload)),
                                        new { exp = 0 }
                                      );
                        if (payload != null)
                        {
                            var expiry = DateTimeOffset.FromUnixTimeSeconds(payload.exp);

                            // 10% time buffer
                            var buffer = TimeSpan.FromSeconds((expiry - DateTimeOffset.Now).TotalSeconds / 10);
                            expiry = expiry - buffer;

                            if (expiry > DateTimeOffset.Now)
                            {
                                // Save token to cache
                                _tokens[tokenKey] = (expiry, result.access_token);
                                return result.access_token;
                            }
                        }
                    }
                }
                if (response.StatusCode == System.Net.HttpStatusCode.Forbidden)
                {
                    var error = JsonConvert.DeserializeAnonymousType(content, new { error = "", error_description = "" });
                    throw new UnauthorizedException($"Failed to retreive valid access token from authentication server. {error?.error_description}");
                }
                throw new UnauthorizedException("Failed to retreive valid access token from authentication server.");
            }
            finally
            {
                requestLock.Release();
            }
        }

        protected async Task<HttpClient> CreateClient()
        {
            var client = _clientFactory.CreateClient();
            client.DefaultRequestHeaders.Clear();
            var token = await GetToken();
            client.DefaultRequestHeaders.Authorization = new AuthenticationHeaderValue("Bearer", token);
            client.DefaultRequestHeaders.Accept.Add(new MediaTypeWithQualityHeaderValue("application/json"));
            client.DefaultRequestHeaders.Add("accept-language", Thread.CurrentThread.CurrentUICulture.TwoLetterISOLanguageName);
            client.BaseAddress = new Uri(_options.BaseUrl);
            return client;
        }

        public async Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken cancellationToken = default)
        {
            var client = await CreateClient();
            var response = await client.SendAsync(request, cancellationToken);
            if (response.IsSuccessStatusCode)
            {
                return response;
            }
            else
            {
                var content = await response.Content.ReadAsStringAsync();
                throw new HttpRequestException($"Failed to retrieve data from {request.RequestUri}. Status code: {response.StatusCode}. Content: {content ?? ""}", null, response.StatusCode);
            }
        }


        internal class TransientHttpClientFactory : IHttpClientFactory
        {
            public HttpClient CreateClient(string name)
            {
                return new HttpClient();
            }
        }
    }
}
