default:
	go run main.go

install:
	go build
	sudo install tallyGo /usr/local/bin/
	mkdir ~/.local/share/tallyGo/ -p
	touch ~/.local/share/tallyGo/save.sav

	mkdir ~/.local/share/icons/hicolor/48x48/apps/ -p
	cp icons/* ~/.local/share/icons/hicolor/48x48/apps/
	gtk-update-icon-cache -f -t ~/.local/share/icons/hicolor

	cp ./lib/tallyGo.desktop ~/.local/share/applications/

uninstall:
	rm ~/.local/share/applications/tallyGo.desktop
	rm /usr/local/bin/tallyGo

clean:
	rm -r ./
