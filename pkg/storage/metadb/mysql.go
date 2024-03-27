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
	_ "github.com/go-sql-driver/mysql"
	"github.com/noquark/nanoid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Metadata interface {
	CreateAccount(ctx context.Context, accountName string) (accountPublicID string, err error)
	CreateProject(ctx context.Context, accountPublicID, projectName string, createBlobPath func(path string) error) (projectPublicID string, err error)
	Stat(ctx context.Context, accountPublicID, projectPublicID string, path *typesv1.Path) (*typesv1.FileInfo, bool, error)
	ListDir(ctx context.Context, accountPublicID, projectPublicID string, path *typesv1.Path) ([]*typesv1.FileInfo, bool, error)
	GetSignature(ctx context.Context, accountPublicID, projectPublicID string, fn ComputeDirSumAction) (*typesv1.DirSum, error)
	GetFileSum(ctx context.Context, accountPublicID, projectName string, path *typesv1.Path, compute ComputeFileSumAction) (*typesv1.FileSum, bool, error)
	CreatePathTx(ctx context.Context, accountPublicID, projectPublicID string, path *typesv1.Path, fi *typesv1.FileInfo, fn FileSaveAction) error
	PatchPathTx(ctx context.Context, accountPublicID, projectPublicID string, path *typesv1.Path, fi *typesv1.FileInfo, sum *typesv1.FileSum, fn FileSaveAction) error
	DeletePaths(ctx context.Context, accountPublicID, projectPublicID string, paths []*typesv1.Path, fn FileDeleteAction) error
}

type ComputeDirSumAction func(projectDir string) (*typesv1.DirSum, error)

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
	ll.InfoContext(ctx, "Stat")
	ll.InfoContext(ctx, "finding project DB ID")
	projectID, ok, err := findProjectID(ctx, ms.db, accountPublicID, projectPublicID)
	if err != nil || !ok {
		return nil, false, err
	}
	ll.InfoContext(ctx, "found project", slog.Uint64("project_id", projectID))
	return ms.stat(ctx, ll, projectID, path)
}

func (ms *MySQL) stat(ctx context.Context, ll *slog.Logger, projectID uint64, path *typesv1.Path) (*typesv1.FileInfo, bool, error) {
	if len(path.Elements) == 0 {
		return nil, false, fmt.Errorf("can't stat a dir, use `listDir` instead")
	}
	var parentDirID *uint64
	if dir := typesv1.DirOf(path); len(dir.Elements) > 0 {
		ll.InfoContext(ctx, "searching for parent dir DB ID")
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
		ll.InfoContext(ctx, "found parent dir")
	}
	filename := path.Elements[len(path.Elements)-1]
	ll.InfoContext(ctx, "looking for filename in dir", slog.String("filename", filename))
	return getFileInfo(ctx, ll, ms.db, projectID, parentDirID, filename)
}

func (ms *MySQL) ListDir(ctx context.Context, accountPublicID, projectPublicID string, path *typesv1.Path) ([]*typesv1.FileInfo, bool, error) {
	ll := ms.ll.With(
		slog.String("account_pub_id", accountPublicID),
		slog.String("project_pub_id", projectPublicID),
		slog.String("path", typesv1.StringFromPath(path)),
	)
	ll.InfoContext(ctx, "finding project DB ID")
	projectID, ok, err := findProjectID(ctx, ms.db, accountPublicID, projectPublicID)
	if err != nil || !ok {
		return nil, false, fmt.Errorf("finding project dir: %w", err)
	}
	ll.InfoContext(ctx, "found project", slog.Uint64("project_id", projectID))
	var dirID *uint64

	if len(path.Elements) > 0 {
		ll.InfoContext(ctx, "searching for parent dir DB ID")
		dirID, ok, err = findDirID(ctx, ll, ms.db, projectID, path)
		if err != nil {
			return nil, false, fmt.Errorf("finding parent dir: %w", err)
		} else if !ok {
			return nil, ok, nil
		}
		ll = ll.With(slog.Uint64("dir_id", *dirID))
		ll.InfoContext(ctx, "found parent dir")
	}

	ll.InfoContext(ctx, "listing child dirs in dir")
	dirfis, err := listDirs(ctx, ll, ms.db, projectID, dirID)
	if err != nil {
		return nil, false, fmt.Errorf("listing children dir: %w", err)
	}
	ll.InfoContext(ctx, "listing files in dir")
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
	ll.InfoContext(ctx, "looking up file", slog.String("query", q))
	return scanFileInfo(row)
}

