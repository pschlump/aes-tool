#!/bin/bash

# Call
# Info: command to run [./bin/backup-log-files.sh] args ----->["./t1/logfile-rtate-current.txt.enc","./t1/logfile.2022-11-22T18:22:22-07:00.log.enc"]<----- at:File: /Users/philip/go/src/github.com/pschlump/aes-tool/runCmd.go LineNo:12

# out, err := RunCmdImpl(*BackupScript, []string{*output, newFn})
#
# $1 - current file name with relative path.
# $2 - new file name with relative path. 

echo "" >>/tmp/,v
echo "" >>/tmp/,v
echo "Date Run: $(date)" >>/tmp/,v
echo "Params: $*" >>/tmp/,v

# 1. TODO
# Copy files to S3 - most recent $2
echo /usr/local/bin/s3cmd put "$2" s3://tcs-docs >>/tmp/,v

# 2. TODO
# Remove old files - look at howmany match in "save" directory
# sorty by oldest to newest.
# loop over and get rid of all but most recent.
mkdir -p ./log-backed-up 

echo rm -f $( ls -tro ./log-backed-up | sed '$d' )  >>/tmp/,v

echo mv "$2" ./log-backed-up >>/tmp/,v

