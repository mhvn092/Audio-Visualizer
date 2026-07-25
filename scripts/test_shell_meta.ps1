$wplPath = "$env:LOCALAPPDATA\Microsoft\Media Player\lastplayed.wpl"
if (Test-Path $wplPath) {
    [xml]$xml = Get-Content $wplPath
    $mediaNodes = @($xml.smil.body.seq.media)
    if ($mediaNodes.Count -gt 0) {
        $relPath = $mediaNodes[-1].src
        $wplFolder = "$env:LOCALAPPDATA\Microsoft\Media Player"
        $fullPath = [System.IO.Path]::GetFullPath([System.IO.Path]::Combine($wplFolder, $relPath))
        Write-Host "Resolved full path: $fullPath"
        Write-Host "File exists: $(Test-Path $fullPath)"

        if (Test-Path $fullPath) {
            $shell = New-Object -ComObject Shell.Application
            $folder = $shell.Namespace([System.IO.Path]::GetDirectoryName($fullPath))
            $file = $folder.ParseName([System.IO.Path]::GetFileName($fullPath))

            # Shell detail indices:
            # 0: Name, 13: Contributing artists, 14: Album, 21: Title
            $title = $folder.GetDetailsOf($file, 21)
            $artist = $folder.GetDetailsOf($file, 13)
            if ([string]::IsNullOrWhiteSpace($artist)) { $artist = $folder.GetDetailsOf($file, 20) }

            Write-Host "Shell Title: '$title'"
            Write-Host "Shell Artist: '$artist'"
        }
    }
}
