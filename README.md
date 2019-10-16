## STF IOS Support
### Initial steps
1. Clone this repo down to your provider machine
1. Install XCode
1. Install certs for your XCode Developer to sign

    1. Double click your cert files to install them into the cert store
1. Add your developer Apple ID to XCode

    1. XCode -> XCode menu -> Preferences -> Accounts Tab
    1. `+` under `Apple IDs` list
    1. Choose `Apple ID`
    1. ...
1. Run `./init.sh`

### Server side setup:
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

### Provider setup:
1. Update config.json
1. Run `make`

### Starting provider
1. Connect to VPN that your stf server is on
1. Plugin an IOS device
1. Run `./run` ( and leave it running )
1. Device shows up in IOS STF with video and can be controlled. Yay

### Additional provider setup:
1. Run `make dist`
1. Copy `offline/dist.tgz` to a new provider
1. On new provider...
1. Run `tar -xf dist.tgz`

### Working around DNS issues
1. Add a host entry in OSX mapping your ios server hostname to the VPN IP address of it. ( so that you can visit it when connected to the VPN )
