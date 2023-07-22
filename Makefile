default:
	go run main.go

install:
	go build
	install tallyGo /usr/local/bin/
	mkdir ~/.local/share/tallyGo/ -p
	touch ~/.local/share/tallyGo/save.sav
	cp ./lib/tallyGo.desktop ~/.local/share/applications/

uninstall:
	rm ~/.local/share/applications/tallyGo.desktop
	rm /usr/local/bin/tallyGo

windows:
	GOPATH=/c/go GOROOT=/mingw64/lib/go TMP=/c/tmp TEMP=/c/tmp GOARCH=amd64 && go build -buildvcs=false -ldflags='-linkmode=external' -x -v -o out/tallGo.exe

clean:
	rm -r ./
