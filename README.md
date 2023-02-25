# Simple Private Virtual Service on Tailscale

This module is intended to remove some of the boilerplate for running simple web services within a [tailscale](https://tailscale.com/) tailnet.

## Adding to project
```
go get -u github.com/kellegous/tsweb
```

## Example Use

```go
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"tailscale.com/tsnet"

	"github.com/kellegous/tsweb"
)

func main() {
	s, err := tsweb.Start(&tsnet.Server{
		AuthKey:  os.Getenv("TS_AUTHKEY"),
		Hostname: "sample",
		Dir:      "data",
	})
	if err != nil {
		log.Panic(err)
	}
	defer s.Close()

	l, err := s.ListenTLS("tcp", ":https")
	if err != nil {
		log.Panic(err)
	}

	log.Panic(
		http.Serve(l, http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintf(w, "Hello Tailnet")
			},
		)),
	)
}
```

## Authors
 - Kelly Norton ([kellegous](https://github.com/kellegous))