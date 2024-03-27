package main

import (
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"time"

	"connectrpc.com/connect"
	syncv1 "github.com/aybabtme/syncy/pkg/gen/svc/sync/v1"
	typesv1 "github.com/aybabtme/syncy/pkg/gen/types/v1"
	"github.com/aybabtme/syncy/pkg/logic/dirsync"
	"github.com/aybabtme/syncy/pkg/logic/patchcodec"
	"github.com/aybabtme/syncy/pkg/logic/syncclient"
	"github.com/aybabtme/syncy/pkg/storage/blobdb"
	"github.com/dustin/go-humanize"
	"github.com/urfave/cli"
	"google.golang.org/protobuf/proto"
)

// debugCommands are low level commands made available to help
// debug syncy
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
						slog.Uint64("total_size_bytes", sum.Info.Size),
						slog.String("total_size", humanize.IBytes(sum.Info.Size)),
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

					fi, err := f.Stat()
					if err != nil {
						return fmt.Errorf("stating file %q: %w", path, err)
					}

					sum, err := dirsync.ComputeFileSum(ctx, f, typesv1.FileInfoFromFS(fi))
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
					sinkSum, err := dirsync.ComputeFileSum(ctx, dstf, typesv1.FileInfoFromFS(dstfi))
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

					dstfi, err := dstf.Stat()
					if err != nil {
						return fmt.Errorf("stating <dst> %q: %w", dst, err)
					}

					ll.Info("computing file sum")
					sum, err := dirsync.ComputeFileSum(ctx, dstf, typesv1.FileInfoFromFS(dstfi))
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

					sink, err := syncclient.ClientAdapter(ll.WithGroup("sink"), client, meta, blockSize)
					if err != nil {
						return fmt.Errorf("configuring sync service client: %w", err)
					}

					ll.InfoContext(ctx, "opening local file", slog.String("path", path))
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

					ll.InfoContext(ctx, "creating file on remote", slog.String("path", path))
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
			{
				Name:  "patch-file",
				Usage: "patch a file on a remote backend",
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

					sink, err := syncclient.ClientAdapter(ll.WithGroup("sink"), client, meta, blockSize)
					if err != nil {
						return fmt.Errorf("configuring sync service client: %w", err)
					}

					ll.InfoContext(ctx, "opening local file", slog.String("path", path))
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
					var fileSum *typesv1.FileSum
					if !fi.IsDir() {
						res, err := client.GetFileSum(ctx, connect.NewRequest(&syncv1.GetFileSumRequest{
							Meta: meta, Path: typesv1.PathFromString(path),
						}))
						if err != nil {
							return fmt.Errorf("getting file sum from remote: %w", err)
						}
						fileSum = res.Msg.Sum
					}

					ll.InfoContext(ctx, "creating file on remote", slog.String("path", path))
					err = sink.PatchFile(
						ctx,
						typesv1.PathFromString(dir),
						typesv1.FileInfoFromFS(fi),
						fileSum,
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
