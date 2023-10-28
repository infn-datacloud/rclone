@echo off

set "psCommand=powershell -Command ""$replacement = Get-Content -Raw -Path 'win-cmd.txt'; $content = Get-Content -Raw -Path 'rclone\Makefile'; $pattern = '(?s)(rclone:.*?\r?\ntest_all:)'; $editedText = $content -replace $pattern, $replacement; Set-Content -Value $editedText -Path 'rclone\Makefile';""

%psCommand%
