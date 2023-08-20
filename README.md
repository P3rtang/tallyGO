# tallyGO

![alt text](https://i.imgur.com/PBqB0U1.png)

### Meant for Pokemon
This program is meant to be used to keep track of pokemon shiny hunts
It will show you the encounters, the time, the progress (%chance you would have finished already)
Other widget can show you the average time per encounter and the time for this encounter and your overall luck

### Building
To build tallyGo you need a linux Operating system, there are no plans to support macOS or windows. If you succeed in building it for those platforms you can open a pull request with the steps and updated makefile
The makefile contains the command to build
Run this command to build
```
make install
```
this will install the app, create the save file and store the needed icons on your system

### Dependencies
This Program depends on the following packages
- libgtk4-dev
- go
- make

In the future the makefile may install these packages, though *make* will always be a requirement
