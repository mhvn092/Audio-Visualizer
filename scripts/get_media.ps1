Add-Type -AssemblyName System.Runtime.WindowsRuntime

$asTaskGeneric = ([System.WindowsRuntimeSystemExtensions].GetMethods() | Where-Object { $_.Name -eq 'AsTask' -and $_.GetParameters().Count -eq 1 -and $_.GetParameters()[0].ParameterType.Name -eq 'IAsyncOperation`1' })[0]

function Await($WinRtTask, $ResultType) {
    $asTask = $asTaskGeneric.MakeGenericMethod($ResultType)
    $netTask = $asTask.Invoke($null, @($WinRtTask))
    $netTask.Wait(-1) | Out-Null
    $netTask.Result
}

# Helper function to get ID3 / Shell metadata for a file path
function Get-FileAudioMetadata($filePath) {
    if (-not (Test-Path $filePath)) { return $null }
    try {
        $fullPath = [System.IO.Path]::GetFullPath($filePath)
        $dir = [System.IO.Path]::GetDirectoryName($fullPath)
        $fileName = [System.IO.Path]::GetFileName($fullPath)

        $shell = New-Object -ComObject Shell.Application
        $folder = $shell.Namespace($dir)
        $file = $folder.ParseName($fileName)

        # Title is property index 21, Contributing artists index 13 / 20
        $title = $folder.GetDetailsOf($file, 21)
        $artist = $folder.GetDetailsOf($file, 13)
        if ([string]::IsNullOrWhiteSpace($artist)) { $artist = $folder.GetDetailsOf($file, 20) }

        if ([string]::IsNullOrWhiteSpace($title)) {
            # Fall back to cleaned filename
            $cleanName = [System.IO.Path]::GetFileNameWithoutExtension($fullPath) -replace '^\d+[\s\.\-]+', ''
            $title = $cleanName
        }

        if (-not [string]::IsNullOrWhiteSpace($title)) {
            if (-not [string]::IsNullOrWhiteSpace($artist)) {
                return "$title|$artist"
            } else {
                return "$title|Unknown Artist"
            }
        }
    } catch {}
    return $null
}

# Method 1: WinRT GSMTC (Spotify, Chrome/Edge YouTube, modern players)
try {
    [void][Windows.Media.Control.GlobalSystemMediaTransportControlsSessionManager, Windows.Media.Control, ContentType=WindowsRuntime]

    $mgr = Await ([Windows.Media.Control.GlobalSystemMediaTransportControlsSessionManager]::RequestAsync()) ([Windows.Media.Control.GlobalSystemMediaTransportControlsSessionManager])

    $session = $mgr.GetCurrentSession()
    if ($null -ne $session) {
        $props = Await ($session.TryGetMediaPropertiesAsync()) ([Windows.Media.Control.GlobalSystemMediaTransportControlsSessionMediaProperties])
        if ($null -ne $props -and $props.Title -ne '') {
            Write-Host "$($props.Title)|$($props.Artist)" -NoNewline
            exit 0
        }
    }

    $sessions = $mgr.GetSessions()
    foreach ($s in $sessions) {
        $props = Await ($s.TryGetMediaPropertiesAsync()) ([Windows.Media.Control.GlobalSystemMediaTransportControlsSessionMediaProperties])
        if ($null -ne $props -and $props.Title -ne '') {
            Write-Host "$($props.Title)|$($props.Artist)" -NoNewline
            exit 0
        }
    }
} catch {}

# Method 2: Windows Media Player Legacy — lastplayed.wpl + Shell ID3 metadata extraction
try {
    $wmpProc = Get-Process wmplayer -ErrorAction SilentlyContinue
    if ($null -ne $wmpProc) {
        $lastPlayedPath = "$env:LOCALAPPDATA\Microsoft\Media Player\lastplayed.wpl"
        if (Test-Path $lastPlayedPath) {
            [xml]$xml = Get-Content $lastPlayedPath
            $mediaNodes = @($xml.smil.body.seq.media)
            if ($null -ne $mediaNodes -and $mediaNodes.Count -gt 0) {
                $relSrc = $mediaNodes[-1].src
                if (-not [string]::IsNullOrWhiteSpace($relSrc)) {
                    $wplFolder = "$env:LOCALAPPDATA\Microsoft\Media Player"
                    $fullAudioPath = [System.IO.Path]::GetFullPath([System.IO.Path]::Combine($wplFolder, $relSrc))

                    $meta = Get-FileAudioMetadata $fullAudioPath
                    if ($null -ne $meta) {
                        Write-Host $meta -NoNewline
                        exit 0
                    }
                }
            }
        }
    }
} catch {}

# Method 3: Window title scraping (Spotify, YouTube, VLC, MusicBee, Foobar)
$procs = Get-Process | Where-Object { $_.MainWindowTitle -ne '' }
foreach ($p in $procs) {
    $name = $p.ProcessName.ToLower()
    $title = $p.MainWindowTitle

    if ($name -eq 'spotify' -and $title -ne 'Spotify' -and $title -ne 'Spotify Free' -and $title -ne 'Spotify Premium') {
        $parts = $title -split ' - ', 2
        if ($parts.Count -ge 2) {
            Write-Host "$($parts[0])|$($parts[1])" -NoNewline
            exit 0
        }
    }

    if ($name -match 'chrome|msedge|firefox') {
        if ($title -match '(.+)\s*-\s*YouTube$') {
            $ytTitle = $Matches[1].Trim()
            Write-Host "$ytTitle|YouTube" -NoNewline
            exit 0
        }
    }

    if ($name -match 'vlc|foobar|musicbee|aimp') {
        Write-Host "$title|$name" -NoNewline
        exit 0
    }
}
