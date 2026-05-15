param(
    [Parameter(Mandatory=$true)][int]$pid,
    [Parameter(Mandatory=$true)][string]$src,
    [Parameter(Mandatory=$true)][string]$dst,
    [Parameter(Mandatory=$false)][string]$serviceName,
    [Parameter(ValueFromRemainingArguments=$true)][string[]]$binArgs
)

# Give it a moment to exit gracefully if it was already triggered
Start-Sleep -Seconds 1

# Kill process if it's still alive
while (Get-Process -Id $pid -ErrorAction SilentlyContinue) {
    Stop-Process -Id $pid -Force -ErrorAction SilentlyContinue
    Start-Sleep -Milliseconds 200
}

# Replace binary with retry logic
$maxRetries = 10
$retryCount = 0
while ($retryCount -lt $maxRetries) {
    try {
        Move-Item -Path $src -Destination $dst -Force -ErrorAction Stop
        break
    } catch {
        $retryCount++
        Start-Sleep -Seconds 1
    }
}

# Restart
if ($serviceName) {
    # Running as Windows Service
    Start-Service -Name $serviceName
} else {
    # Running as standalone
    if ($binArgs) {
        Start-Process -FilePath $dst -ArgumentList $binArgs
    } else {
        Start-Process -FilePath $dst
    }
}

# Remove self
Remove-Item -Path $PSCommandPath -Force -ErrorAction SilentlyContinue