const dirInfoColumns = "`name`, `mod_time`, `mode`"

func scanDirInfo(row scanner) (*typesv1.FileInfo, bool, error) {
	var modTimeUnix int64
	fi := new(typesv1.FileInfo)
	if err := row.Scan(
		&fi.Name,
		&modTimeUnix,
		&fi.Mode,
	); err == sql.ErrNoRows {
		return nil, false, nil
	} else if err != nil {
		return nil, false, err
	}
	fi.IsDir = true
	fi.ModTime = timestamppb.New(time.Unix(modTimeUnix, 0))
	return fi, true, nil
}

const fileInfoColumns = "`name`, `size`, `mod_time`, `mode`"

func scanFileInfo(row scanner) (*typesv1.FileInfo, bool, error) {
	var modTimeUnix int64
	fi := new(typesv1.FileInfo)
	if err := row.Scan(
		&fi.Name,
		&fi.Size,
		&modTimeUnix,
		&fi.Mode,
	); err == sql.ErrNoRows {
		return nil, false, nil
	} else if err != nil {
		return nil, false, err
	}
	fi.IsDir = false
	fi.ModTime = timestamppb.New(time.Unix(modTimeUnix, 0))
	return fi, true, nil
}

func (ms *MySQL) GetSignature(ctx context.Context, accountPublicID, projectPublicID string, fn ComputeDirSumAction) (*typesv1.DirSum, error) {
	ll := ms.ll.With(
		slog.String("account_pub_id", accountPublicID),
		slog.String("project_pub_id", projectPublicID),
	)
	ll.InfoContext(ctx, "GetSignature")
	projectID, ok, err := findProjectID(ctx, ms.db, accountPublicID, projectPublicID)
	if err != nil {
		return nil, fmt.Errorf("finding project ID: %w", err)
	}
	if !ok {
		return nil, ErrProjectDoesntExist
	}
	ll.InfoContext(ctx, "found project", slog.Uint64("project_id", projectID))
	projectDir := filepath.Join(accountPublicID, projectPublicID)
	sum, err := fn(projectDir)
	if err != nil {
		return nil, fmt.Errorf("computing sum: %v", err)
	}
	return sum, nil
}

