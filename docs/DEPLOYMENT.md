# Deployment

## Relevance
This instruction actual on 2023/08/09

## Tested working configuration
- PC: MacMini 2018 (x86_64)
- OS: MacOS Ventura 13.5
- IDE: XCode 14.3.1
- Mobile devices: iPhone and iPad different years with iOS versions: 14.1, 14.2, 14.6, 14.7, 15.0, 15.4.1, 16.6

## 1. Prepare PC (build machine)
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
1. Install [Homebrew](https://docs.brew.sh/Installation)
1. Install Python 2.7 from MacPorts
    1. [Install MacPorts](https://www.macports.org/install.php)
    1. In terminal: `ports install python27`

## 2. Prepare server STF
1. Add in NGINX config (in STF server deployment) block configuration for new provider:
- Client IP should be changed from `[^/]` to some more specific range such as: `(?<client_ip>192.168.255.[0-9]+)` to restrict it to a reasonable IP range
- If left alone this example config will let clients arbitrarily tunnel to any IP on ports `8000-8009`
```
# MacOS
location ~ "^/frames/(?<client_ip>[^/]+)/(?<client_port>800[0-9])/x$" {
  proxy_pass http://$client_ip:$client_port/echo/;
  proxy_http_version 1.1;
  proxy_set_header Upgrade $http_upgrade;
  proxy_set_header Connection $connection_upgrade;
  proxy_set_header X-Forwarded-For $remote_addr;
  proxy_set_header X-Real-IP $remote_addr;
}
```

## 3. Prepare mobile devices for work with provider
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
1. Plug in your IOS device to PC
1. Accept pairing on IOS device screen
1. Access device on PC:

    1. Open `Finder`
    2. Click to device
    3. Accept pairing to device
1. Have Xcode setup the "developer image" on your IOS device:

    1. Open Xcode
    1. Go to Windows... Devices and Simulators
    1. Wait while Developer Image is installed to your phone
1. Download actual profiles for device by Xcode or TestFlight:
https://docs.fastlane.tools/actions/sigh/

## 4. Prepare dependenses on build machine
1. Run `./init.sh`
1. View error:
```bash
mobiledevice        => version 2.0.0
libplist - Installing HEAD
Running `brew update --auto-update`...
==> Auto-updated Homebrew!
Updated 2 taps (homebrew/core and homebrew/cask).
 
You have 32 outdated formulae installed.
You can upgrade them with brew upgrade
or list them with brew outdated.
 
Cloning into '/Users/emb/Library/Caches/Homebrew/libplist--git'...
Already on 'master'
/usr/local/lib/pkgconfig/libplist.pc was missing; creating symlink to /usr/local/Cellar/libplist/HEAD-c3af449/lib/pkgconfig//libplist-2.0.pc
ln: /usr/local/lib/pkgconfig/libplist.pc: File exists
libusbmuxd - Installing HEAD
Already on 'master'
Please create pull requests instead of asking for help on Homebrew's GitHub,
Twitter or any other official channels.
Could not fix pkgconfig for libusbmuxd; could not locate installed pc file in Cellar
```
1. Need manual install needed [libs](https://github.com/libimobiledevice), run: `brew install --HEAD libusbmuxd`
1. And view new error:
```bash
==> Cloning https://github.com/libimobiledevice/libusbmuxd.git
Cloning into '/Users/emb/Library/Caches/Homebrew/libusbmuxd--git'...
==> Checking out branch master
Already on 'master'
Your branch is up to date with 'origin/master'.
==> ./autogen.sh
Last 15 lines from /Users/emb/Library/Logs/Homebrew/libusbmuxd/01.autogen.sh:
checking whether to build static libraries... yes
checking for pkg-config... /usr/local/Homebrew/Library/Homebrew/shims/mac/super/pkg-config
checking pkg-config is at least version 0.9.0... yes
checking for libplist-2.0 >= 2.2.0... yes
checking for libimobiledevice-glue-1.0 >= 1.0.0... no
configure: error: Package requirements (libimobiledevice-glue-1.0 >= 1.0.0) were not met:
 
No package 'libimobiledevice-glue-1.0' found
 
Consider adjusting the PKG_CONFIG_PATH environment variable if you
installed software in a non-standard prefix.
 
Alternatively, you may set the environment variables limd_glue_CFLAGS
and limd_glue_LIBS to avoid the need to call pkg-config.
See the pkg-config man page for more details.
 
Do not report this issue to Homebrew/brew or Homebrew/core!
 
Please create pull requests instead of asking for help on Homebrew's GitHub,
Twitter or any other official channels.
```

This error initial by new dependenses: https://github.com/libimobiledevice/libimobiledevice-glue

In Homebrew no formulae for it: https://github.com/Homebrew/homebrew-core/pull/87059

1. Resolve by: https://github.com/libimobiledevice/libimobiledevice/issues/1217

    1. `brew create --set-name libimobiledevice-glue-1.0 "https://github.com/libimobiledevice/libimobiledevice-glue.git"`
    1. Save file without changes
    1. Edit file: `brew edit libimobiledevice-glue-1.0`

    It text needed:
    ```text
    class LibimobiledeviceGlue10 < Formula
      desc ""
      homepage ""
      url "https://github.com/libimobiledevice/libimobiledevice-glue.git"
      head "https://github.com/libimobiledevice/libimobiledevice-glue.git"
      version "1.0.0"
      sha256 ""
      license ""
    
      depends_on "autoconf" => :build
      depends_on "automake" => :build
      depends_on "libtool" => :build
      depends_on "pkg-config" => :build
      depends_on "libplist"
    
      def install
        system "./autogen.sh"
        system "./configure", "--disable-dependency-tracking", "--disable-silent-rules", "--prefix=#{prefix}"
        system "make", "install"
      end
    
      test do
        system "false"
      end
    end
    ```
    1. Install library: `HOMEBREW_NO_INSTALL_FROM_API=1 HOMEBREW_NO_INSTALL_CLEANUP=1 brew install --HEAD libimobiledevice-glue-1.0`
    1. Manual fix symlinks:
    ```bash
    ./util/brewser.pl fixpc libimobiledevice-glue-1.0 1.0.0
 
    cd /usr/local/lib/pkgconfig/
    rm libimobiledevice-glue-1.0.pc libimobiledevice-glue-1.0-1.0.0.pc
    ln -s ../../Cellar/libimobiledevice-glue-1.0/HEAD-<HASH>/lib/pkgconfig/libimobiledevice-glue-1.0.pc libimobiledevice-glue-1.0.pc
    ln -s ../../Cellar/libimobiledevice-glue-1.0/HEAD-<HASH>/lib/pkgconfig/libimobiledevice-glue-1.0.pc libimobiledevice-glue-1.0-1.0.0.pc
    ls -al libimobiledevice-glue-1.0*
    ```
    1. Fix env: `export PKG_CONFIG_PATH=$(find /usr/local/Cellar -name 'pkgconfig' -type d | grep lib/pkgconfig | tr '\n' ':' | sed s/.$//)`
    1. Fix formulae: `brew edit libusbmuxd`
    1. Add in file new dependens:
    ```text
    depends_on "libimobiledevice-glue-1.0"
    ```
    1. Save file
    1. Install libs:
    ```bash
    HOMEBREW_NO_INSTALL_FROM_API=1 HOMEBREW_NO_INSTALL_CLEANUP=1 brew install --HEAD libusbmuxd
    HOMEBREW_NO_INSTALL_FROM_API=1 HOMEBREW_NO_INSTALL_CLEANUP=1 brew install --HEAD libimobiledevice
    ./util/brewser.pl fixpc libusbmuxd 2.0
    ./util/brewser.pl fixpc libimobiledevice 1.0
    ```

## 5. Prepare sources
1. Clone the various needed repos

    1. Run `make clone`

1. Configure WebDriverAgent to use your identity for signing

    1. Open `repos/WebDriverAgent/WebDriverAgent.xcodeproj` in XCode
    1. Select the ***WebDriverAgentLib*** target
    1. Go to the `Signing & Capabilities` tab
    1. Select your team under `Team`
    1. Change ***Bundle Identifier*** to your
    1. Repeate this action for ***WebDriverAgentRunner***, ***UnitTests***, ***IntegrationTests_[1:3]*** and ***IntegrationApps***
    1. Select in top target for builder: ***WebDriverAgentRunner***
    1. Select in top physical device for builder
    1. Test build and run test: select in XCode menu `Product` and `Test`
    1. In log chech what start server and success
    1. Close XCode

1. Create config:

    1. Copy the first {} block from `config.json.example` into `config.json`. Do not include any comment lines starting with //
    1. Edit config.json
	    1. Update `xcode_dev_team_id` to be the OU of your developer account. If you add your account into Xcode first, you can then run
	   `make ou` to display what the OU is. You can also find it by opening the keychain, selecting the Apple Development certificate
	   for your account, and then looking at what the `Organization Unit` is.
	    1. Update `root_path` to be where provider code should be installed, such as `/Users/user/stf`
	    1. Update `config_path` to match that, such as `/Users/user/stf/config.json`
        1. Set `use_vnc` to `false` (https://githubhelp.com/devicefarmer/stf_ios_support/issues/37)
    1. Sample:
    ```json
    {
      "xcode_dev_team_id": "XXXXXXXXX",
      "stf": {
        "ip": "192.168.1.10",
        "hostname": "stf.example.com"
      },
      "video": {
        "enabled": true,
        "use_vnc": false,
        "vnc_scale": 2,
        "vnc_password": "",
        "frame_rate": 10
      },
      "install": {
        "root_path": "/Users/user/stf",
        "config_path": "/Users/user/stf/config.json",
        "set_working_dir": false
      },
      "bin_paths": {
        "video_enabler": "bin/video_enabler"
      }
    }
    ```

1. Fix [error](https://github.com/DeviceFarmer/stf_ios_support/issues/100) in repos `ios_video_pull`

    1. Edit file `repos/ios_video_pull/go.mod`
    2. New text file:
    ```text
    module github.com/nanoscopic/ios_video_pull
 
    go 1.14
 
    require (
            github.com/danielpaulus/quicktime_video_hack v0.0.0-20200514194616-c4570b6b687c
            github.com/google/gousb v0.0.0-20190812193832-18f4c1d8a750 // <--
            github.com/nanomsg/mangos v2.0.0+incompatible
            github.com/sirupsen/logrus v1.6.0
            go.nanomsg.org/mangos/v3 v3.0.1
            nanomsg.org/go/mangos/v2 v2.0.8 // indirect
    )
    ```

## 6. Build and package application
1. Fix env before build:
```bash
export PKG_CONFIG_PATH=$(find /usr/local/Cellar -name 'pkgconfig' -type d | grep lib/pkgconfig | tr '\n' ':' | sed s/.$//)
```
1. Fix [error](https://juejin.cn/post/6985818094794965006#heading-6) repos `wdaproxy`:
```bash
cd repos/wdaproxy
go get github.com/DHowett/go-plist@v0.0.0-20170330020707-795cf23fd27f
go install github.com/DHowett/go-plist@v0.0.0-20170330020707-795cf23fd27f
go mod tidy
```
1. Build: `make`
1. Run then `make dist`

    1. ***dist.tgz*** will be created

## 7. Deploy provider setup:
1. Copy `dist.tgz` from build machine
1. Run `tar -xf dist.tgz`
1. Tweak `config.json` as desired

## 8. Starting provider
1. Check plugging devices: `idevice_id`
1. Pair its with your system

	1. Run `idevicepair pair`
	1. Accept pairing on IOS device screen
1. Activate video by devices

    1. Activate screen devices
    1. Run ***QuickTime Player*** in GUI Mac
    1. `File` --> `New video`
    1. Choice device
    1. Check what video work
    1. Exit ***QuickTime Player***
    1. Check what video activate on all physical devices:
    ```bash
    ./bin/ivf_pull list
    ```
1. Run `./bin/ios_video_pull -devices -decimal` to determine the PID ( product ID ) of your IOS device in decimal
1. Run `./bin/devreset [decimal product ID] 1452` to reset the video streaming status of your IOS device
1. Activate screen on all physical devices and run `./run` ( and leave it running )
1. Permissions dialog boxes appear for coordinator to listen on various ports; select accept for all of them
1. Device shows up in STF with video and can be controlled.
