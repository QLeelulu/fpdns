#!/bin/bash

version=v0.1
if [[ $# -gt 0 ]]; then
	version="$1"
fi

rm -rf dist/*
mkdir -p dist/conf/k8s
cp conf/k8s/qa-k8s.dns-conf ./dist/conf/k8s/
cp conf/mydomain.com.dns-conf ./dist/conf/
cp conf/test.dns-conf ./dist/conf/
cp conf/resolv.conf ./dist/conf/

sh build.sh

declare -a goos=(
	linux-arm
    linux-amd64
	darwin-amd64
    windows-amd64
)

for osarch in "${goos[@]}"; do
    IFS='-' #setting comma as delimiter  
    read -a osarchIN <<<"$osarch" #reading str as an array as tokens separated by IFS 
	export GOOS=${osarchIN[0]} GOARCH=${osarchIN[1]}
	echo building $GOOS-$GOARCH
    binName="fpdns_${GOOS}_${GOARCH}"
    if [ "$GOOS" = "windows" ]; then
        binName="fpdns_${GOOS}_${GOARCH}.exe"
    fi
	cp -r dist fpdns-$version
    cp bin/$binName fpdns-$version/fpdns
    rm -f fpdns-$version-$GOOS-$GOARCH.zip
	7z a fpdns-$version-$GOOS-$GOARCH.zip fpdns-$version
	rm -rf fpdns-$version
	echo
done