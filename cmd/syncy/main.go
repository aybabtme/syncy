package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"connectrpc.com/connect"
	syncv1 "github.com/aybabtme/syncy/pkg/gen/svc/sync/v1"
	"github.com/aybabtme/syncy/pkg/gen/svc/sync/v1/syncv1connect"
	typesv1 "github.com/aybabtme/syncy/pkg/gen/types/v1"
	"github.com/aybabtme/syncy/pkg/logic/dirsync"
	"github.com/aybabtme/syncy/pkg/logic/syncclient"
	"github.com/urfave/cli"
)

const (
	name    = "syncy"
	version = "dev"
)

var (
	accountIDFlag = cli.StringFlag{
		Name:   "account_id",
		EnvVar: "SYNCY_ACCOUNT_ID",
	}
	projectIDFlag = cli.StringFlag{
		Name:   "project_id",
		EnvVar: "SYNCY_PROJECT_ID",
	}
	serverSchemeFlag = cli.StringFlag{
		Name:  "server.schema",
		Value: "http",
	}
	serverAddrFlag = cli.StringFlag{
		Name:  "server.addr",
		Value: "127.0.0.1",
	}
	serverPortFlag = cli.StringFlag{
		Name:  "server.port",
		Value: "7071",
	}
	serverPathFlag = cli.StringFlag{
		Name:  "server.path",
		Value: "/",
	}
	maxParallelFileStreamFlag = cli.UintFlag{
		Name:  "max.parallel_file_stream",
		Value: 1,
		Usage: "max number of files being uploaded or patches in parallel at any given time",
	}
	blockSizeFlag = cli.UintFlag{
		Name:  "block.size",
		Value: 2 << 16,
		Usage: "block size for rsync algorithm",
	}
	scratchLocalPath = cli.StringFlag{
		Name:  "scratch.local_path",
		Value: "/tmp/syncy_scratch",
	}
	outFlag = cli.StringFlag{
		Name:  "out",
		Usage: "if specified, the file where to write the output",
	}
	printerFlag = cli.StringFlag{
		Name:  "printer",
		Usage: "must be one of: text, json",
		Value: "json",
	}
)

