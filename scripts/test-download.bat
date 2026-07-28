@echo off
set "DlSecret=your-secret"
set "DlProxy=https://your-domain.com/downloadpath"
set "DlTarget=https://github.com/{USER}/{REPO}/releases/download/{TAG}/{FILE}"

@rem Invoke-WebRequest version
@rem powershell -NoProfile -c "&{param($u)$t=[DateTimeOffset]::UtcNow.ToUnixTimeSeconds();$c=[Text.Encoding]::UTF8;$L='%DlProxy%?url='+[Net.WebUtility]::UrlEncode($u)+'&time='+$t+'&sign='+([BitConverter]::ToString([Security.Cryptography.HMACSHA256]::new($c.GetBytes('%DlSecret%')).ComputeHash($c.GetBytes('url='+$u+'&time='+$t))) -replace '-','').ToLower();iwr $L -OutFile ([uri]$u).Segments[-1]} '%DlTarget%'"

@rem curl version , faster
powershell -NoProfile -c "&{param($u)$t=[DateTimeOffset]::UtcNow.ToUnixTimeSeconds();$c=[Text.Encoding]::UTF8;$L='%DlProxy%?url='+[Net.WebUtility]::UrlEncode($u)+'&time='+$t+'&sign='+([BitConverter]::ToString([Security.Cryptography.HMACSHA256]::new($c.GetBytes('%DlSecret%')).ComputeHash($c.GetBytes('url='+$u+'&time='+$t))) -replace '-','').ToLower();curl.exe $L -o ([uri]$u).Segments[-1]} '%DlTarget%'"

pause
