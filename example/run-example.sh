#!/bin/bash

echo "This example will compile the hello-updater application a few times with different version strings and demonstrate go-selfupdate's functionality."
echo  "If the version is 'dev', no update checking will be performed."
echo  "I'm not sure why go-selfupdate doesn't put the files under the <appname> folder since that seems to be required for checking"
echo

rm -rf deployment/update deployment/hello* public/hello-updater

echo "Building example-server"; echo
go build -o example-server src/example-server/main.go

echo "Running example server"; echo
killall example-server
./example-server &

read -n 1 -p "Press any key to start." ignored; echo

echo "Building dev version of hello-updater"; echo
go build -ldflags="-X main.version=dev" -o hello-updater src/hello-updater/main.go

echo "Copying it to deployment folder"; echo
cp hello-updater deployment/


echo "Running deployment/hello-updater"
deployment/hello-updater
read -n 1 -p "Press any key to continue." ignored; echo
echo; echo "=========="; echo

for (( minor=0; minor<=2; minor++ )); do
    echo "Building hello-updater with version set to 1.$minor"
    go build -ldflags="-X main.version=1.$minor" -o hello-updater src/hello-updater/main.go

    echo "Running go-update to make update available via example-server"; echo
    go-selfupdate -o public/hello-updater/ hello-updater 1.$minor

    if (( $minor == 0 )); then
        echo "Copying version 1.0 to deployment so it can self-update"; echo
        cp hello-updater deployment/
        cp hello-updater deployment/hello-updater-1.0
    fi

    echo "Running deployment/hello-updater"
    deployment/hello-updater
    read -n 1 -p "Press any key to continue." ignored; echo
    echo; echo "=========="; echo
done

echo "Running deployment/hello-updater-1.0 backup copy"
deployment/hello-updater-1.0
read -n 1 -p "Press any key to continue." ignored; echo
echo; echo "=========="; echo

echo "Building unknown version of hello-updater"; echo
go build -ldflags="-X main.version=unknown" -o hello-updater src/hello-updater/main.go
echo "Copying unknown version to deployment so it can self-update"; echo
cp hello-updater deployment/

echo "Running deployment/hello-updater"
deployment/hello-updater
sleep 5
echo; echo "Re-running deployment/hello-updater"
deployment/hello-updater
sleep 5
echo; echo

echo "Shutting down example-server"
killall example-server
