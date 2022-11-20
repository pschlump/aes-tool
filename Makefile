
all:
	go build


.PHONEY: test setup_test test001 test002 test003
test: setup_test test001 test002 test003
	@echo PASS

setup_test:
	go build
	mkdir -p ./out ./ref ./t1 ./aes-tool-log

test001: export AES_TOOL_PASSWORD = "Humpty Dumpty"

test001: setup_test
	./aes-tool --encode ./testdata/t2 --output ./out/t2.enc --password "#env#AES_TOOL_PASSWORD" 
	echo ""
	./aes-tool --decode ./out/t2.enc --output ./out/t2.txt --password "#env#AES_TOOL_PASSWORD" 
	echo ""
	diff testdata/t2 out/t2.txt

# TODO - test with --password <fn>

install:
	rm -f ~/bin/aes-tool
	( cd ~/bin ; ln -s ../go/src/github.com/pschlump/aes-tool/aes-tool . )

linux:
	GOOS=linux GOARCH=amd64 go build -o aes-tool_linux

deploy:
	scp aes-tool_linux philip@45.79.53.54:/home/philip/tmp

test002: export AES_TOOL_PASSWORD = "Humpty Dumpty"

test002: setup_test
	./aes-tool --encode ./testdata/eek2.sh --output ./out/eek2.enc --password "#env#AES_TOOL_PASSWORD" 

test003: export AES_TOOL_PASSWORD = "Humpty Dumpty"

test003: setup_test
	./create-named-pipe.sh ./t1/data.txt
	./aes-tool --encode ./t1/data.txt --pipe-input --output ./out/goo3.enc --password "#env#AES_TOOL_PASSWORD" \
		--debug-flag "x-echo-password-env,x-dump-debug-flag" \
		&
	ls -l ./testdata | tee ./ref/save.out >t1/data.txt
	ls -l ./testdata >t1/data.txt
	ls -l ./testdata >t1/data.txt
	ls -l ./testdata >t1/data.txt
	./aes-tool --decode ./out/goo3.enc --password "#env#AES_TOOL_PASSWORD" \
		--debug-flag "x-echo-password-env,x-dump-debug-flag" \
		>t1/decrypted.out 
	cat ./ref/save.out ./ref/save.out ./ref/save.out ./ref/save.out >./ref/all.out
	diff ./ref/all.out ./t1/decrypted.out
	@echo PASS | color-cat -c green

x: export AES_TOOL_PASSWORD = "Humpty Dumpty"

x:
	go build
	./aes-tool --decode ./out/goo0.enc --password "#env#AES_TOOL_PASSWORD" >t1/decrypted.out


# Test of setting environment variables in Make

my_target: export MY_VAR_1 = foo
my_target: export MY_VAR_2 = bar
my_target: export MY_VAR_3 = baz

my_target: 
	./show-env.sh

