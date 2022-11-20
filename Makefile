
all:
	go build


test: test001 test002 test003
	@echo PASS

test001: export AES_TOOL_PASSWORD = "Humpty Dumpty"

test001:
	go build
	mkdir -p ./out ./ref
	./aes-tool --encode ./testdata/t2 --output ./out/t2.enc --password "!env!AES_TOOL_PASSWORD" 
	echo ""
	./aes-tool --decode ./out/t2.enc --output ./out/t2.txt --password "!env!AES_TOOL_PASSWORD" 
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

test002:
	./aes-tool --encode ./testdata/eek2.sh --output ./out/eek2.enc --password "!env!AES_TOOL_PASSWORD" 

test003:
	go build
	mkdir -p ./out ./ref
	./aes-tool --encode ./t1/data.txt --pipe-input --output ./out/goo3.enc --password "!env!AES_TOOL_PASSWORD" &
	ls -l ./testdata >t1/data.txt
	ls -l ./testdata >t1/data.txt
	ls -l ./testdata >t1/data.txt
	ls -l ./testdata >t1/data.txt



# Test of setting environment variables in Make

my_target: export MY_VAR_1 = foo
my_target: export MY_VAR_2 = bar
my_target: export MY_VAR_3 = baz

my_target: 
	./show-env.sh

