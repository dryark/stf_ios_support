## STF IOS Support

[Actual instruction for deploy service on 2023/08/09](docs/DEPLOYMENT.md)

### Prerequisites
1. A machine running MacOS ( to build and run the "provider" )
1. A machine running Linux with Docker container support ( to run the STF server )

### Build machine setup
1. Clone this repo down to your build machine
1. Install XCode
1. Add your developer Apple ID to XCode

    1. XCode -> XCode menu -> Preferences -> Accounts Tab
    1. Click `+` under `Apple IDs` list
    1. Choose `Apple ID`
    1. Login to your account
1. Download a "Apple Development certificate" for your user

	1. Continue from previous step, right after logging into your Developer account in Xcode
    1. Select `Manage Certificates`
    1. Click `+` in the lower left corner
    1. Select `Apple Development`
1. Clone the various needed repos ( includes WebDriverAgent )

    1. Run `make clone`
1. Configure WebDriverAgent to use your identity for signing

    1. Open `repos/WebDriverAgent/WebDriverAgent.xcodeproj` in XCode
    1. Select the WebDriverAgentLib target
    1. Go to the `Signing & Capabilities` tab
    1. Select your team under `Team`
    1. Select the WebDriverAgentRunner target
    1. Go to the `Signing & Capabilities` tab
    1. Select your team under `Team`
1. Run `./init.sh`

### Deploy server side:
1. On your Linux machine
    1. Copy `server` folder to your Linux machine
    1. Run `server/cert/gencert.sh` to generate a self-signed cert ( or use your own )
    1. Note! Plain http STF server is not supported. It can be done, but it shouldn't and tickets to make it so will be closed.
	1. Update `server/.env` to reflect the IP and hostname for your server
	1. Start STF

		1. docker-compose up

### Using a standard OpenSTF server:
1. Setup your server as normal following upstream instructions
1. Create an SSL certificate for your server using the method you desire.
1. Configure the OpenSTF server for SSL
1. Alter stf_ios_support/coordinator/proc_stf_provider --connect-sub and --connect-push lines to match your server config

### Build provider files:
1. Copy the first {} block from `config.json.example` into `config.json`. Do not include any comment lines starting with //
1. Update config.json

	1. Update `xcode_dev_team_id` to be the OU of your developer account. If you add your account into Xcode first, you can then run
	   `make ou` to display what the OU is. You can also find it by opening the keychain, selecting the Apple Development certificate
	   for your account, and then looking at what the `Organization Unit` is.
	1. Update `root_path` to be where provider code should be installed, such as `/Users/user/stf`
	1. Update `config_path` to match that, such as `/Users/user/stf/config.json`
1. Run `make` then `make dist`

    1. dist.tgz will be created

### Deploy provider setup:
1. Copy `dist.tgz` from build machine
1. Run `tar -xf dist.tgz`
1. Tweak `config.json` as desired

### Starting provider
1. Register(provision) your IOS device to your developer account as a developer device.

    1. Use the API -or-
    
    	1. Follow https://developer.apple.com/documentation/appstoreconnectapi/creating_api_keys_for_app_store_connect_api to create
    	   an app store connect API key. Give it Developer access.
    	1. Gain a session using JSON Web Tokens
    	1. Create a provisioning profile if none exist using profiles: https://developer.apple.com/documentation/appstoreconnectapi/profiles
    	   See also https://github.com/cidertool/asc-go/blob/f08b8151f7fd92ff54924480338dafbf8a383255/asc/provisioning_profiles.go
    	1. Post to the devices endpoint to register a device: https://developer.apple.com/documentation/appstoreconnectapi/devices
    	   See also https://github.com/cidertool/asc-go/blob/f08b8151f7fd92ff54924480338dafbf8a383255/asc/provisioning_devices.go
    1. Follow these instructions: https://www.telerik.com/blogs/how-to-add-ios-devices-to-your-developer-profile
       I couldn't find updated instructions on Apple's website. If you find them please let me know so I can link to them.
1. Plug your IOS device in
1. Pair it with your system

	1. Run `idevicepair pair`
	1. Accept pairing on IOS device screen
1. Have Xcode setup the "developer image" on your IOS device:

    1. Open Xcode
    1. Go to Windows... Devices and Simulators
    1. Wait while Developer Image is installed to your phone
1. Run `./bin/ios_video_pull -devices -decimal` to determine the PID ( product ID ) of your IOS device in decimal
1. Run `./bin/devreset [decimal product ID] 1452` to reset the video streaming status of your IOS device
1. Run `./run` ( and leave it running )
1. Permissions dialog boxes appear for coordinator to listen on various ports; select accept for all of them
1. Device shows up in STF with video and can be controlled. Yay

### Using runner
Runner is a command that runs coordinator in a loop and also enables installation/update of a distribution.
Runner is not necessary to use stf_ios_support. It is provided to make it easier to remotely maintain a cluster
of providers.
To use it:
1. Run `make` to build all the things
1. Run `runner/runner -pass [some password]` to generate crypted password of your choice
1. Adjust `runner/runner.json`

	1. Update user password with the crypted output of previous step
	1. Update user to something else ( default user/pass are both replaceme )
	1. Update update_server to be host/IP address of the server you will use to run update_server
	1. Update updates path to be path on a provider machine where you want downloaded updates to be saved/cached
	1. Update install_dir path to be the path where you want `coordinator` to be installed
	1. Update config path to be a path where `config.json` for `coordinator` will be located on provider machine
1. Rerun `make` to rebuild `runner.tgz`
1. Run `make updatedist` to build `update_server.tgz`
1. Copy `update_server.tgz` to a server
1. Extract it

	1. `tar -xf update_server`
1. Run it and leave it running

	1. `update_server/server`
1. Copy `runner.tgz` to a provider machine
1. Extract it

	1. `tar -xf runner.tgz`
1. Stop any instance you may be running of `coordinator` already on the provider
1. Run it in a visual GUI MacOS session
1. Go to `https://[provider ip/host]:8021` in your browser
1. Accept the self-signed cert ( or make your own non-self signed cert and adjust updaet_server config )
1. Click the update button to download/install/start `coordinator` on the provider

### Known Issues
1. libimobiledevice won't install properly right now from brew

	1. `./init.sh`
    1. `make libimd` ( init.sh actually already runs this )
1. Video streaming will sometimes be left in a "stuck" state
    
    1. ios_video_pull sub-process of coordinator depends on quicktime_video_hack upstream repo/library. That library
       does not properly "stop" itself if you start and then stop reading video from an IOS device. As a result, if
       you run coordinator, stop it, then start it again, it won't be able to start up again correctly.
    1. To fix this you can use devreset. This is why the devreset command is mentioned above currently to run before
       starting coordinator. devreset effectively stops the video streaming entirely, resetting it so that it can
       be started up again.
       
### Setting up with VPN
1. Install openvpn-server on your STF server machine
1. Create client certificate(s) using your favorite process...
1. Create ovpn file(s) with those client certs
1. Deploy those cert(s) to your provider machines; setting them up in Tunnelblick
1. Alter config.json on each provider to have the name of the cert setup in Tunnelblick
1. Start openvpn server on STF server
1. Start coordinator/provider on each provider machine

### Handling video not working
1. Run `./view_log proc ios_video_pull` to check for errors fetching h264 data from the IOS device
1. Run `./view_log -proc ios_video_stream` to check for errors streaming jpegs via websocket to browser
1. Reboot your IOS device and try again

### Debugging
1. run `./view_log` to see list of things that log
1. run `./view_log -proc [one from list]`

### FAQ
See https://github.com/devicefarmer/stf_ios_support/wiki/FAQ