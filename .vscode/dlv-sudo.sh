#!/bin/sh
#if ! which dlv ; then
#fi
PATH="${GOPATH}/bin:${GOROOT}/bin:$PATH"
GOFLAGS="-buildvcs=false"
DLV=$(which dlv)
if [ "$DEBUG_AS_ROOT" = "true" ]; then
    echo "Debug $DLV as root"
	exec sudo "PATH=$PATH" "GOFLAGS=$GOFLAGS" "$DLV" "$@"
else
	exec $DLV "$@"
fi

