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
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/kellegous/tsweb"
)

func main() {
	os.MkdirAll("data", 0700)

	s, err := tsweb.Start(
		context.Background(),
		&tsweb.ServiceOptions{
			AuthKey:  os.Getenv("TS_AUTHKEY"),
			Hostname: "sample",
			StateDir: "data",
		})
	if err != nil {
		log.Panic(err)
	}
	defer s.Close()

	log.Panic(
		http.Serve(s.Listener, http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintln(w, "Hello Tailnet")
			},
		)),
	)
}
```

## Authors
 - Kelly Norton ([kellegous](https://github.com/kellegous))