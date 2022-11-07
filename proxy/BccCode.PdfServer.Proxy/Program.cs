using Microsoft.AspNetCore.Authentication.JwtBearer;
using Microsoft.AspNetCore.Authorization;
using Yarp.ReverseProxy.Configuration;

var builder = WebApplication.CreateBuilder(args);

// Proxy routes
var routes = new[]
{
    new RouteConfig()
    {
        RouteId = "route1",
        ClusterId = "cluster1",
        AuthorizationPolicy = "Default",
        Match = new RouteMatch { Path = "{**catch-all}" }
    }
};
var clusters = new[]
{
    new ClusterConfig()
    {
        ClusterId = "cluster1",
        Destinations = new Dictionary<string, DestinationConfig>(StringComparer.OrdinalIgnoreCase)
        {
            { "destination1", new DestinationConfig() { Address = "http://localhost:8080" } }
        }
    }
};

// Add services to the container.
builder.Services.AddAuthorization(c =>
{
    c.DefaultPolicy = new AuthorizationPolicyBuilder()
        .RequireClaim("scope", "pdf#create")
        .Build();
});
builder.Services.AddAuthentication(JwtBearerDefaults.AuthenticationScheme)
     .AddJwtBearer(JwtBearerDefaults.AuthenticationScheme, c =>
     {
         c.Authority = $"{builder.Configuration["Auth:Authority"]}";
         c.TokenValidationParameters = new Microsoft.IdentityModel.Tokens.TokenValidationParameters
         {
             ValidAudiences = builder.Configuration["Auth:Audiences"].Split(','),
             ValidIssuer = $"{builder.Configuration["Auth:Authority"]}".Replace("https://","").TrimEnd('/')
         };
     });

builder.Services.AddControllers();
builder.Services.AddReverseProxy().LoadFromMemory(routes, clusters);
// Learn more about configuring Swagger/OpenAPI at https://aka.ms/aspnetcore/swashbuckle
builder.Services.AddEndpointsApiExplorer();
builder.Services.AddSwaggerGen();
builder.Services.AddApplicationInsightsTelemetry(builder.Configuration["APPLICATIONINSIGHTS_CONNECTION_STRING"]);

var app = builder.Build();

// Configure the HTTP request pipeline.
if (app.Environment.IsDevelopment())
{
    app.UseSwagger();
    app.UseSwaggerUI();
}

app.UseHttpsRedirection();

app.UseAuthentication();
app.UseAuthorization();

app.MapControllers();
app.MapReverseProxy(proxy =>
{

});

app.Run();


