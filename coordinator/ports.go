package main

import "sort"

type PortItem struct {
    available bool
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