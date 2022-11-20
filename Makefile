
all:
	go build


test: test001
	@echo PASS

test001:
	go build
	mkdir -p ./out ./ref
	./aes-tool --encode ./testdata/t2 --output ./out/t2.enc
	echo ""
	./aes-tool --decode ./out/t2.enc --output ./out/t2.txt
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

test002:
	./aes-tool --encode ./testdata/eek2.sh --output ./out/eek2.enc


test003:
	go build
	./aes-tool --encode ./t1/data.txt --pipe-input --output ./out/goo3.enc --password "!env!SENDGRID_API_KEY" &
	ls -l ~/ >t1/data.txt
	ls -l ~/ >t1/data.txt


