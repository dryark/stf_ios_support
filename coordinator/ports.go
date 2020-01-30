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
}

func NewPortMap( config *Config ) ( *PortMap ) {
    wdaPorts    := construct_ports( "WDA", config, config.Network.Wda    )
    vidPorts    := construct_ports( "Video", config, config.Network.Video  )
    devIosPorts := construct_ports( "Dev IOS", config, config.Network.DevIos ) 
    vncPorts    := construct_ports( "VNC", config, config.Network.Vnc    )
    portMap := PortMap {
        wdaPorts: wdaPorts,
        vidPorts: vidPorts,
        devIosPorts: devIosPorts,
        vncPorts: vncPorts,
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

func assign_ports( gConfig *Config, portMap *PortMap ) ( int,int,int,int,*Config ) {
    dupConfig := *gConfig

    wdaPort := 0
    vidPort := 0
    devIosPort := 0
    vncPort := 0

    wKeys := make( []int, len(portMap.wdaPorts) )
    wI := 0
    for k := range portMap.wdaPorts {
        wKeys[wI] = k
        wI++
    }
    sort.Ints( wKeys )

    vKeys := make( []int, len(portMap.vidPorts) )
    vI := 0
    for k := range portMap.vidPorts {
        vKeys[vI] = k
        vI++
    }
    sort.Ints( vKeys )
    
    xKeys := make( []int, len(portMap.devIosPorts) )
    xI := 0
    for k := range portMap.devIosPorts {
        xKeys[xI] = k
        xI++
    }
    sort.Ints( xKeys )
    
    vncKeys := make( []int, len(portMap.vncPorts) )
    vncI := 0
    for k := range portMap.vncPorts {
        vncKeys[vncI] = k
        vncI++
    }
    sort.Ints( vncKeys )

    for _,port := range wKeys {
        portItem := portMap.wdaPorts[port]
        if portItem.available {
            portItem.available = false
            dupConfig.WDAProxyPort = port
            wdaPort = port
            break
        }
    }

    for _,port := range vKeys {
        portItem := portMap.vidPorts[port]
        if portItem.available {
            portItem.available = false
            dupConfig.MirrorFeedPort = port
            vidPort = port
            break
        }
    }
    
    for _,port := range xKeys {
        portItem := portMap.devIosPorts[port]
        if portItem.available {
            portItem.available = false
            dupConfig.DevIosPort = port
            devIosPort = port
            break
        }
    }
    
    for _,port := range vncKeys {
        portItem := portMap.vncPorts[port]
        if portItem.available {
            portItem.available = false
            dupConfig.VncPort = port
            vncPort = port
            break
        }
    }

    return wdaPort, vidPort, devIosPort, vncPort, &dupConfig
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