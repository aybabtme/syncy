package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/pprof"
	"os"

	"github.com/aybabtme/syncy/pkg/gen/svc/sync/v1/syncv1connect"
	"github.com/aybabtme/syncy/pkg/storage"
	"github.com/aybabtme/syncy/pkg/storage/blobdb"
	"github.com/aybabtme/syncy/pkg/storage/metadb"
	"github.com/aybabtme/syncy/pkg/svc/syncsvc"
)

func main() {
	var (
		listenIface      = flag.String("listen.iface", "127.0.0.1", "interface to listen on")
		listenPort       = flag.String("listen.port", "7071", "port to listen on")
		metadbMySQLAddr  = flag.String("metadb.mysql.addr", "root@tcp(127.0.0.1:3306)/syncy", "")
		blobLocalPath    = flag.String("blob.local.path", "tmp/blobs", "")
		scratchLocalPath = flag.String("scratch.local.path", "/tmp/blobs", "")
	)
	flag.Parse()

	slogOpts := &slog.HandlerOptions{}
	ll := slog.New(slog.NewJSONHandler(os.Stderr, slogOpts))
	if err := realMain(ll,
		*listenIface,
		*listenPort,
		*metadbMySQLAddr,
		*blobLocalPath,
		*scratchLocalPath,
	); err != nil {
		ll.Error("program failed", slog.Any("error", err))
		os.Exit(1)
	}
}

func realMain(
	ll *slog.Logger,
	listenIface, listenPort string,
	metadbMySQLAddr string,
	blobLocalPath string,
	scratchLocalPath string,
) error {
	var (
		meta metadb.Metadata
		blob blobdb.Blob
		err  error
	)
	if metadbMySQLAddr != "" {
		ll.Info("using MySQL for metadata", slog.String("dsn", metadbMySQLAddr))
		db, err := sql.Open("mysql", metadbMySQLAddr)
		if err != nil {
			return fmt.Errorf("opening mysql for metadata DB: %w", err)
		}
		defer db.Close()
		meta = metadb.NewMySQL(ll.WithGroup("mysql"), db)
	}
	if meta == nil {
		return fmt.Errorf("no metadata backend provided")
	}
	if blobLocalPath != "" {
		if scratchLocalPath == "" {
			scratchLocalPath = "/tmp/blobs"
		}
		ll.Info("using LocalFS for blobs",
			slog.String("path", blobLocalPath),
			slog.String("scratch", scratchLocalPath),
		)
		blob, err = blobdb.NewLocalFS(blobLocalPath, scratchLocalPath)
	}
	if err != nil {
		return fmt.Errorf("creating blob backend: %w", err)
	}
	if blob == nil {
		return fmt.Errorf("no blob backend provided")
	}

	l, err := net.Listen("tcp", net.JoinHostPort(listenIface, listenPort))
	if err != nil {
		return fmt.Errorf("creating network listener: %w", err)
	}
	defer l.Close()

	// grab back the actual address, because ambiguous requests like "0.0.0.0:0"
	// will result in an IP and port assignment that's not the same as what was
	// requested
	addr := l.Addr().(*net.TCPAddr)
	ll.Info("preparing listener",
		slog.String("listen.iface", addr.IP.String()),
		slog.Int("listen.port", addr.Port),
	)

	state := storage.NewState(meta, blob)

	syncsvcPath, synchdl := syncv1connect.NewSyncServiceHandler(
		syncsvc.NewHandler(ll.WithGroup("syncsvc"), state),
	)

	mux := http.NewServeMux()
	mux.Handle(syncsvcPath, synchdl)

	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	srv := http.Server{
		Handler: mux,
		// TODO: all the read/write settings, tls, etc
	}
	ll.Info("ready to serve requests")
	if err := srv.Serve(l); err != nil {
		return fmt.Errorf("serving: %w", err)
	}
	return nil
}
