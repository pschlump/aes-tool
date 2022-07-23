
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

