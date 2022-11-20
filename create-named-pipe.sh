#!/bin/bash

if ls -l "$1" >/dev/null ; then
	if ls -l "$1" | grep '^p' ; >/dev/null then
		:
	else
		echo "not pipe"
		echo "rm -f \"$1\""
		echo mkfifo "$1"
	fi
else
	echo "not found"
	echo mkfifo "$1"
fi



