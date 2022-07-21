$version = $args[0]

#update verification.txt and chocolateyInstall.ps1 files
$folder = "tools"
$files_to_update = "$folder\VERIFICATION.txt", "$folder\chocolateyInstall.ps1"

foreach ($file in $files_to_update) {
    $script_path = "$PSScriptRoot\$file"
    $content    = Get-Content $script_path -Raw  

    write-host $version 

    Write-Host "Updating $file file with version: $version"
    $update_version = $content -replace "\d+\.\d+\.\d+", "$version"
    Set-Content -Path $script_path -Value $update_version
}

#update testkube.nuspec
$file = "testkube.nuspec"
$file_content  = Get-Content $file -Raw  

Write-Host "Updating $file file with version: $version"
$update_version = $file_content -replace ">\d+\.\d+\.\d+<", ">$version<"
Set-Content -Path $file -Value $update_version

#package with chocolatey
choco pack

#push package
choco push .\Testkube.$version.nupkg --source https://push.chocolatey.org