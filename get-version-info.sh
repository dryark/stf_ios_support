#!/usr/bin/python

import json
import os
import subprocess
import sys
import argparse

parser = argparse.ArgumentParser( description = "Collect git version info" )
parser.add_argument('--repo',default="")
parser.add_argument('--unix',action="count")
parser.add_argument('--wdasource',action="count")
args = parser.parse_args()

def git_info( dir ):
  if args.unix==1:
    cmd = ["/usr/bin/git","-C","./"+dir,"log","-1","--date=unix", "--no-merges"]
  else:
    cmd = ["/usr/bin/git","-C","./"+dir,"log","-1", "--no-merges"]

  try:
    res = subprocess.check_output( cmd, stderr=subprocess.STDOUT )
  except subprocess.CalledProcessError as e:
    #sys.stderr.write( e.output )
    return {
      "error": "missing"#e.output
    }
  
  remote = subprocess.check_output( ["/usr/bin/git", "-C", "./" + dir, "remote","-v"] )
  
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

if args.repo != "":
  if args.repo == 'wda':
    data = {
      "wda": git_info( 'repos/WebDriverAgent' ),
    }
    data['wda']['xcode'] = xcode_version()
  if args.repo == 'ios_support':
    data = {
      "ios_support": git_info( '.' ),
    }
else:
  data = {
    "wda": git_info( 'repos/WebDriverAgent' ),
    "h264_to_jpeg": git_info( 'repos/h264_to_jpeg' ),
    "device_trigger": git_info( 'repos/osx_ios_device_trigger' ),
    "stf": git_info( 'repos/stf-ios-provider' ),
    "ios_video_stream": git_info( 'repos/ios_video_stream' ),
    "wdaproxy": git_info( 'repos/wdaproxy' ),
    "ios_support": git_info( '.' ),
    "ios_avf_pull": git_info( 'repos/ios_avf_pull' )
  }
  if os.path.exists( 'bin/wda/build_info.json' ):
    fh = open( 'bin/wda/build_info.json', 'r' )
    wda_root = json.load( fh )
    if args.wdasource != 1:
      data["wda"] = wda_root["wda"]

print json.dumps( data, indent = 2 )

