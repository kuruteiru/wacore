package main

import (
    "log"
    "os"
)

func main() {
    logger := log.New(os.Stdout, "wacore: ", log.LstdFlags)

    router := Route()
    router = Logging(logger)(router)
    router = Tracing(NextRequestID)(router)

    Serve(&router, logger)
}
