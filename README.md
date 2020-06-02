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

### Deploy server side:
1. On your STF server machine
    1. Pull STF server image `docker pull openstf/stf`
	1. Copy `docker-compose.yml` and `.env` from server/
	1. Generate certs for your system / domain
	1. Update `docker-compose.yml` cert paths and `.env`
	1. Start STF

		1. docker-compose up

### Using a standard OpenSTF server:
1. Setup your server as normal following upstream instructions
1. Alter stf_ios_support/coordinator/proc_stf_provider --connect-sub and --connect-push lines to match your server config

### Build provider files:
1. Update config.json
1. Run `make` then `make dist`

    1. dist.tgz will be created

### Deploy provider setup:
1. Copy `dist.tgz` from build machine
1. Run `tar -xf dist.tgz`
1. Tweak `config.json` as desired

### Starting provider
1. Register(provision) your IOS device to your developer account as a developer device

    1. Use the API?? https://developer.apple.com/documentation/appstoreconnectapi/devices
    1. Follow these instructions: https://www.telerik.com/blogs/how-to-add-ios-devices-to-your-developer-profile
       I couldn't find updated instructions on Apples website. If you find them please let me know so I can link to them.
1. Plug your IOS device in
1. Pair it with your system
1. Have Xcode setup the "developer image" on your IOS device:

    1. Open Xcode
    1. Go to Windows... Devices and Simulators
    1. Wait while Developer Image is installed to your phone
1. Run `./bin/ios_video_pull -devices -decimal` to determine the PID ( product ID ) of your IOS device in decimal
1. Run `./bin/devreset [decimal product ID] 1452` to reset the video streaming status of your IOS device
1. Run `./run` ( and leave it running )
1. Permissions dialog boxes appear for coordinator to listen on various ports; select accept for all of them
1. Device shows up in STF with video and can be controlled. Yay

### Known Issues
1. libimobiledevice won't install properly right now

    1. The brew version cannot be used because it is both far out of date and broken
    1. The brew --HEAD version that is installed by init.sh does not build correctly right now because HEAD is broken,
       and additionally HEAD of libimobiledevice depends on HEAD of libplist which the init.sh script is not setup
       to build and install correctly.
    1. This all will be taken care of soon. For the time being; be aware that libimobiledevice won't install via init.sh
       so you'll need to figure out on your own somehow how to get it installed until this repo is updated to automate
       that process.
    1. It is possible to build current HEAD of libimobiledevice using --disable-openssl and by brew installing HEAD
       of libplist. You'll also need to brew install libgcrypt.
1. Video streaming will sometimes be left in a "stuck" state
    
    1. ios_video_pull sub-process of coordinator depends on quicktime_video_hack upstream repo/library. That library
       does not properly "stop" itself if you start and then stop reading video from an IOS device. As a result, if
       you run coordinator, stop it, then start it again, it won't be able to start up again correctly.
    1. To fix this you can use devreset. This is why the devreset command is mentioned above currently to run before
       starting coordinator. devreset effectively stops the video streaming entirely, resetting it so that it can
       be started up again.
1. The init.sh script does not created ~/Library/LaunchAgents folder; despite it needing to exist

    1. Your system may already have such a folder for your user, in which case everything will work fine.
    1. If it doesn't exist, coordinator will be unable to create the plist within it used to start WDA. You'll need
       to create the folder then to fix this. The error message in this case is somewhat obvious; but this issue
       will bite many users. Autocreation of this folder will be added soon.
       
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
1. Run `./view_log -proc h264_to_jpeg` to check for errors decoding h264 into jpegs
1. Run `./view_log -proc ios_video_stream` to check for errors streaming jpegs via websocket to browser

1. Reboot your IOS device and try again

### Increase clicking speed
1. Jailbreak your IOS device
1. Install Veency through Cydia
1. Configure a VNC password if desired
1. Alter `config.json`

    1. Set `"use_vnc": true`
    1. Set `"vnc_scale": 2` ( or 3 depending on your device scale )
    1. If password used, set `"vnc_password": "[your password]"`
1. Start coordinator
1. Clicking is now nearly immediate!

### Debugging
1. run `./view_log` to see list of things that log
1. run `./view_log -proc [one from list]`

### FAQ
See https://github.com/tmobile/stf_ios_support/wiki/FAQ