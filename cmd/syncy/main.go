package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"connectrpc.com/connect"
	syncv1 "github.com/aybabtme/syncy/pkg/gen/svc/sync/v1"
	"github.com/aybabtme/syncy/pkg/gen/svc/sync/v1/syncv1connect"
	typesv1 "github.com/aybabtme/syncy/pkg/gen/types/v1"
	"github.com/aybabtme/syncy/pkg/logic/dirsync"
	"github.com/aybabtme/syncy/pkg/logic/patchcodec"
	"github.com/aybabtme/syncy/pkg/logic/syncclient"
	"github.com/aybabtme/syncy/pkg/storage/blobdb"
	"github.com/dustin/go-humanize"
	"github.com/urfave/cli"
	"google.golang.org/protobuf/proto"
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
		Value: 8,
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

			sink, err := syncclient.ClientAdapter(client, meta, blockSize)
			if err != nil {
				return fmt.Errorf("configuring sync service client: %w", err)
			}

			maxParallelFileStream := cctx.Uint(maxParallelFileStreamFlag.Name)

			syncParams := dirsync.Params{
				MaxParallelFileStreams: int(maxParallelFileStream),
			}

			ll.InfoContext(ctx, "preparing to sync", slog.String("path", path))

			src := os.DirFS(path).(dirsync.Source)

			err = dirsync.Sync(ctx, path, src, sink, syncParams)
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

					ll.DebugContext(ctx, "stating file", slog.String("path", path))

					res, err := client.Stat(ctx, connect.NewRequest(&syncv1.StatRequest{
						Meta: meta,
						Path: &typesv1.Path{
							Elements: filepath.SplitList(path),
						},
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

					ll.DebugContext(ctx, "listing dirs", slog.String("path", path))

					res, err := client.ListDir(ctx, connect.NewRequest(&syncv1.ListDirRequest{
						Meta: meta,
						Path: &typesv1.Path{
							Elements: filepath.SplitList(path),
						},
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

func debugCommands(outFlag, scratchLocalPath cli.StringFlag) cli.Command {
	return cli.Command{
		Name:  "debug",
		Usage: "debug commands",
		Subcommands: []cli.Command{
			{
				Name:  "create-account",
				Usage: "create an account on a remote backend",
				Flags: []cli.Flag{serverSchemeFlag, serverAddrFlag, serverPortFlag, serverPathFlag, blockSizeFlag},
				Action: func(cctx *cli.Context) error {
					account := cctx.Args().Get(0)
					if account == "" {
						return fmt.Errorf("<account> is required")
					}
					ctx, ll, printer, err := makeDeps(cctx)
					if err != nil {
						return fmt.Errorf("preparing dependencies: %w", err)
					}
					httpClient, err := makeHttpClient(cctx)
					if err != nil {
						return fmt.Errorf("creating http client: %w", err)
					}
					client, _, err := makeClient(cctx, httpClient, serverSchemeFlag, serverAddrFlag, serverPortFlag, serverPathFlag)
					if err != nil {
						return fmt.Errorf("creating sync service client: %w", err)
					}
					ll.InfoContext(ctx, "creating account")
					res, err := client.CreateAccount(ctx, connect.NewRequest(&syncv1.CreateAccountRequest{
						AccountName: account,
					}))
					if err != nil {
						return fmt.Errorf("creating account: %w", err)
					}
					ll.InfoContext(ctx, "account created, use the account_id (SYNCY_ACCOUNT_ID) for future requests")
					printer.Emit(map[string]string{"account_id": res.Msg.GetAccountId()})
					return nil
				},
			},
			{
				Name:  "create-project",
				Usage: "create an project on a remote backend",
				Flags: []cli.Flag{serverSchemeFlag, serverAddrFlag, serverPortFlag, serverPathFlag, blockSizeFlag},
				Action: func(cctx *cli.Context) error {
					project := cctx.Args().Get(0)
					if project == "" {
						return fmt.Errorf("<project> is required")
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
					ll.InfoContext(ctx, "creating project")
					res, err := client.CreateProject(ctx, connect.NewRequest(&syncv1.CreateProjectRequest{
						AccountId:   meta.AccountId,
						ProjectName: project,
					}))
					if err != nil {
						return fmt.Errorf("creating project: %w", err)
					}
					ll.InfoContext(ctx, "project created, set the project (SYNCY_PROJECT_ID) for future requests")

					printer.Emit(map[string]string{"project_id": res.Msg.GetProjectId()})
					return nil
				},
			},
			{
				Name:  "dirsum",
				Usage: "builds the sum tree of a dir",
				Flags: []cli.Flag{},
				Action: func(cctx *cli.Context) error {
					path := cctx.Args().First()
					if path == "" {
						return fmt.Errorf("<path> is required")
					}
					ctx, ll, printer, err := makeDeps(cctx)
					if err != nil {
						return fmt.Errorf("preparing dependencies: %w", err)
					}

					blockSize := cctx.Uint(blockSizeFlag.Name)
					if blockSize < 128 {
						return fmt.Errorf("minimum block size is 128")
					}
					if blockSize >= math.MaxUint32 {
						return fmt.Errorf("block size must fit in a uint32")
					}

					scratch := cctx.String(scratchLocalPath.Name)
					dir := filepath.Dir(path)

					root := filepath.Base(path)
					blob, err := blobdb.NewLocalFS(dir, scratch)
					if err != nil {
						return fmt.Errorf("creating blob backend: %w", err)
					}

					ll.Info("starting trace of filesystem from path", slog.String("path", path))
					sum, err := dirsync.TraceSink(ctx, root, blob)
					if err != nil {
						return fmt.Errorf("tracing the filesystem: %w", err)
					}

					ll.Info("done tracing",
						slog.Uint64("total_size_bytes", sum.Size),
						slog.String("total_size", humanize.IBytes(sum.Size)),
					)
					printer.Emit(sum)

					return nil
				},
			},
			{
				Name:  "filesum",
				Usage: "builds the sum of a file",
				Flags: []cli.Flag{blockSizeFlag},
				Action: func(cctx *cli.Context) error {
					path := cctx.Args().First()
					if path == "" {
						return fmt.Errorf("<path> is required")
					}
					ctx, ll, printer, err := makeDeps(cctx)
					if err != nil {
						return fmt.Errorf("preparing dependencies: %w", err)
					}

					blockSize := cctx.Uint(blockSizeFlag.Name)
					if blockSize < 128 {
						return fmt.Errorf("minimum block size is 128")
					}
					if blockSize >= math.MaxUint32 {
						return fmt.Errorf("block size must fit in a uint32")
					}

					f, err := os.Open(path)
					if err != nil {
						return fmt.Errorf("opening file %q: %w", path, err)
					}
					defer f.Close()
					ll.Info("computing file sum", slog.String("path", path))

					sum, err := dirsync.ComputeFileSum(ctx, f)
					if err != nil {
						return fmt.Errorf("computing file sum: %w", err)
					}

					ll.Info("done computing file sum",
						slog.Uint64("size_bytes", sum.Info.Size),
						slog.String("size", humanize.IBytes(sum.Info.Size)),
					)
					printer.Emit(sum)

					return nil
				},
			},
			{
				Name:  "make-patch",
				Usage: "builds the patch list of a file",
				Flags: []cli.Flag{blockSizeFlag, outFlag},
				Action: func(cctx *cli.Context) error {
					src := cctx.Args().Get(0)
					if src == "" {
						return fmt.Errorf("<src> is required")
					}
					dst := cctx.Args().Get(1)
					if dst == "" {
						return fmt.Errorf("<dst> is required")
					}
					ctx, ll, _, err := makeDeps(cctx)
					if err != nil {
						return fmt.Errorf("preparing dependencies: %w", err)
					}

					blockSize := cctx.Uint(blockSizeFlag.Name)
					if blockSize < 128 {
						return fmt.Errorf("minimum block size is 128")
					}
					if blockSize >= math.MaxUint32 {
						return fmt.Errorf("block size must fit in a uint32")
					}

					var outf *os.File
					if out := cctx.String(outFlag.Name); out != "" {
						outf, err = os.Create(out)
						if err != nil {
							return fmt.Errorf("opening output file %q: %w", out, err)
						}
						defer outf.Close()
					} else {
						outf = os.Stdout
					}

					dstf, err := os.Open(dst)
					if err != nil {
						return fmt.Errorf("opening destination file %q: %w", dst, err)
					}
					defer dstf.Close()
					dstfi, err := dstf.Stat()
					if err != nil {
						return fmt.Errorf("stating destination file %q: %w", src, err)
					}

					srcf, err := os.Open(src)
					if err != nil {
						return fmt.Errorf("opening source file %q: %w", src, err)
					}
					defer srcf.Close()
					srcfi, err := srcf.Stat()
					if err != nil {
						return fmt.Errorf("stating source file %q: %w", src, err)
					}

					ll.Info("computing file sum for destination", slog.String("path", dst))
					sinkSum, err := dirsync.ComputeFileSum(ctx, dstf)
					if err != nil {
						return fmt.Errorf("computing file sum for destination: %w", err)
					}

					ll.Info("done computing file sum",
						slog.Uint64("size_bytes", sinkSum.Info.Size),
						slog.String("size", humanize.IBytes(sinkSum.Info.Size)),
					)

					ll.Info("generating patch list")
					start := time.Now()
					enc := patchcodec.NewEncoder(outf)
					patchSize, err := dirsync.Rsync(ctx, srcf, sinkSum,
						enc.WriteBlock,
						enc.WriteBlockID,
					)
					duration := time.Since(start)
					if err != nil {
						return fmt.Errorf("generating patch list for file: %w", err)
					}
					bytesPerSec := uint64(float64(srcfi.Size()) / duration.Seconds())
					ll.Info("patch list generated",
						slog.Int64("src_size_bytes", srcfi.Size()),
						slog.Int64("dst_size_bytes", dstfi.Size()),
						slog.Int("patch_size_bytes", patchSize),
						slog.String("src_size", humanize.IBytes(uint64(srcfi.Size()))),
						slog.String("dst_size", humanize.IBytes(uint64(dstfi.Size()))),
						slog.String("patch_size", humanize.IBytes(uint64(patchSize))),
						slog.Float64("speedup", float64(srcfi.Size())/float64(patchSize)),
						slog.String("patch_speed", humanize.IBytes(bytesPerSec)+"/s"),
					)

					return nil
				},
			},
			{
				Name:  "apply-patch",
				Usage: "applies a patch list to a file",
				Flags: []cli.Flag{},
				Action: func(cctx *cli.Context) error {
					orig := cctx.Args().Get(0)
					if orig == "" {
						return fmt.Errorf("<orig> is required")
					}
					patch := cctx.Args().Get(1)
					if patch == "" {
						return fmt.Errorf("<patch> is required")
					}
					sum := cctx.Args().Get(2)
					if sum == "" {
						return fmt.Errorf("<sum> is required")
					}
					dst := cctx.Args().Get(3)
					if dst == "" {
						return fmt.Errorf("<dst> is required")
					}

					_, ll, _, err := makeDeps(cctx)
					if err != nil {
						return fmt.Errorf("preparing dependencies: %w", err)
					}

					origf, err := os.Open(orig)
					if err != nil {
						return fmt.Errorf("opening original file %q: %w", orig, err)
					}
					defer origf.Close()
					origfi, err := origf.Stat()
					if err != nil {
						return fmt.Errorf("stating original file %q: %w", orig, err)
					}

					patchf, err := os.Open(patch)
					if err != nil {
						return fmt.Errorf("opening patch file %q: %w", patch, err)
					}
					defer patchf.Close()
					patchfi, err := patchf.Stat()
					if err != nil {
						return fmt.Errorf("stating patch file %q: %w", patch, err)
					}

					sumf, err := os.Open(sum)
					if err != nil {
						return fmt.Errorf("opening sum file %q: %w", sum, err)
					}
					defer sumf.Close()
					sumfi, err := sumf.Stat()
					if err != nil {
						return fmt.Errorf("stating sum file %q: %w", sum, err)
					}

					ll.Info("decoding file sum",
						slog.Uint64("size_bytes", uint64(sumfi.Size())),
						slog.String("size", humanize.IBytes(uint64(sumfi.Size()))),
					)

					treeSum := new(typesv1.FileSum)
					if sumdata, err := io.ReadAll(sumf); err != nil {
						return fmt.Errorf("reading sum file %q: %w", sum, err)
					} else if err := proto.Unmarshal(sumdata, treeSum); err != nil {
						return fmt.Errorf("decoding sum file %q: %w", sum, err)
					}

					ll.Info("done decoding file sum",
						slog.Uint64("size_bytes", treeSum.Info.Size),
						slog.String("size", humanize.IBytes(treeSum.Info.Size)),
					)

					dstf, err := os.Create(dst)
					if err != nil {
						return fmt.Errorf("creating destination file %q: %w", dst, err)
					}
					defer dstf.Close()

					ll.Info("decoding patch list and applying onto destination file")
					start := time.Now()
					patcher := dirsync.NewFilePatcher(origf, dstf, treeSum)
					patchedSize, err := patchcodec.NewDecoder(patchf).Decode(
						patcher.WriteBlock,
						patcher.Copy,
					)
					if err != nil {
						return fmt.Errorf("patching file %q: %w", patch, err)
					}
					duration := time.Since(start)
					bytesPerSec := uint64(float64(patchedSize) / duration.Seconds())
					ll.Info("patch applied",
						slog.Int64("orig_size_bytes", origfi.Size()),
						slog.Int64("patch_size_bytes", patchfi.Size()),
						slog.Int64("sum_size_bytes", sumfi.Size()),
						slog.Int("dst_size_bytes", patchedSize),
						slog.String("orig_size", humanize.IBytes(uint64(origfi.Size()))),
						slog.String("patch_size", humanize.IBytes(uint64(patchfi.Size()))),
						slog.String("sum_size", humanize.IBytes(uint64(sumfi.Size()))),
						slog.String("dst_size", humanize.IBytes(uint64(patchedSize))),
						slog.Float64("speedup", float64(patchfi.Size())/float64(patchedSize)),
						slog.String("patch_speed", humanize.IBytes(bytesPerSec)+"/s"),
					)

					return nil
				},
			},
			{
				Name:  "local-rsync",
				Usage: "perform the full rsync algorithm on local files",
				Flags: []cli.Flag{},
				Action: func(cctx *cli.Context) error {
					src := cctx.Args().Get(0)
					if src == "" {
						return fmt.Errorf("<src> is required")
					}
					orig := cctx.Args().Get(1)
					if orig == "" {
						return fmt.Errorf("<orig> is required")
					}
					patch := cctx.Args().Get(2)
					if patch == "" {
						return fmt.Errorf("<patch> is required")
					}
					dst := cctx.Args().Get(3)
					if dst == "" {
						return fmt.Errorf("<dst> is required")
					}

					ctx, ll, _, err := makeDeps(cctx)
					if err != nil {
						return fmt.Errorf("preparing dependencies: %w", err)
					}

					srcf, err := os.Open(src)
					if err != nil {
						return fmt.Errorf("opening <src> %q: %w", src, err)
					}
					defer srcf.Close()
					origf, err := os.Open(orig)
					if err != nil {
						return fmt.Errorf("opening <orig> %q: %w", orig, err)
					}
					defer origf.Close()
					patchf, err := os.Create(patch)
					if err != nil {
						return fmt.Errorf("creating <patch> %q: %w", patch, err)
					}
					defer patchf.Close()
					dstf, err := os.Create(dst)
					if err != nil {
						return fmt.Errorf("creating <dst> %q: %w", dst, err)
					}
					defer dstf.Close()

					ll.Info("computing file sum")
					sum, err := dirsync.ComputeFileSum(ctx, dstf)
					if err != nil {
						return fmt.Errorf("computing file sum of <dst>: %w", err)
					}

					ll.Info("creating patch")
					enc := patchcodec.NewEncoder(patchf)
					_, err = dirsync.Rsync(ctx, srcf, sum, enc.WriteBlock, enc.WriteBlockID)
					if err != nil {
						return fmt.Errorf("computing patch file from <src> to <dst>: %w", err)
					}

					if err := patchf.Close(); err != nil {
						return fmt.Errorf("flushing patch file <patch>: %w", err)
					} else if patchf, err = os.Open(patch); err != nil {
						return fmt.Errorf("reopening patch file <patch>: %w", err)
					}

					// server side
					ll.Info("applying patch")
					patcher := dirsync.NewFilePatcher(origf, dstf, sum)
					_, err = patchcodec.NewDecoder(patchf).Decode(patcher.WriteBlock, patcher.Copy)
					if err != nil {
						return fmt.Errorf("patching destination <dst>: %w", err)
					}
					if err := dstf.Close(); err != nil {
						return fmt.Errorf("flushing destination <dst>: %w", err)
					}

					ll.Info("patch applied")

					return nil
				},
			},
			{
				Name:  "create-file",
				Usage: "create a file on a remote backend",
				Flags: []cli.Flag{serverSchemeFlag, serverAddrFlag, serverPortFlag, serverPathFlag, blockSizeFlag},
				Action: func(cctx *cli.Context) error {
					base := cctx.Args().Get(0)
					if base == "" {
						return fmt.Errorf("<base> is required")
					}

					path := cctx.Args().Get(1)
					if path == "" {
						return fmt.Errorf("<path> is required")
					}
					if err := os.Chdir(base); err != nil {
						return fmt.Errorf("can't chdir to %q: %w", base, err)
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

					sink, err := syncclient.ClientAdapter(client, meta, blockSize)
					if err != nil {
						return fmt.Errorf("configuring sync service client: %w", err)
					}

					ll.InfoContext(ctx, "creating file", slog.String("path", path))
					f, err := os.Open(path)
					if err != nil {
						return fmt.Errorf("opening file at <path>: %w", err)
					}
					defer f.Close()
					fi, err := f.Stat()
					if err != nil {
						return fmt.Errorf("stating file at <path>: %w", err)
					}

					dir := filepath.Dir(path)
					if dir == "." {
						dir = ""
					}
					dir, err = filepath.Rel(base, dir)
					if err != nil {
						return fmt.Errorf("preparing request: %w", err)
					}

					err = sink.CreateFile(
						ctx,
						typesv1.PathFromString(dir),
						typesv1.FileInfoFromFS(fi),
						f,
					)
					if err != nil {
						return fmt.Errorf("creating file at on remote: %w", err)
					}

					printer.Emit("done")
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
