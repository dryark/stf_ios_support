## STF IOS Support
### Initial steps
1. Clone this repo down to your provider machine
1. Run `init.sh`

### Process for setting up server side:
1. On your provider machine

	1. Build docker image

		1. Run `make docker`
	1. Copy your docker image to your ios stf service machine

		1. Via push/pull
		1. Or export/import
1. On your stf server machine

	1. Start it

		1. docker-compose up -d
	1. Setup openvpn-server

### Process for setting up a new provider:
1. Clone this repo down to your provider machine
1. Install XCode
1. Install brew
1. brew install [local brew file]
1. Clone repos and build everything

    1. export XCODE_DEVTEAM="[your xcode development team id]"
    1. Run `make`
1. Update config.json
1. Add a host entry in OSX mapping your ios server hostname to the 192.168.255 VPN address of it. ( so that you can visit it when connected to the VPN )
1. Connect to VPN that your stf server is on ( the 192.168.255 addresses... )
1. Plugin an IOS device
1. Run coordinator ( and leave it running )
1. Run stf unit 'device-ios'
1. Device shows up in IOS STF with video and can be controlled. Yay