@echo off
set "DlSecret=your-secret"
set "DlProxy=https://your-domain.com/downloadpath"
set "DlTarget=https://github.com/{USER}/{REPO}/releases/download/{TAG}/{FILE}"

powershell -NoProfile -c "&{param($u)$t=[DateTimeOffset]::UtcNow.ToUnixTimeSeconds();$L='%DlProxy%?url='+[Net.WebUtility]::UrlEncode($u)+'&time='+$t+'&sign='+([BitConverter]::ToString([Security.Cryptography.HMACSHA256]::new([Text.Encoding]::UTF8.GetBytes('%DlSecret%')).ComputeHash([Text.Encoding]::UTF8.GetBytes('url='+$u+'&time='+$t))) -replace '-','').ToLower();iwr $L -OutFile ([uri]$u).Segments[-1]} '%DlTarget%'"

pause