func main() {

	log.SetFlags(0)
	log.SetPrefix(name + ": ")

	app := cli.NewApp()
	app.Name = name
	app.Usage = "an utility to sync local directories with a remote backend"
	app.Version = version
	app.Flags = []cli.Flag{
		printerFlag,
		accountIDFlag,
		projectIDFlag,
	}
	app.Commands = []cli.Command{
		syncCommand(serverSchemeFlag, serverAddrFlag, serverPortFlag, serverPathFlag, maxParallelFileStreamFlag),
		statsCommands(serverSchemeFlag, serverAddrFlag, serverPortFlag, serverPathFlag),
		debugCommands(outFlag, scratchLocalPath),
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

// Sync project command: sync <absolute folder path>

func syncCommand(serverSchemeFlag, serverAddrFlag, serverPortFlag, serverPathFlag cli.StringFlag, maxParallelFileStreamFlag cli.UintFlag) cli.Command {
	return cli.Command{
		Name:  "sync",
		Usage: "sync a path against a backend",
		Flags: []cli.Flag{serverSchemeFlag, serverAddrFlag, serverPortFlag, serverPathFlag, maxParallelFileStreamFlag, blockSizeFlag},
		Action: func(cctx *cli.Context) error {
			path := cctx.Args().First()
			if !filepath.IsAbs(path) {
				return fmt.Errorf("<path> is not absolute")
			}
			ctx, ll, printer, err := makeDeps(cctx)
			if err != nil {
				return fmt.Errorf("preparing dependencies: %w", err)
			}
			httpClient, err := makeHttpClient(cctx)
			if err != nil {
				return fmt.Errorf("creating http client: %w", err)
			}
			client, meta, err := makeClient(cctx, httpClient, serverSchemeFlag, serverAddrFlag, serverPortFlag, serverPathFlag)
			if err != nil {
				return fmt.Errorf("creating sync service client: %w", err)
			}
			blockSize := cctx.Uint(blockSizeFlag.Name)
			if blockSize < 128 {
				return fmt.Errorf("minimum block size is 128")
			}
			if blockSize >= math.MaxUint32 {
				return fmt.Errorf("block size must fit in a uint32")
			}

			sink, err := syncclient.ClientAdapter(ll, client, meta, blockSize)
			if err != nil {
				return fmt.Errorf("configuring sync service client: %w", err)
			}

			maxParallelFileStream := cctx.Uint(maxParallelFileStreamFlag.Name)

			syncParams := dirsync.Params{
				MaxParallelFileStreams: int(maxParallelFileStream),
			}

			ll.InfoContext(ctx, "preparing to sync", slog.String("path", path))

			src := os.DirFS(path).(dirsync.Source)

			err = dirsync.Sync(ctx, ".", src, sink, syncParams)
			if err != nil {
				return fmt.Errorf("failed to sync: %w", err)
			}

			printer.Emit("sync completed")

			return nil
		},
	}
}

// File stats command: stats file <absolute file path>
// Folder stats command: stats folder <absolute folder path>

func statsCommands(serverSchemeFlag, serverAddrFlag, serverPortFlag, serverPathFlag cli.StringFlag) cli.Command {
	return cli.Command{
		Name:  "stats",
		Usage: "stats about files and directories",
		Subcommands: []cli.Command{
			{
				Name:  "file",
				Usage: "describes a file",
				Flags: []cli.Flag{serverSchemeFlag, serverAddrFlag, serverPortFlag, serverPathFlag},
				Action: func(cctx *cli.Context) error {
					path := cctx.Args().First()
					if path == "" {
						return fmt.Errorf("<path> is required")
					}
					ctx, ll, printer, err := makeDeps(cctx)
					if err != nil {
						return fmt.Errorf("preparing dependencies: %w", err)
					}
					httpClient, err := makeHttpClient(cctx)
					if err != nil {
						return fmt.Errorf("creating http client: %w", err)
					}
					client, meta, err := makeClient(cctx, httpClient, serverSchemeFlag, serverAddrFlag, serverPortFlag, serverPathFlag)
					if err != nil {
						return fmt.Errorf("creating sync service client: %w", err)
					}

					ll.InfoContext(ctx, "stating file", slog.String("path", path))

					res, err := client.Stat(ctx, connect.NewRequest(&syncv1.StatRequest{
						Meta: meta,
						Path: typesv1.PathFromString(path),
					}))
					if err != nil {
						if cerr, ok := err.(*connect.Error); ok {
							if cerr.Code() == connect.CodeNotFound {
								printer.Error("no file with this name")
							} else {
								printer.Error(cerr.Error())
							}
							return nil
						}
						return fmt.Errorf("listing path: %w", err)
					}

					fi := res.Msg.Info
					if fi.IsDir {
						// it's possible to stat a directory as a file, but the brief requests otherwise
						printer.Error("path is a directory, not a file")
						return nil
					}

					printer.Emit(fi)

					return nil
				},
			},
			{
				Name:  "dir",
				Usage: "list a directory's files and child directories",
				Flags: []cli.Flag{serverSchemeFlag, serverAddrFlag, serverPortFlag, serverPathFlag},
				Action: func(cctx *cli.Context) error {
					path := cctx.Args().First()
					if path == "" {
						return fmt.Errorf("<path> is required")
					}
					ctx, ll, printer, err := makeDeps(cctx)
					if err != nil {
						return fmt.Errorf("preparing dependencies: %w", err)
					}
					httpClient, err := makeHttpClient(cctx)
					if err != nil {
						return fmt.Errorf("creating http client: %w", err)
					}
					client, meta, err := makeClient(cctx, httpClient, serverSchemeFlag, serverAddrFlag, serverPortFlag, serverPathFlag)
					if err != nil {
						return fmt.Errorf("creating sync service client: %w", err)
					}

					ll.InfoContext(ctx, "listing dirs", slog.String("path", path))

					res, err := client.ListDir(ctx, connect.NewRequest(&syncv1.ListDirRequest{
						Meta: meta,
						Path: typesv1.PathFromString(path),
					}))
					if err != nil {
						if cerr, ok := err.(*connect.Error); ok {
							if cerr.Code() == connect.CodeNotFound {
								printer.Error("no dir with this name")
							} else {
								printer.Error(cerr.Error())
							}
							return nil
						}
						return fmt.Errorf("listing path: %w", err)
					}

					printer.Emit(res.Msg.DirEntries)

					return nil
				},
			},
		},
	}
}

func makeDeps(cctx *cli.Context) (context.Context, *slog.Logger, printer, error) {
	ctx, err := makeContext(cctx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("creating context: %w", err)
	}
	logger, err := makeLogger(cctx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("creating logger: %w", err)
	}
	printer, err := makePrinter(cctx, printerFlag)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("creating printer: %w", err)
	}
	return ctx, logger, printer, nil
}

func makeContext(_ *cli.Context) (context.Context, error) {
	return context.Background(), nil
}

func makeLogger(_ *cli.Context) (*slog.Logger, error) {
	return slog.New(slog.NewTextHandler(os.Stderr, nil)), nil
}

func makePrinter(cctx *cli.Context, printerFlag cli.StringFlag) (printer, error) {
	printer := cctx.GlobalString(printerFlag.Name)
	switch printer {
	case "text":
		return newTextPrinter(os.Stdout), nil
	case "json":
		return newJSONPrinter(os.Stdout), nil
	case "proto", "pb":
		return newProtoPrinter(os.Stdout), nil
	default:
		return nil, fmt.Errorf("unsupported printer format: %q", printer)
	}
}

func makeHttpClient(
	_ *cli.Context,
) (connect.HTTPClient, error) {
	return &http.Client{
		// configure read/write stuff, etc
	}, nil
}

func makeClient(
	cctx *cli.Context,
	httpClient connect.HTTPClient,
	serverSchemeFlag cli.StringFlag,
	serverAddrFlag cli.StringFlag,
	serverPortFlag cli.StringFlag,
	serverPathFlag cli.StringFlag,
) (syncv1connect.SyncServiceClient, *typesv1.ReqMeta, error) {
	baseURL := url.URL{
		Scheme: cctx.String(serverSchemeFlag.Name),
		Host: net.JoinHostPort(
			cctx.String(serverAddrFlag.Name),
			cctx.String(serverPortFlag.Name),
		),
		Path: cctx.String(serverPathFlag.Name),
	}
	req := &typesv1.ReqMeta{
		AccountId: stringFlagOrEnvVar(cctx, accountIDFlag),
		ProjectId: stringFlagOrEnvVar(cctx, projectIDFlag),
	}

	return syncv1connect.NewSyncServiceClient(httpClient, baseURL.String()), req, nil
}

func stringFlagOrEnvVar(cctx *cli.Context, flag cli.StringFlag) string {
	if cctx.IsSet(flag.Name) {
		return cctx.String(flag.Name)
	}
	return os.Getenv(flag.EnvVar)
}
