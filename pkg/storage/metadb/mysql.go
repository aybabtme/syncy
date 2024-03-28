package metadb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"
	"time"

	typesv1 "github.com/aybabtme/syncy/pkg/gen/types/v1"
	"github.com/aybabtme/syncy/pkg/logic/dirsync"
	_ "github.com/go-sql-driver/mysql"
	"github.com/noquark/nanoid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Metadata interface {
	CreateAccount(ctx context.Context, accountName string) (accountPublicID string, err error)
	CreateProject(ctx context.Context, accountPublicID, projectName string, createBlobPath func(path string) error) (projectPublicID string, err error)
	Stat(ctx context.Context, accountPublicID, projectPublicID string, path *typesv1.Path) (*typesv1.FileInfo, bool, error)
	ListDir(ctx context.Context, accountPublicID, projectPublicID string, path *typesv1.Path) ([]*typesv1.FileInfo, bool, error)
	GetSignature(ctx context.Context, accountPublicID, projectPublicID string, fn ComputeFileSumAction) (*typesv1.DirSum, error)
	GetFileSum(ctx context.Context, accountPublicID, projectName string, path *typesv1.Path, compute ComputeFileSumAction) (*typesv1.FileSum, bool, error)
	CreatePathTx(ctx context.Context, accountPublicID, projectPublicID string, path *typesv1.Path, fi *typesv1.FileInfo, fn FileSaveAction) error
	PatchPathTx(ctx context.Context, accountPublicID, projectPublicID string, path *typesv1.Path, fi *typesv1.FileInfo, sum *typesv1.FileSum, fn FileSaveAction) error
	DeletePath(ctx context.Context, accountPublicID, projectPublicID string, path *typesv1.Path, fn FileDeleteAction) error
}

type ComputeFileSumAction func(projectDir, filename string, fi *typesv1.FileInfo) (*typesv1.FileSum, bool, error)

type FileSaveAction func(projectDir, filepath string) (blake3_64_256_sum []byte, err error)

type FileDeleteAction func(projectDir, filepath string) error

var _ Metadata = (*MySQL)(nil)

type MySQL struct {
	ll *slog.Logger
	db *sql.DB
}

func NewMySQL(ll *slog.Logger, db *sql.DB) *MySQL {
	return &MySQL{ll: ll, db: db}
}

func (ms *MySQL) CreateAccount(ctx context.Context, accountName string) (accountPublicID string, err error) {
	accountPublicID = nanoid.Must(12)
	_, err = ms.db.ExecContext(ctx, "INSERT INTO accounts (`name`, `public_id`) VALUES (?, ?)",
		accountName, accountPublicID,
	)
	if err != nil {
		return "", fmt.Errorf("inserting account: %w", err)
	}
	return accountPublicID, nil
}

func (ms *MySQL) CreateProject(ctx context.Context, accountPublicID, projectName string, createBlobPath func(path string) error) (projectPublicID string, err error) {
	projectPublicID = nanoid.Must(12)
	return projectPublicID, withTx(ctx, ms.db, func(tx *sql.Tx) error {
		accountInternalID, ok, err := findAccountID(ctx, tx, accountPublicID)
		if err != nil {
			return fmt.Errorf("looking up account: %w", err)
		}
		if !ok {
			return ErrAccountDoesntExist
		}

		_, err = tx.ExecContext(ctx, "INSERT INTO projects (`account_id`, `name`, `public_id`) VALUES (?, ?, ?)",
			accountInternalID, projectName, projectPublicID,
		)
		if err != nil {
			return fmt.Errorf("inserting project: %w", err)
		}

		projectDir := filepath.Join(accountPublicID, projectPublicID)
		return createBlobPath(projectDir)
	})
}

func filepathName(path *typesv1.Path, fi *typesv1.FileInfo) string {
	if fi == nil {
		return typesv1.StringFromPath(path)
	} else {
		return typesv1.StringFromPath(typesv1.PathJoin(path, fi.Name))
	}
}

func (ms *MySQL) Stat(ctx context.Context, accountPublicID, projectPublicID string, path *typesv1.Path) (*typesv1.FileInfo, bool, error) {
	ll := ms.ll.With(
		slog.String("account_pub_id", accountPublicID),
		slog.String("project_pub_id", projectPublicID),
		slog.String("path", typesv1.StringFromPath(path)),
	)
	ll.DebugContext(ctx, "Stat")
	ll.DebugContext(ctx, "finding project DB ID")
	projectID, ok, err := findProjectID(ctx, ms.db, accountPublicID, projectPublicID)
	if err != nil || !ok {
		return nil, false, err
	}
	ll.DebugContext(ctx, "found project", slog.Uint64("project_id", projectID))
	return ms.stat(ctx, ll, projectID, path)
}

