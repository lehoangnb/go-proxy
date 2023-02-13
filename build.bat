@echo off

rem Define the architectures
set ARCHITECTURES=386 amd64 arm arm64 mipsle mips64le

rem Define the operating systems
set OPERATING_SYSTEMS=linux

rem Define the project name
for %%F in ("%CD%") do set "PROJECT_NAME=%%~nxF"

rem Define the project directory
set PROJECT_DIR=%cd%

rem Define the output directory
set OUTPUT_DIR="%cd%\build"

rem Create the output directory if it does not exist and remove old build file
if not exist %OUTPUT_DIR% mkdir %OUTPUT_DIR%
if exist %OUTPUT_DIR% del /q %OUTPUT_DIR%\*

rem clean prj
go clean

rem Loop through the architectures
for %%a in (%ARCHITECTURES%) do (

    rem Loop through the operating systems
    for %%o in (%OPERATING_SYSTEMS%) do (

        rem Set the environment variables for the architecture and operating system
        set GOOS=%%~o
        set GOARCH=%%~a

        echo Building for %%~o %%~a

        rem Build the project
        go build -ldflags="-s -w" -o %OUTPUT_DIR%\%PROJECT_NAME%_%%o_%%a "%PROJECT_DIR%"
        if "%o%" == "windows" move %OUTPUT_DIR%\%PROJECT_NAME%_%%o_%%a %OUTPUT_DIR%\%PROJECT_NAME%_%%o_%%a.exe
    )
)
echo Build done!