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

clean:
	rm -r ./