func (ms *MySQL) GetFileSum(ctx context.Context, accountPublicID, projectPublicID string, path *typesv1.Path, fn ComputeFileSumAction) (*typesv1.FileSum, bool, error) {
	ll := ms.ll.With(
		slog.String("account_pub_id", accountPublicID),
		slog.String("project_pub_id", projectPublicID),
		slog.String("path", typesv1.StringFromPath(path)),
	)
	ll.InfoContext(ctx, "GetFileSum")
	ll.InfoContext(ctx, "finding project DB ID")
	projectID, ok, err := findProjectID(ctx, ms.db, accountPublicID, projectPublicID)
	if err != nil {
		return nil, false, fmt.Errorf("finding project ID: %w", err)
	}
	if !ok {
		return nil, false, ErrProjectDoesntExist
	}
	ll.InfoContext(ctx, "found project", slog.Uint64("project_id", projectID))

	ll.InfoContext(ctx, "looking for file")
	fi, ok, err := ms.stat(ctx, ll, projectID, path)
	if err != nil {
		return nil, ok, fmt.Errorf("stating file: %w", err)
	}
	if !ok {
		return nil, false, nil
	}
	projectDir := filepath.Join(accountPublicID, projectPublicID)
	filepath := filepathName(path, nil)
	ll.InfoContext(ctx, "file found, computing filesum", slog.String("filepath", filepath))
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
	ll.InfoContext(ctx, "CreatePathTx")
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
				return fmt.Errorf("parent dir %q doesn't exist, you must create it first", typesv1.StringFromPath(path))
			}
			parentDirID = id
		}
		if fi.IsDir {
			_, err := fn(projectDir, filepath)
			if err != nil {
				return fmt.Errorf("creating filepath in blob: %w", err)
			}
			_, err = createDir(ctx, tx, projectID, parentDirID, fi.Name, fi)
			if err != nil {
				return fmt.Errorf("creating dir in mysql: %w", err)
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
		return fmt.Errorf("in transaction: %w", err)
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
	ll.InfoContext(ctx, "PatchPathTx")
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
				return fmt.Errorf("parent dir %q doesn't exist, you must create it first", typesv1.StringFromPath(path))
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
		return fmt.Errorf("in transaction: %w", err)
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

func (ms *MySQL) DeletePaths(ctx context.Context, accountPublicID, projectPublicID string, paths []*typesv1.Path, fn FileDeleteAction) error {
	ll := ms.ll.With(
		slog.String("account_pub_id", accountPublicID),
		slog.String("project_pub_id", projectPublicID),
	)
	ll.InfoContext(ctx, "DeletePaths")
	projectID, ok, err := findProjectID(ctx, ms.db, accountPublicID, projectPublicID)
	if err != nil {
		return fmt.Errorf("finding project ID: %w", err)
	}
	if !ok {
		return ErrProjectDoesntExist
	}

	projectDir := filepath.Join(accountPublicID, projectPublicID)
	for _, path := range paths {
		path := path
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
		ll.InfoContext(ctx, "searching for parent dir DB ID")
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
		ll.InfoContext(ctx, "found parent dir")
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
			"INSERT INTO files (`project_id`, `dir_id`, `name`, `size`, `mod_time`, `mode`) VALUES (?,?,?,?,?,?)",
			projectID,
			parentDirID,
			name,
			fi.Size,
			fi.ModTime.AsTime().Unix(),
			fi.Mode,
		)
	} else {
		res, err = execer.ExecContext(ctx,
			"INSERT INTO files (`project_id`, `name`, `size`, `mod_time`, `mode`) VALUES (?,?,?,?,?)",
			projectID,
			name,
			fi.Size,
			fi.ModTime.AsTime().Unix(),
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
				"	`mod_time` = ?,\n"+
				"	`mode` = ?,\n"+
				"	`blake3_64_256_sum` = ?\n"+
				"WHERE id = ? LIMIT 1",
			fi.Size,
			fi.ModTime.AsTime().Unix(),
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
			"INSERT INTO dirs (`project_id`, `parent_id`, `name`, `mod_time`, `mode`) VALUES (?, ?, ?, ?, ?)",
			projectID,
			parentDirID,
			name,
			fi.ModTime.AsTime().Unix(),
			fi.Mode,
		)
	} else {
		res, err = execer.ExecContext(ctx,
			"INSERT INTO dirs (`project_id`, `name`, `mod_time`, `mode`) VALUES (?, ?, ?, ?)",
			projectID,
			name,
			fi.ModTime.AsTime().Unix(),
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
	ll.InfoContext(ctx, "dirID", slog.Uint64("dirID", dirID))
	if err == sql.ErrNoRows {
		ll.InfoContext(ctx, "no top level dir found with name", slog.String("name", path.Elements[0]))
		return nil, false, nil
	} else if err != nil {
		return nil, false, fmt.Errorf("finding first element of path: %w", err)
	}
	if len(path.Elements) == 1 {
		ll.InfoContext(ctx, "found it at root", slog.Uint64("dirID", dirID))
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
			ll.InfoContext(ctx, "no dir found with name", slog.String("name", elem))
			return nil, false, nil
		} else if err != nil {
			return nil, false, fmt.Errorf("finding %q: %w", strings.Join(path.Elements, "/"), err)
		}
		ll.InfoContext(ctx, "iterating", slog.Uint64("dirID", dirID))
	}
	ll.InfoContext(ctx, "done iterating", slog.Uint64("dirID", dirID))
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
