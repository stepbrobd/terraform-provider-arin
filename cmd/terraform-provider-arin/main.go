// terraform-provider-arin serves the arin rpki provider over grpc
package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/stepbrobd/terraform-provider-arin/provider"
)

var version = "dev"

func main() {
	debug := flag.Bool("debug", false, "serve with support for debuggers like delve")
	flag.Parse()
	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/stepbrobd/arin",
		Debug:   *debug,
	}
	if err := providerserver.Serve(context.Background(), provider.New(version), opts); err != nil {
		log.Fatal(err)
	}
}
