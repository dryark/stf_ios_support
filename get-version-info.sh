#!/usr/bin/python

import json
import os
import subprocess
import sys

def git_info( dir, ascend ):
  os.chdir( dir )
  res = subprocess.check_output( "git log -1 | head -3", shell=True )
  remote = subprocess.check_output( ["/usr/bin/git", "remote","-v"] )
  os.chdir( ascend )
  
  res = res[:-1] # remove trailing "\n"
  remote = remote[:-1]
  remote = remote.split("\n")[0].split("\t") # just first line
  
  parts = res.split("\n")
  
  return {
    "commit": parts[0][7:],          # remove 'commit '
    "author": parts[1][7:].lstrip(), # remove 'Author:' and spaces
    "date": parts[2][5:].lstrip(),   # remove 'Date:' and spaces
    "remote": remote[1].replace(" (fetch)",""), # just url to fetch
  }

def xcode_version():
  res = subprocess.check_output( ["/usr/bin/xcodebuild", "-version"] )
  res = res[:-1]
  return res.split("\n")

if( len( sys.argv ) < 2 ):
  data = {
    "wda": git_info( 'repos/WebDriverAgent', '../..' ),
    "ffmpeg": git_info( 'repos/ffmpeg', '../..' ),
    "device_trigger": git_info( 'repos/osx_ios_device_trigger', '../..' ),
    "stf": git_info( 'repos/stf-ios-provider', '../..' ),
    "mirrorfeed": git_info( 'repos/stf_ios_mirrorfeed', '../..' ),
    "wdaproxy": git_info( 'repos/wdaproxy', '../..' ),
    "ios_support": git_info( '.', '.' ),
  }
  if os.path.exists( 'bin/wda/build_info.json' ):
    fh = open( 'bin/wda/build_info.json', 'r' )
    wda_root = json.load( fh )
    data["wda"] = wda_root["wda"]
else:
  if( sys.argv[1] == 'wda' ):
    data = {
      "wda": git_info( 'repos/WebDriverAgent', '../..' ),
    }
    data['wda']['xcode'] = xcode_version()
  if( sys.argv[1] == 'ios_support' ):
    data = {
      "ios_support": git_info( '.', '.' ),
    }

print json.dumps( data, indent = 2 )

