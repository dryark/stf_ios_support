## STF IOS Support
### Build machine setup
1. Clone this repo down to your build machine
1. Install XCode
1. Add your developer Apple ID to XCode

    1. XCode -> XCode menu -> Preferences -> Accounts Tab
    1. `+` under `Apple IDs` list
    1. Choose `Apple ID`
    1. Login to your account so that dev certs can be downloaded
1. Run `./init.sh`

### Build server side:
1. Follow upstream docs -or-
1. On your build machine

	1. Build docker image
	    
	    1. `cd repos/stf`
	    1. `make docker`
	    1. Image is built and tagged as `stf_with_ios:1.0`
		
### Deploy server side:
1. Follow upstream STF docs -or-
1. On your STF server machine
    1. Get your built stf_with_ios image onto the server machine

		1. Via push/pull
		1. Or export/import
	1. Copy docker-compose.yml from server/docker-compose.yml
	1. Start STF

		1. docker-compose up

### Build provider files:
1. Update config.json
1. Run `make dist`

    1. offline/dist.tgz will be created

### Deploy provider setup:
1. Copy `offline/dist.tgz` from build machine
1. Run `tar -xf dist.tgz`
1. Tweak `config.json` as desired

### Starting provider
1. Run `./run` ( and leave it running )
1. Plugin one or more IOS device(s)
1. Device(s) shows up in STF with video and can be controlled. Yay

### Setting up with VPN
1. Install openvpn-server on your STF server machine
1. Create client certificate(s) using your favorite process...
1. Create ovpn file(s) with those client certs
1. Deploy those cert(s) to your provider machines; setting them up in Tunnelblick
1. Alter config.json on each provider to have the name of the cert setup in Tunnelblick
1. Start openvpn server on STF server
1. Start coordinator/provider on each provider machine

### Working around DNS issues
1. Add a host entry in OSX mapping your ios server hostname to the VPN IP address of it. ( so that you can visit it when connected to the VPN )

### Handling video not working
1. Reboot your IOS device and try again