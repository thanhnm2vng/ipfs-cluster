package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	ipfslite "github.com/hsanjuan/ipfs-lite"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/namespace"
	crdt "github.com/ipfs/go-ds-crdt"
	"github.com/ipfs/ipfs-cluster/cmdutils"
	"github.com/ipfs/ipfs-cluster/version"
	"github.com/urfave/cli"
)

// Default location for the configurations and data
var (
	// DefaultFolder is the name of the cluster folder
	DefaultFolder = ".ipfs-cluster"
	// DefaultPath is set on init() to $HOME/DefaultFolder
	// and holds all the ipfs-cluster data
	DefaultPath string
	// The name of the configuration file inside DefaultPath
	DefaultConfigFile = "service.json"
	// The name of the identity file inside DefaultPath
	DefaultIdentityFile = "identity.json"
)

var (
	configPath   string
	identityPath string
)

func init() {
	// We try guessing user's home from the HOME variable. This
	// allows HOME hacks for things like Snapcraft builds. HOME
	// should be set in all UNIX by the OS. Alternatively, we fall back to
	// usr.HomeDir (which should work on Windows etc.).
	home := os.Getenv("HOME")
	if home == "" {
		usr, err := user.Current()
		if err != nil {
			panic(fmt.Sprintf("cannot get current user: %s", err))
		}
		home = usr.HomeDir
	}

	DefaultPath = filepath.Join(home, DefaultFolder)
}

func main() {
	app := cli.NewApp()
	app.Name = "crdtdot"
	app.Usage = "Dot exporter for Cluster CRDT dag"
	app.Description = "Export a dot file containing the full CRDT dag for this node"
	app.Version = version.Version.String()
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "config, c",
			Value:  DefaultPath,
			Usage:  "path to the configuration and data `FOLDER`",
			EnvVar: "IPFS_CLUSTER_PATH",
		},
	}
	app.Before = func(c *cli.Context) error {
		absPath, err := filepath.Abs(c.String("config"))
		if err != nil {
			return err
		}

		configPath = filepath.Join(absPath, DefaultConfigFile)
		identityPath = filepath.Join(absPath, DefaultIdentityFile)
		return nil
	}

	app.Action = func(c *cli.Context) error {
		cfgHelper, err := cmdutils.NewLoadedConfigHelper(
			configPath,
			identityPath,
		)
		checkErr("loading configurations", err)
		cfgHelper.Manager().Shutdown()
		mgr, err := cmdutils.NewStateManagerWithHelper(cfgHelper)
		checkErr("creating state manager", err)

		store, err := mgr.GetStore()
		checkErr("opening datastore", err)
		batching, ok := store.(datastore.Batching)
		if !ok {
			checkErr("", errors.New("no batching store"))
		}

		opts := crdt.DefaultOptions()
		cfg := cfgHelper.Configs().Crdt

		var blocksDatastore datastore.Batching = namespace.Wrap(
			batching,
			datastore.NewKey(cfg.DatastoreNamespace).ChildString("b"),
		)

		ipfs, err := ipfslite.New(
			context.Background(),
			blocksDatastore,
			nil,
			nil,
			&ipfslite.Config{
				Offline: true,
			},
		)
		checkErr("creating ipfs-lite node", err)

		crdt, err := crdt.New(
			batching,
			datastore.NewKey(cfg.DatastoreNamespace),
			ipfs,
			nil,
			opts,
		)
		checkErr("creating crdt node", err)

		buf := bufio.NewWriter(os.Stdout)
		defer buf.Flush()

		checkErr("generating graph", crdt.DotDAG(buf))
		return nil
	}

	app.Run(os.Args)
}

func checkErr(doing string, err error, args ...interface{}) {
	if err != nil {
		if len(args) > 0 {
			doing = fmt.Sprintf(doing, args...)
		}
		fmt.Fprintf(os.Stderr, "error %s: %s\n", doing, err)
		os.Exit(1)
	}
}
