## STF IOS Support
### Process for setting up a new provider:
1. Clone this repo down to your provider machine
2. Install XCode...
3. Run init.sh to clone down the other needed repos
4. Build STF
	5. In repos/stf
	6. bower install
	7. npm install
	8. npm link
5. Build WebDriverAgent
	6. In repos/WebDriverAgent
	7. carthage update ( to install dependencies )
	8. Open WebDriverAgent.xcodeproj in Xcode
	9. Change WebDriverAgentLib and WebDriverAgentRunner "Signing/Team" to be your development team
	10. Build/Install it against your IOS device
	11. Update path to repos/WebDriverAgent in coordinator.go
6. Build mirrorfeed
	7. In repos/stf_ios_mirrorfeed
	8. Install golang if not already installed
	9. go build mirrorfeed.go
		10. May have to 'go get [dependency]' for deps 
	10. Update path to exe for mirrorfeed in coordinator.go 
7. Build osx_ios_device_trigger
	8. In repos/osx_ios_device_trigger
	9. Open osx_ios_device_trigger.xcodeproj in Xcode
	10. Build it
	11. Update path to build executable in coordinator.go 
		12. This can be determined easily by right clicking the built exe in xcode to show in finder, then right clicking the file in finder, then holding old to show 'copy as pathname'
8. Build osx_ios_video_enabler
	9. In repos/osx_ios_video_enabler
	10. Open osx_ios_video_enabler.xcodeproj in Xcode
	11. Build it
	12. Update path to build exe in coordinator.go	 
9. Build coordinator
	10. In root of stf_ios_support checkout
	11. go build coordinator.go 
		12. May have to 'go get [dependency]' for deps
10. Update config.sh to have your developer id and the path to stf_ios_support/repos/stf
11. Update run-stf.sh with the hostname of your ios stf server
12. Update "fakes.xml" in the iosfake folder on your stf server with the output of the various idevice* commands when run locally for your phone. For testing purposes you can mostly just copy an existing device and change the UUID to match that of your IOS devices.
13. Add a host entry in OSX mapping your ios server hostname to the 192.168.255 VPN address of it. ( so that you can visit it when connected to the VPN )
14. Connect to VPN that your stf server is on ( the 192.168.255 addresses... )
15. Run coordinator ( and leave it running )
16. Plugin an IOS device
17. Device shows up in IOS STF with video and can be controlled. Yay

