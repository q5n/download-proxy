# One-liner test script for download-proxy.
# Usage: .\scripts\test-download-oneliner.ps1 "https://target/url"
# Environment: DOWNLOAD_PROXY_SECRET, DOWNLOAD_PROXY_URL (optional, default: http://127.0.0.1:8001/download)
$secret = if ($env:DOWNLOAD_PROXY_SECRET) { $env:DOWNLOAD_PROXY_SECRET } else { Read-Host 'Enter secret' }; $target = $args[0]; if (-not $target) { throw 'target url required' }; $time = [int64](([DateTime]::UtcNow - (New-Object DateTime (1970,1,1,0,0,0,[DateTimeKind]::Utc))).TotalSeconds); Add-Type -AssemblyName System.Web; $enc = [System.Web.HttpUtility]::UrlEncode($target); $hmac = New-Object System.Security.Cryptography.HMACSHA256; $hmac.Key = [System.Text.Encoding]::UTF8.GetBytes($secret); $sign = ([BitConverter]::ToString($hmac.ComputeHash([System.Text.Encoding]::UTF8.GetBytes("url=$target&time=$time"))) -replace '-','').ToLower(); $url = "$(if ($env:DOWNLOAD_PROXY_URL) { $env:DOWNLOAD_PROXY_URL } else { 'http://127.0.0.1:8001/download' })?url=$enc&time=$time&sign=$sign"; Write-Host $url; curl.exe -L -o "download-$((Get-Date -Format 'yyyyMMdd-HHmmss')).tmp" $url


