package main

import (
    "fmt"
    "os"
    "os/signal"
    "syscall"
    "strings"
    ps "github.com/jviney/go-proc"
)

func coro_sigterm( proc *GenericProc, binary string ) {
    c := make(chan os.Signal, 2)
    signal.Notify(c, os.Interrupt, syscall.SIGTERM)
    go func() {
        <- c
        fmt.Println("\nShutdown started")

        // shutdown proc
        proc.Kill()
        cleanup_subprocs( binary )
        
        fmt.Println("Shutdown finished")

        os.Exit(0)
    }()
}

func cleanup_procs() {
    // Cleanup hanging processes if any
    procs := ps.GetAllProcessesInfo()
    for _, proc := range procs {
        cmd := proc.CommandLine
        if strings.HasSuffix( cmd[0], "/bin/coordinator" ) {
            fmt.Printf("Leftover coordinator with PID %d. Killing\n", proc.Pid )
            syscall.Kill( proc.Pid, syscall.SIGTERM )
        }
    }
}