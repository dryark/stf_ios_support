package main

import (
  "sort"
  "strconv"
  "strings"
  log "github.com/sirupsen/logrus"
)

type PortItem struct {
    available bool
}

type PortMap struct {
    wdaPorts    map[int] *PortItem
    vidPorts    map[int] *PortItem
    devIosPorts map[int] *PortItem
    vncPorts    map[int] *PortItem
    decodePorts map[int] *PortItem
}

func NewPortMap( config *Config ) ( *PortMap ) {
    wdaPorts    := construct_ports( "WDA", config, config.Network.Wda    )
    vidPorts    := construct_ports( "Video", config, config.Network.Video  )
    devIosPorts := construct_ports( "Dev IOS", config, config.Network.DevIos ) 
    vncPorts    := construct_ports( "VNC", config, config.Network.Vnc    )
    decodePorts := construct_ports( "Decode", config, config.Network.Decode )
    portMap := PortMap {
        wdaPorts: wdaPorts,
        vidPorts: vidPorts,
        devIosPorts: devIosPorts,
        vncPorts: vncPorts,
        decodePorts: decodePorts,
    }
    return &portMap
}

func construct_ports( name string, config *Config, spec string ) ( map [int] *PortItem ) {
    ports := make( map [int] *PortItem )
    if strings.Contains( spec, "-" ) {
        parts := strings.Split( spec, "-" )
        from, _ := strconv.Atoi( parts[0] )
        to, _ := strconv.Atoi( parts[1] )
        for i := from; i <= to; i++ {
            portItem := PortItem{
                available: true,
            }
            ports[ i ] = &portItem
        }
    } else {
        log.WithFields( log.Fields{
            "type":    "portmap",
            "related": name,
            "spec":    spec,
        } ).Fatal("Invalid ports spec")
    }
    return ports
}

func map_keys( amap map[int] *PortItem ) ( []int ) {
    arr := make( []int, len(amap) )
    i := 0
    for k := range amap {
        arr[i] = k
        i++
    }
    sort.Ints( arr )
    return arr
}

func assign_port( amap map[int] *PortItem ) (int) {
    arr := map_keys( amap )
    for _,port := range arr {
        portItem := amap[port]
        if portItem.available {
            portItem.available = false
            return port
        }
    }
    return 0
}

func assign_ports( gConfig *Config, portMap *PortMap ) ( int,int,int,int,int,int,*Config ) {
    dupConfig := *gConfig

    wdaPort := assign_port( portMap.wdaPorts )
    dupConfig.WDAProxyPort = wdaPort
    
    vidPort := assign_port( portMap.vidPorts )
    dupConfig.MirrorFeedPort = vidPort
    
    devIosPort := assign_port( portMap.devIosPorts )
    dupConfig.DevIosPort = devIosPort
    
    vncPort := assign_port( portMap.vncPorts )
    dupConfig.VncPort = vncPort
    
    nanoOutPort := assign_port( portMap.decodePorts )
    nanoInPort := assign_port( portMap.decodePorts )
    dupConfig.DecodeOutPort = nanoOutPort
    dupConfig.DecodeInPort = nanoInPort

    return wdaPort, vidPort, devIosPort, vncPort, nanoOutPort, nanoInPort, &dupConfig
}

func free_ports(
        wdaPort int,
        vidPort int,
        devIosPort int,
        vncPort int,
        portMap *PortMap ) {
    wdaItem := portMap.wdaPorts[ wdaPort ]
    wdaItem.available = true

    vidItem := portMap.vidPorts[ vidPort ]
    vidItem.available = true
    
    dItem := portMap.devIosPorts[ devIosPort ]
    dItem.available = true
    
    vncItem := portMap.vncPorts[ vncPort ]
    vncItem.available = true
}