func (ms *MySQL) stat(ctx context.Context, ll *slog.Logger, projectID uint64, path *typesv1.Path) (*typesv1.FileInfo, bool, error) {
	if len(path.Elements) == 0 {
		return nil, false, fmt.Errorf("can't stat base dir")
	}
	var parentDirID *uint64
	if dir := typesv1.DirOf(path); len(dir.Elements) > 0 {
		ll.DebugContext(ctx, "searching for parent dir DB ID")
		var (
			ok  bool
			err error
		)
		parentDirID, ok, err = findDirID(ctx, ll, ms.db, projectID, dir)
		if err != nil {
			return nil, false, fmt.Errorf("finding parent dir: %w", err)
		} else if !ok {
			return nil, ok, nil
		}
		ll = ll.With(slog.Uint64("dir_id", *parentDirID))
		ll.DebugContext(ctx, "found parent dir")
	}
	filename := path.Elements[len(path.Elements)-1]
	ll.DebugContext(ctx, "looking for filename in dir", slog.String("filename", filename))
	fi, ok, err := getFileInfo(ctx, ll, ms.db, projectID, parentDirID, filename)
	if err != nil {
		return nil, false, fmt.Errorf("looking up file info: %w", err)
	}
	if !ok {
		// maybe it's a directory
		fi, ok, err = getDirInfo(ctx, ll, ms.db, projectID, parentDirID, filename)
		if err != nil {
			return nil, false, fmt.Errorf("looking up file info: %w", err)
		}
	}
	return fi, ok, nil
}

func (ms *MySQL) ListDir(ctx context.Context, accountPublicID, projectPublicID string, path *typesv1.Path) ([]*typesv1.FileInfo, bool, error) {
	ll := ms.ll.With(
		slog.String("account_pub_id", accountPublicID),
		slog.String("project_pub_id", projectPublicID),
		slog.String("path", typesv1.StringFromPath(path)),
	)
	ll.DebugContext(ctx, "finding project DB ID")
	projectID, ok, err := findProjectID(ctx, ms.db, accountPublicID, projectPublicID)
	if err != nil || !ok {
		return nil, false, fmt.Errorf("finding project dir: %w", err)
	}
	ll.DebugContext(ctx, "found project", slog.Uint64("project_id", projectID))
	return ms.listDir(ctx, ll, projectID, path)
}

func (ms *MySQL) listDir(ctx context.Context, ll *slog.Logger, projectID uint64, path *typesv1.Path) ([]*typesv1.FileInfo, bool, error) {
	var (
		dirID *uint64
		ok    bool
		err   error
	)

	if len(path.Elements) > 0 {
		ll.DebugContext(ctx, "searching for parent dir DB ID")
		dirID, ok, err = findDirID(ctx, ll, ms.db, projectID, path)
		if err != nil {
			return nil, false, fmt.Errorf("finding parent dir: %w", err)
		} else if !ok {
			return nil, ok, nil
		}
		ll = ll.With(slog.Uint64("dir_id", *dirID))
		ll.DebugContext(ctx, "found parent dir")
	}

	ll.DebugContext(ctx, "listing child dirs in dir")
	dirfis, err := listDirs(ctx, ll, ms.db, projectID, dirID)
	if err != nil {
		return nil, false, fmt.Errorf("listing children dir: %w", err)
	}
	ll.DebugContext(ctx, "listing files in dir")
	filefis, err := listFiles(ctx, ll, ms.db, projectID, dirID)
	if err != nil {
		return nil, false, fmt.Errorf("listing files: %w", err)
	}
	all := append(dirfis, filefis...)
	sort.Slice(all, func(i, j int) bool {
		return all[i].Name < all[j].Name
	})
	return all, true, nil // TODO: add pagination
}

