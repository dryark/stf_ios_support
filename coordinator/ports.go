package main

import "sort"

type PortItem struct {
    available bool
}

func assign_ports( 
        gConfig *Config,
        wdaPorts    map[int] *PortItem,
        vidPorts    map[int] *PortItem,
        devIosPorts map[int] *PortItem ) ( int,int,int,*Config ) {
    dupConfig := *gConfig

    wdaPort := 0
    vidPort := 0
    devIosPort := 0

    wKeys := make( []int, len(wdaPorts) )
    wI := 0
    for k := range wdaPorts {
        wKeys[wI] = k
        wI++
    }
    sort.Ints( wKeys )

    vKeys := make( []int, len(vidPorts) )
    vI := 0
    for k := range vidPorts {
        vKeys[vI] = k
        vI++
    }
    sort.Ints( vKeys )
    
    xKeys := make( []int, len(devIosPorts) )
    xI := 0
    for k := range devIosPorts {
        xKeys[xI] = k
        xI++
    }
    sort.Ints( xKeys )

    for _,port := range wKeys {
        portItem := wdaPorts[port]
        if portItem.available {
            portItem.available = false
            dupConfig.WDAProxyPort = port
            wdaPort = port
            break
        }
    }

    for _,port := range vKeys {
        portItem := vidPorts[port]
        if portItem.available {
            portItem.available = false
            dupConfig.MirrorFeedPort = port
            vidPort = port
            break
        }
    }
    
    for _,port := range xKeys {
        portItem := devIosPorts[port]
        if portItem.available {
            portItem.available = false
            dupConfig.DevIosPort = port
            devIosPort = port
            break
        }
    }

    return wdaPort, vidPort, devIosPort, &dupConfig
}

func free_ports(
        wdaPort int,
        vidPort int,
        devIosPort int,
        wdaPorts map [int] *PortItem, 
        vidPorts map [int] *PortItem,
        devIosPorts map[int] *PortItem ) {
    wdaItem := wdaPorts[ wdaPort ]
    wdaItem.available = true

    vidItem := vidPorts[ vidPort ]
    vidItem.available = true
    
    dItem := devIosPorts[ devIosPort ]
    dItem.available = true
}