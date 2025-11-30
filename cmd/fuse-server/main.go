package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	_ "bazil.org/fuse/fs/fstestutil"
	"github.com/fwslash/biggieLoss/internal/filesystem"
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s MOUNTPOINT\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() != 1 {
		usage()
		os.Exit(2)
	}
	mountpoint := flag.Arg(0)

	myFS := filesystem.NewFS()

	c, err := fuse.Mount(
		mountpoint,
		fuse.FSName("We will lose your data. I had couple a beers so this message is now too long for the df output. I mean it works it just looks ugly."),
		fuse.Subtype("biggieLoss-FS"),
	)

	if err != nil {
		log.Fatal(err)
	}

	defer c.Close()
	defer myFS.RedisClient.Close()

	err = fs.Serve(c, myFS)
	if err != nil {
		log.Fatal(err)
	}
}
