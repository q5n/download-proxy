#Requires -Version 5.1

[CmdletBinding()]
param(
    [Parameter(Mandatory = $true, Position = 0)]
    [string]$TargetUrl,

    [string]$OutputPath = "download-$((Get-Date -Format 'yyyyMMdd-HHmmss')).tmp",

    [string]$ProxyUrl = "http://127.0.0.1:8001/download"
)

# 读取 secret：优先环境变量，否则交互式输入
$secret = $env:DOWNLOAD_PROXY_SECRET
if (-not $secret) {
    $secureSecret = Read-Host -Prompt "Enter secret" -AsSecureString
    $secret = [Runtime.InteropServices.Marshal]::PtrToStringAuto(
        [Runtime.InteropServices.Marshal]::SecureStringToBSTR($secureSecret)
    )
}

if (-not $secret) {
    Write-Error "secret is required"
    exit 1
}

# 过期时间：5 分钟后（兼容 PowerShell 5.1）
$unixEpoch = New-Object DateTime (1970, 1, 1, 0, 0, 0, [DateTimeKind]::Utc)
$time = [int64](([DateTime]::UtcNow - $unixEpoch).TotalSeconds)

# URL 编码 target
Add-Type -AssemblyName System.Web -ErrorAction Stop
$targetEncoded = [System.Web.HttpUtility]::UrlEncode($TargetUrl)

# HMAC-SHA256 签名
$payload = "url=${TargetUrl}&time=${time}"
Write-Host "payload: $payload"
$hmac = New-Object System.Security.Cryptography.HMACSHA256
$hmac.Key = [System.Text.Encoding]::UTF8.GetBytes($secret)
$hash = $hmac.ComputeHash([System.Text.Encoding]::UTF8.GetBytes($payload))
$sign = ([BitConverter]::ToString($hash) -replace "-", "").ToLower()

# 构造请求 URL
$url = "${ProxyUrl}?url=${targetEncoded}&time=${time}&sign=${sign}"

Write-Host "Request URL: $url"
Write-Host "Output file: $OutputPath"

# 调用 curl.exe 下载（Windows 10/11 自带）
curl.exe -L -o "$OutputPath" "$url"
