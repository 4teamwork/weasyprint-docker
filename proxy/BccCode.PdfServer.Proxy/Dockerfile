#See https://aka.ms/containerfastmode to understand how Visual Studio uses this Dockerfile to build your images for faster debugging.

FROM mcr.microsoft.com/dotnet/aspnet:6.0 AS base
WORKDIR /app
EXPOSE 80
EXPOSE 443
EXPOSE 5130			 

FROM mcr.microsoft.com/dotnet/sdk:6.0 AS build
WORKDIR /src
COPY ["BccCode.PdfServer.Proxy.csproj", "."]
RUN dotnet restore "./BccCode.PdfServer.Proxy.csproj"
COPY . .
WORKDIR "/src/."
RUN dotnet build "BccCode.PdfServer.Proxy.csproj" -c Release -o /app/build

FROM build AS publish
RUN dotnet publish "BccCode.PdfServer.Proxy.csproj" -c Release -o /app/publish /p:UseAppHost=false

FROM base AS final
WORKDIR /app
COPY --from=publish /app/publish .
ENTRYPOINT ["dotnet", "BccCode.PdfServer.Proxy.dll"]