package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
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
	"github.com/urfave/cli"
)

const (
	name    = "syncy"
	version = "dev"
)

var (
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
	app.Version = version
	app.Flags = []cli.Flag{
		printerFlag,
	}
	app.Commands = []cli.Command{
		syncCommand(serverSchemeFlag, serverAddrFlag, serverPortFlag, serverPathFlag),
		statsCommands(serverSchemeFlag, serverAddrFlag, serverPortFlag, serverPathFlag),
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

// Sync project command: sync <absolute folder path>

func syncCommand(serverSchemeFlag, serverAddrFlag, serverPortFlag, serverPathFlag cli.StringFlag) cli.Command {
	return cli.Command{
		Name:  "sync",
		Usage: "sync a path against a backend",
		Flags: []cli.Flag{serverSchemeFlag, serverAddrFlag, serverPortFlag, serverPathFlag},
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
			client, err := makeClient(cctx, httpClient, serverSchemeFlag, serverAddrFlag, serverPortFlag, serverPathFlag)
			if err != nil {
				return fmt.Errorf("creating sync service client: %w", err)
			}

			ll.InfoContext(ctx, "preparing to sync", slog.String("path", path))

			res, err := client.GetSignature(ctx, connect.NewRequest(&syncv1.GetSignatureRequest{}))
			if err != nil {
				return fmt.Errorf("getting signature of remote tree: %w", err)
			}

			src := os.DirFS(path).(dirsync.Source)

			createOp, deleteOp, patchOp, err := dirsync.ComputeTreeDiff(ctx, path, src, res.Msg.Root)
			if err != nil {
				return fmt.Errorf("computing tree diff: %w", err)
			}
			_, _, _ = createOp, deleteOp, patchOp

			printer.Emit(res.Msg)

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
					ctx, ll, printer, err := makeDeps(cctx)
					if err != nil {
						return fmt.Errorf("preparing dependencies: %w", err)
					}
					httpClient, err := makeHttpClient(cctx)
					if err != nil {
						return fmt.Errorf("creating http client: %w", err)
					}
					client, err := makeClient(cctx, httpClient, serverSchemeFlag, serverAddrFlag, serverPortFlag, serverPathFlag)
					if err != nil {
						return fmt.Errorf("creating sync service client: %w", err)
					}

					path := cctx.Args().First()
					ll.InfoContext(ctx, "preparing to sync", slog.String("path", path))

					res, err := client.Stat(ctx, connect.NewRequest(&syncv1.StatRequest{
						Path: &typesv1.Path{
							Elements: filepath.SplitList(path),
						},
					}))
					if err != nil {
						return fmt.Errorf("stating path: %w", err)
					}

					fi := res.Msg.Info
					if fi.IsDir {
						// it's possible to stat a directory as a file, but the brief requests otherwise
						printer.Error("path is a directory, not a file")
						return fmt.Errorf("%q is a directory, not a file", path)
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
					ctx, ll, printer, err := makeDeps(cctx)
					if err != nil {
						return fmt.Errorf("preparing dependencies: %w", err)
					}
					httpClient, err := makeHttpClient(cctx)
					if err != nil {
						return fmt.Errorf("creating http client: %w", err)
					}
					client, err := makeClient(cctx, httpClient, serverSchemeFlag, serverAddrFlag, serverPortFlag, serverPathFlag)
					if err != nil {
						return fmt.Errorf("creating sync service client: %w", err)
					}

					path := cctx.Args().First()
					ll.InfoContext(ctx, "preparing to sync", slog.String("path", path))

					res, err := client.ListDir(ctx, connect.NewRequest(&syncv1.ListDirRequest{
						Path: &typesv1.Path{
							Elements: filepath.SplitList(path),
						},
					}))
					if err != nil {
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
) (syncv1connect.SyncServiceClient, error) {
	baseURL := url.URL{
		Scheme: cctx.String(serverSchemeFlag.Name),
		Host: net.JoinHostPort(
			cctx.String(serverAddrFlag.Name),
			cctx.String(serverPortFlag.Name),
		),
		Path: cctx.String(serverPathFlag.Name),
	}
	return syncv1connect.NewSyncServiceClient(httpClient, baseURL.String()), nil
}