func listDirs(ctx context.Context, ll *slog.Logger, querier querier, projectID uint64, dirID *uint64) ([]*typesv1.FileInfo, error) {
	var (
		out  []*typesv1.FileInfo
		rows *sql.Rows
		err  error
	)
	// TODO: CTE to get dir size
	if dirID != nil {
		rows, err = querier.QueryContext(ctx,
			"SELECT "+dirInfoColumns+" FROM dirs\n"+
				"WHERE `project_id` = ? AND\n"+
				"      `parent_id` = ?\n"+
				"LIMIT 10000", // TODO: pagination
			projectID, *dirID,
		)
	} else {
		q := "SELECT " + dirInfoColumns + " FROM dirs\n" +
			"WHERE `project_id` = ? AND\n" +
			"      `parent_id` IS NULL\n" +
			"LIMIT 10000"
		rows, err = querier.QueryContext(ctx, q, projectID)
	}
	if err != nil {
		return nil, fmt.Errorf("querying for dirs in dir: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		fi, _, err := scanDirInfo(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}
		out = append(out, fi)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating rows: %w", err)
	}
	return out, nil
}

func getDirInfo(ctx context.Context, ll *slog.Logger, querier querier, projectID uint64, parentDirID *uint64, dirName string) (*typesv1.FileInfo, bool, error) {
	var (
		q   string
		row *sql.Row
	)
	if parentDirID != nil {
		q = "SELECT " + dirInfoColumns + " FROM dirs\n" +
			"WHERE `project_id` = ? AND\n" +
			"      `parent_id` = ? AND\n" +
			"      `name` = ?\n" +
			"LIMIT 1"
		row = querier.QueryRowContext(ctx, q, projectID, *parentDirID, dirName)
	} else {
		q = "SELECT " + dirInfoColumns + " FROM dirs\n" +
			"WHERE `project_id` = ? AND\n" +
			"      `parent_id` IS NULL AND\n" +
			"      `name` = ?\n" +
			"LIMIT 1"
		row = querier.QueryRowContext(ctx, q, projectID, dirName)
	}
	ll.DebugContext(ctx, "looking up dir", slog.String("query", q))
	return scanDirInfo(row)
}

func listFiles(ctx context.Context, ll *slog.Logger, querier querier, projectID uint64, dirID *uint64) ([]*typesv1.FileInfo, error) {
	var (
		out  []*typesv1.FileInfo
		rows *sql.Rows
		err  error
	)
	if dirID != nil {
		rows, err = querier.QueryContext(ctx,
			"SELECT "+fileInfoColumns+" FROM files\n"+
				"WHERE `project_id` = ? AND\n"+
				"      `dir_id` = ?\n"+
				"LIMIT 10000", // TODO: pagination
			projectID, *dirID,
		)
	} else {
		rows, err = querier.QueryContext(ctx,
			"SELECT "+fileInfoColumns+" FROM files\n"+
				"WHERE `project_id` = ? AND\n"+
				"      `dir_id` IS NULL\n"+
				"LIMIT 10000", // TODO: pagination
			projectID,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("querying for files in dir: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		fi, _, err := scanFileInfo(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}
		out = append(out, fi)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating rows: %w", err)
	}
	return out, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func getFileID(ctx context.Context, querier querier, projectID uint64, parentDirID *uint64, filename string) (uint64, bool, error) {
	var row *sql.Row
	if parentDirID != nil {
		row = querier.QueryRowContext(ctx,
			"SELECT `id` FROM files\n"+
				"WHERE `project_id` = ? AND\n"+
				"      `dir_id` = ? AND\n"+
				"      `name` = ?\n"+
				"LIMIT 1",
			projectID, *parentDirID, filename,
		)
	} else {
		row = querier.QueryRowContext(ctx,
			"SELECT `id` FROM files\n"+
				"WHERE `project_id` = ? AND\n"+
				"      `dir_id` IS NULL AND\n"+
				"      `name` = ?\n"+
				"LIMIT 1",
			projectID, filename,
		)
	}
	var id uint64
	if err := row.Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return 0, false, nil
		}
		return 0, false, err
	}
	return id, true, nil
}

func getFileInfo(ctx context.Context, ll *slog.Logger, querier querier, projectID uint64, parentDirID *uint64, filename string) (*typesv1.FileInfo, bool, error) {
	var (
		q   string
		row *sql.Row
	)
	if parentDirID != nil {
		q = "SELECT " + fileInfoColumns + " FROM files\n" +
			"WHERE `project_id` = ? AND\n" +
			"      `dir_id` = ? AND\n" +
			"      `name` = ?\n" +
			"LIMIT 1"
		row = querier.QueryRowContext(ctx, q, projectID, *parentDirID, filename)
	} else {
		q = "SELECT " + fileInfoColumns + " FROM files\n" +
			"WHERE `project_id` = ? AND\n" +
			"      `dir_id` IS NULL AND\n" +
			"      `name` = ?\n" +
			"LIMIT 1"
		row = querier.QueryRowContext(ctx, q, projectID, filename)
	}
	ll.DebugContext(ctx, "looking up file", slog.String("query", q))
	return scanFileInfo(row)
}

const dirInfoColumns = "`name`, `mod_time_unix_ns`, `mode`"

func scanDirInfo(row scanner) (*typesv1.FileInfo, bool, error) {
	var (
		modTimeUnixNs int64
	)
	fi := new(typesv1.FileInfo)
	if err := row.Scan(
		&fi.Name,
		&modTimeUnixNs,
		&fi.Mode,
	); err == sql.ErrNoRows {
		return nil, false, nil
	} else if err != nil {
		return nil, false, err
	}
	fi.IsDir = true
	fi.ModTime = timestamppb.New(time.Unix(0, modTimeUnixNs))
	return fi, true, nil
}

const fileInfoColumns = "`name`, `size`, `mod_time_unix_ns`, `mode`"

func scanFileInfo(row scanner) (*typesv1.FileInfo, bool, error) {
	var (
		modTimeUnixNs int64
	)
	fi := new(typesv1.FileInfo)
	if err := row.Scan(
		&fi.Name,
		&fi.Size,
		&modTimeUnixNs,
		&fi.Mode,
	); err == sql.ErrNoRows {
		return nil, false, nil
	} else if err != nil {
		return nil, false, err
	}
	fi.IsDir = false
	fi.ModTime = timestamppb.New(time.Unix(0, modTimeUnixNs))
	return fi, true, nil
}

func (ms *MySQL) GetSignature(ctx context.Context, accountPublicID, projectPublicID string, fn ComputeFileSumAction) (*typesv1.DirSum, error) {
	ll := ms.ll.With(
		slog.String("account_pub_id", accountPublicID),
		slog.String("project_pub_id", projectPublicID),
	)
	ll.DebugContext(ctx, "GetSignature")
	projectID, ok, err := findProjectID(ctx, ms.db, accountPublicID, projectPublicID)
	if err != nil {
		return nil, fmt.Errorf("finding project ID: %w", err)
	}
	if !ok {
		return nil, ErrProjectDoesntExist
	}
	ll.DebugContext(ctx, "found project", slog.Uint64("project_id", projectID))
	projectDir := filepath.Join(accountPublicID, projectPublicID)

	sink := &traceSinkAdapter{ms: ms, ll: ll, projectID: projectID, projectDir: projectDir, fn: fn}
	sum, err := dirsync.TraceSink(ctx, projectDir, sink)
	if err != nil {
		return nil, fmt.Errorf("computing sum: %v", err)
	}
	return sum, nil
}

type traceSinkAdapter struct {
	ms         *MySQL
	ll         *slog.Logger
	projectID  uint64
	projectDir string
	fn         ComputeFileSumAction
}

func (tsa *traceSinkAdapter) Stat(ctx context.Context, _ string, name string) (*typesv1.FileInfo, bool, error) {
	return tsa.ms.stat(ctx, tsa.ll, tsa.projectID, typesv1.PathFromString(name))
}

func (tsa *traceSinkAdapter) ListDir(ctx context.Context, _ string, name string) ([]*typesv1.FileInfo, bool, error) {
	return tsa.ms.listDir(ctx, tsa.ll, tsa.projectID, typesv1.PathFromString(name))
}

func (tsa *traceSinkAdapter) GetFileSum(ctx context.Context, _ string, path string, fi *typesv1.FileInfo) (*typesv1.FileSum, bool, error) {
	filepath := typesv1.PathFromString(path)
	return tsa.ms.getFileSum(ctx, tsa.ll, tsa.projectDir, filepath, fi, tsa.fn)
}

func (ms *MySQL) GetFileSum(ctx context.Context, accountPublicID, projectPublicID string, path *typesv1.Path, fn ComputeFileSumAction) (*typesv1.FileSum, bool, error) {
	ll := ms.ll.With(
		slog.String("account_pub_id", accountPublicID),
		slog.String("project_pub_id", projectPublicID),
		slog.String("path", typesv1.StringFromPath(path)),
	)
	ll.DebugContext(ctx, "GetFileSum")
	ll.DebugContext(ctx, "finding project DB ID")
	projectID, ok, err := findProjectID(ctx, ms.db, accountPublicID, projectPublicID)
	if err != nil {
		return nil, false, fmt.Errorf("finding project ID: %w", err)
	}
	if !ok {
		return nil, false, ErrProjectDoesntExist
	}
	ll.DebugContext(ctx, "found project", slog.Uint64("project_id", projectID))
	projectDir := filepath.Join(accountPublicID, projectPublicID)
	ll.DebugContext(ctx, "looking for file")
	fi, ok, err := ms.stat(ctx, ll, projectID, path)
	if err != nil {
		return nil, ok, fmt.Errorf("stating file: %w", err)
	}
	if !ok {
		return nil, false, nil
	}
	return ms.getFileSum(ctx, ll, projectDir, path, fi, fn)
}

func (ms *MySQL) getFileSum(ctx context.Context, ll *slog.Logger, projectDir string, path *typesv1.Path, fi *typesv1.FileInfo, fn ComputeFileSumAction) (*typesv1.FileSum, bool, error) {

	filepath := filepathName(path, fi)
	ll.DebugContext(ctx, "file found, computing filesum", slog.String("filepath", filepath))
	sum, ok, err := fn(projectDir, filepath, fi)
	if err != nil {
		return nil, ok, fmt.Errorf("computing filesum: %w", err)
	}
	if !ok {
		ll.ErrorContext(ctx, "internal inconsistency, file doesn't exist", slog.String("filepath", filepath))
		return nil, false, fmt.Errorf("computing filesum (internal): %w", err)
	}
	return sum, true, nil
}

func (ms *MySQL) CreatePathTx(ctx context.Context, accountPublicID, projectPublicID string, path *typesv1.Path, fi *typesv1.FileInfo, fn FileSaveAction) error {
	ll := ms.ll.With(
		slog.String("account_pub_id", accountPublicID),
		slog.String("project_pub_id", projectPublicID),
		slog.String("path", typesv1.StringFromPath(path)),
	)
	ll.DebugContext(ctx, "CreatePathTx")
	projectID, ok, err := findProjectID(ctx, ms.db, accountPublicID, projectPublicID)
	if err != nil {
		return fmt.Errorf("finding project ID: %w", err)
	}
	if !ok {
		return ErrProjectDoesntExist
	}

	projectDir := filepath.Join(accountPublicID, projectPublicID)
	filepath := filepathName(path, fi)

	var pendingFileID uint64
	err = withTx(ctx, ms.db, func(tx *sql.Tx) error {
		var parentDirID *uint64
		if len(path.Elements) > 0 {
			id, ok, err := findDirID(ctx, ll, tx, projectID, path)
			if err != nil {
				return fmt.Errorf("finding dir %q, %w", typesv1.StringFromPath(path), err)
			}
			if !ok {
				return ErrParentDirDoesntExist
			}
			parentDirID = id
		}
		if fi.IsDir {
			_, err = createDir(ctx, tx, projectID, parentDirID, fi.Name, fi)
			if err != nil {
				return fmt.Errorf("creating dir in mysql: %w", err)
			}
			_, err := fn(projectDir, filepath)
			if err != nil {
				return fmt.Errorf("creating filepath in blob: %w", err)
			}
			return nil
		}

		pendingFileID, err = createPendingFile(ctx, tx, projectID, parentDirID, fi.Name, fi)
		if err != nil {
			return fmt.Errorf("creating pending file entry: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if fi.IsDir {
		return nil // we're done
	}
	sum, err := fn(projectDir, filepath)
	if err != nil {
		return fmt.Errorf("writing file in blob: %w", err)
	}

	if err := finishPendingFile(ctx, ms.db, pendingFileID, sum); err != nil {
		return fmt.Errorf("finishing pending file: %w", err)
	}
	return nil
}

func (ms *MySQL) PatchPathTx(ctx context.Context, accountPublicID, projectPublicID string, path *typesv1.Path, fi *typesv1.FileInfo, sum *typesv1.FileSum, fn FileSaveAction) error {
	ll := ms.ll.With(
		slog.String("account_pub_id", accountPublicID),
		slog.String("project_pub_id", projectPublicID),
		slog.String("path", typesv1.StringFromPath(path)),
	)
	ll.DebugContext(ctx, "PatchPathTx")
	projectID, ok, err := findProjectID(ctx, ms.db, accountPublicID, projectPublicID)
	if err != nil {
		return fmt.Errorf("finding project ID: %w", err)
	}
	if !ok {
		return ErrProjectDoesntExist
	}

	projectDir := filepath.Join(accountPublicID, projectPublicID)
	filepath := filepathName(path, fi)

	var pendingFileID uint64
	err = withTx(ctx, ms.db, func(tx *sql.Tx) error {
		var parentDirID *uint64
		if len(path.Elements) > 0 {
			id, ok, err := findDirID(ctx, ll, tx, projectID, path)
			if err != nil {
				return fmt.Errorf("finding dir %q, %w", typesv1.StringFromPath(path), err)
			}
			if !ok {
				return ErrParentDirDoesntExist
			}
			parentDirID = id
		}
		if fi.IsDir {
			// we just update metadata so there's no need to mark anything pending
			_, err := fn(projectDir, filepath)
			if err != nil {
				return fmt.Errorf("updating filepath in blob: 5w")
			}
			err = updateDirInfo(ctx, tx, projectID, parentDirID, fi.Name, fi)
			if err != nil {
				return fmt.Errorf("updating dir info in mysql: %w", err)
			}
			return nil
		}

		fileID, ok, err := getFileID(ctx, tx, projectID, parentDirID, fi.Name)
		if err != nil {
			return fmt.Errorf("looking up file: %w", err)
		}
		if !ok {
			return fmt.Errorf("file doesn't exist, cannot be patched")
		}

		pendingFileID, err = markFileAsPending(ctx, tx, projectID, fileID, fi)
		if err != nil {
			return fmt.Errorf("creating pending file entry: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if fi.IsDir {
		return nil // we're done
	}
	blake3sum, err := fn(projectDir, filepath)
	if err != nil {
		return fmt.Errorf("creating file in blob: %w", err)
	}

	err = finishPendingPatchFile(ctx, ms.db, pendingFileID, fi, blake3sum)
	if err != nil {
		return fmt.Errorf("finishing pending file: %w", err)
	}
	return nil
}

func (ms *MySQL) DeletePath(ctx context.Context, accountPublicID, projectPublicID string, path *typesv1.Path, fn FileDeleteAction) error {
	ll := ms.ll.With(
		slog.String("account_pub_id", accountPublicID),
		slog.String("project_pub_id", projectPublicID),
	)
	ll.DebugContext(ctx, "DeletePath")
	projectID, ok, err := findProjectID(ctx, ms.db, accountPublicID, projectPublicID)
	if err != nil {
		return fmt.Errorf("finding project ID: %w", err)
	}
	if !ok {
		return ErrProjectDoesntExist
	}

	projectDir := filepath.Join(accountPublicID, projectPublicID)
	filepath := filepathName(path, nil)
	err = withTx(ctx, ms.db, func(tx *sql.Tx) error {
		err := deletePath(ctx, ll, tx, projectID, path)
		if err != nil {
			return fmt.Errorf("deleting path in metadata: %w", err)
		}
		err = fn(projectDir, filepath)
		if err != nil {
			return fmt.Errorf("deleting from blobs: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("deleting path %q: %w", filepath, err)
	}
	return nil
}

func deletePath(ctx context.Context, ll *slog.Logger, execer execer, projectID uint64, path *typesv1.Path) error {
	n := len(path.Elements)
	filename := path.Elements[n-1]
	parentDirID, err := findParentDir(ctx, ll, execer, projectID, path)
	if err != nil {
		return fmt.Errorf("looking up file's parent dir: %w", err)
	}
	if parentDirID != nil {
		_, err = execer.ExecContext(ctx, "DELETE FROM files WHERE `project_id` = ? AND `parent_dir` = ? AND `name` = ?",
			projectID, parentDirID, filename,
		)
	} else {
		_, err = execer.ExecContext(ctx, "DELETE FROM files WHERE `project_id` = ? AND `parent_dir` IS NULL AND `name` = ?",
			projectID, filename,
		)
	}
	if err != nil {
		return fmt.Errorf("execing query: %w", err)
	}
	return nil
}

func findParentDir(ctx context.Context, ll *slog.Logger, execer execer, projectID uint64, path *typesv1.Path) (*uint64, error) {
	var parentDirID *uint64
	if dir := typesv1.DirOf(path); len(dir.Elements) > 0 {
		ll.DebugContext(ctx, "searching for parent dir DB ID")
		var (
			ok  bool
			err error
		)
		parentDirID, ok, err = findDirID(ctx, ll, execer, projectID, dir)
		if err != nil {
			return nil, fmt.Errorf("finding parent dir: %w", err)
		} else if !ok {
			return nil, fmt.Errorf("no such dir: %w", err)
		}
		ll = ll.With(slog.Uint64("dir_id", *parentDirID))
		ll.DebugContext(ctx, "found parent dir")
	}
	return parentDirID, nil
}

func createPendingFile(ctx context.Context, execer execer, projectID uint64, parentDirID *uint64, name string, fi *typesv1.FileInfo) (uint64, error) {
	var (
		res sql.Result
		err error
	)
	if parentDirID != nil {
		res, err = execer.ExecContext(ctx,
			"INSERT INTO files (`project_id`, `dir_id`, `name`, `size`, `mod_time_unix_ns`, `mode`) VALUES (?,?,?,?,?,?)",
			projectID,
			parentDirID,
			name,
			fi.Size,
			fi.ModTime.AsTime().UnixNano(),
			fi.Mode,
		)
	} else {
		res, err = execer.ExecContext(ctx,
			"INSERT INTO files (`project_id`, `name`, `size`, `mod_time_unix_ns`, `mode`) VALUES (?,?,?,?,?)",
			projectID,
			name,
			fi.Size,
			fi.ModTime.AsTime().UnixNano(),
			fi.Mode,
		)
	}
	if err != nil {
		return 0, fmt.Errorf("inserting file: %w", err)
	}
	fileID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("getting file ID: %w", err)
	}
	res, err = execer.ExecContext(ctx,
		"INSERT INTO pending_files (file_id) VALUES (?)",
		fileID,
	)
	if err != nil {
		return 0, fmt.Errorf("inserting pending file: %w", err)
	}
	_ = res
	return uint64(fileID), nil
}

func markFileAsPending(ctx context.Context, execer execer, projectID uint64, fileID uint64, fi *typesv1.FileInfo) (uint64, error) {
	res, err := execer.ExecContext(ctx,
		"INSERT INTO pending_files (`file_id`) VALUES (?)",
		fileID,
	)
	if err != nil {
		return 0, fmt.Errorf("inserting pending file: %w", err)
	}
	_ = res
	return uint64(fileID), nil
}

func finishPendingFile(ctx context.Context, db *sql.DB, pendingFileID uint64, blake3_64_256_sum []byte) error {
	return withTx(ctx, db, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, "UPDATE files SET blake3_64_256_sum=? WHERE id = ? LIMIT 1", blake3_64_256_sum, pendingFileID)
		if err != nil {
			return fmt.Errorf("updating file with blake3_64_256_sum (len=%d): %w", len(blake3_64_256_sum), err)
		}
		_, err = tx.ExecContext(ctx, "DELETE FROM pending_files WHERE file_id = ? LIMIT 1", pendingFileID)
		if err != nil {
			return fmt.Errorf("deleting pending file entry: %w", err)
		}
		return nil
	})
}

func finishPendingPatchFile(ctx context.Context, db *sql.DB, pendingFileID uint64, fi *typesv1.FileInfo, blake3_64_256_sum []byte) error {
	return withTx(ctx, db, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx,
			"UPDATE files\n"+
				"SET\n"+
				"	`size` = ?,\n"+
				"	`mod_time_unix_ns` = ?,\n"+
				"	`mode` = ?,\n"+
				"	`blake3_64_256_sum` = ?\n"+
				"WHERE id = ? LIMIT 1",
			fi.Size,
			fi.ModTime.AsTime().UnixNano(),
			fi.Mode,
			blake3_64_256_sum,
			pendingFileID,
		)
		if err != nil {
			return fmt.Errorf("updating file with blake3_64_256_sum: %w", err)
		}
		_, err = tx.ExecContext(ctx, "DELETE FROM pending_files WHERE file_id = ? LIMIT 1", pendingFileID)
		if err != nil {
			return fmt.Errorf("deleting pending file entry: %w", err)
		}
		return nil
	})
}

func updateDirInfo(ctx context.Context, execer execer, projectID uint64, parentDirID *uint64, name string, fi *typesv1.FileInfo) error {
	var err error
	if parentDirID != nil {
		_, err = execer.ExecContext(ctx,
			"UPDATE dirs\n"+
				"SET\n"+
				"	name=?,\n"+
				"	mod_time=?,\n"+
				"	mode=?\n"+
				"WHERE `project_id` = ? AND\n"+
				"	`parent_id` = ? AND\n"+
				"	`name` = ?\n"+
				"LIMIT 1",
			fi.Name, fi.ModTime.AsTime().Unix(), fi.Mode,
			projectID, *parentDirID, name,
		)
	} else {
		_, err = execer.ExecContext(ctx,
			"UPDATE dirs\n"+
				"SET\n"+
				"	name=?,\n"+
				"	mod_time=?,\n"+
				"	mode=?\n"+
				"WHERE `project_id` = ? AND\n"+
				"	`parent_id` IS NULL AND\n"+
				"	`name` = ?\n"+
				"LIMIT 1",
			fi.Name, fi.ModTime.AsTime().Unix(), fi.Mode,
			projectID, name,
		)
	}
	return err
}

func findAccountID(ctx context.Context, querier querier, accountPublicID string) (uint64, bool, error) {
	var accountID uint64
	err := querier.QueryRowContext(ctx,
		"SELECT accounts.`id` FROM accounts\n"+
			"WHERE accounts.`public_id` = ?\n"+
			"LIMIT 1",
		accountPublicID,
	).Scan(&accountID)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	return accountID, true, nil
}

func findProjectID(ctx context.Context, querier querier, accountPublicID, projectPublicID string) (uint64, bool, error) {
	var projectID uint64
	err := querier.QueryRowContext(ctx,
		"SELECT projects.`id` FROM projects\n"+
			"JOIN accounts ON (accounts.`id` = projects.`account_id`)\n"+
			"WHERE accounts.`public_id` = ? AND\n"+
			"      projects.`public_id` = ?\n"+
			"LIMIT 1",
		accountPublicID,
		projectPublicID,
	).Scan(&projectID)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	return projectID, true, err
}

type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	querier
}

type querier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func createDir(ctx context.Context,
	execer execer,
	projectID uint64,
	parentDirID *uint64,
	name string,
	fi *typesv1.FileInfo,
) (uint64, error) {
	var (
		res sql.Result
		err error
	)
	if parentDirID != nil {

		res, err = execer.ExecContext(ctx,
			"INSERT INTO dirs (`project_id`, `parent_id`, `name`, `mod_time_unix_ns`, `mode`) VALUES (?, ?, ?, ?, ?)",
			projectID,
			parentDirID,
			name,
			fi.ModTime.AsTime().UnixNano(),
			fi.Mode,
		)
	} else {
		res, err = execer.ExecContext(ctx,
			"INSERT INTO dirs (`project_id`, `name`, `mod_time_unix_ns`, `mode`) VALUES (?, ?, ?, ?)",
			projectID,
			name,
			fi.ModTime.AsTime().UnixNano(),
			fi.Mode,
		)
	}
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return uint64(id), err
}

func findDirID(
	ctx context.Context,
	ll *slog.Logger,
	querier querier,
	projectID uint64,
	path *typesv1.Path,
) (*uint64, bool, error) {
	if len(path.Elements) == 0 {
		return nil, true, nil
	}
	var dirID uint64
	err := querier.QueryRowContext(ctx,
		"SELECT id FROM dirs WHERE `project_id` = ? AND `parent_id` IS NULL AND `name` = ? LIMIT 1",
		projectID,
		path.Elements[0],
	).Scan(&dirID)
	ll.DebugContext(ctx, "dirID", slog.Uint64("dirID", dirID))
	if err == sql.ErrNoRows {
		ll.DebugContext(ctx, "no top level dir found with name", slog.String("name", path.Elements[0]))
		return nil, false, nil
	} else if err != nil {
		return nil, false, fmt.Errorf("finding first element of path: %w", err)
	}
	if len(path.Elements) == 1 {
		ll.DebugContext(ctx, "found it at root", slog.Uint64("dirID", dirID))
		return &dirID, true, nil
	}
	for _, elem := range path.Elements[1:] {
		err = querier.QueryRowContext(ctx,
			"SELECT id FROM dirs WHERE `project_id` = ? AND `parent_id` = ? AND `name` = ? LIMIT 1",
			projectID,
			dirID,
			elem,
		).Scan(&dirID)
		if err == sql.ErrNoRows {
			ll.DebugContext(ctx, "no dir found with name", slog.String("name", elem))
			return nil, false, nil
		} else if err != nil {
			return nil, false, fmt.Errorf("finding %q: %w", strings.Join(path.Elements, "/"), err)
		}
		ll.DebugContext(ctx, "iterating", slog.Uint64("dirID", dirID))
	}
	ll.DebugContext(ctx, "done iterating", slog.Uint64("dirID", dirID))
	return &dirID, true, nil
}

func withTx(ctx context.Context, db *sql.DB, fn func(tx *sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("starting transaction: %w", err)
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commiting transaction: %w", err)
	}
	return nil
}
